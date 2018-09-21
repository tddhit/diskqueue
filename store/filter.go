package store

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path"
	"sync/atomic"
	"time"

	"github.com/tracymacding/sync"

	"github.com/tddhit/tools/bloom"
	"github.com/tddhit/tools/log"
)

const (
	maxNumElements    = 1e8
	falsePostive      = 0.0001
	maxMmapSize       = 1 << 30
	eliminateInterval = 15 * 24 * time.Hour
	syncInterval      = 10 * time.Second
	createInterval    = 24 * time.Hour
)

type bmeta struct {
	Count      uint64 `json:"count"`
	CreateTime int64  `json:"createTime"`
	FormatTime string `json:"formatTime"`
}

type fmeta struct {
	Blooms []*bmeta `json:"blooms"`
}

type filter struct {
	sync.RWMutex
	name    string
	dataDir string
	meta    fmeta

	cur    *bloom.Bloom
	blooms []*bloom.Bloom

	wg    sync.WaitGroup
	exitC chan struct{}
}

func newFilter(dataDir, topic string) (*filter, error) {
	f := &filter{
		name:    topic,
		dataDir: path.Join(dataDir, topic, "bloom"),

		exitC: make(chan struct{}),
	}
	if err := os.MkdirAll(f.dataDir, 0755); err != nil && !os.IsExist(err) {
		log.Error(err)
		return nil, err
	}
	if err := f.loadMetadata(); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	}
	if len(f.blooms) == 0 {
		now := time.Now()
		meta := &bmeta{
			CreateTime: now.Unix(),
			FormatTime: now.Format("2006-01-02 15:04:05"),
		}
		if err := f.createBloom(meta); err != nil {
			return nil, err
		}
		f.meta.Blooms = append(f.meta.Blooms, meta)
		f.sync()
	}
	f.wg.Add(1)
	go func() {
		f.createLoop()
		f.wg.Done()
	}()
	f.wg.Add(1)
	go func() {
		f.eliminateLoop()
		f.wg.Done()
	}()
	f.wg.Add(1)
	go func() {
		f.syncLoop()
		f.wg.Done()
	}()
	return f, nil
}

func (f *filter) loadMetadata() error {
	filename := fmt.Sprintf(path.Join(f.dataDir, "%s.bloom.meta"), f.name)
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&f.meta); err != nil {
		return err
	}
	for _, meta := range f.meta.Blooms {
		if err := f.createBloom(meta); err != nil {
			return err
		}
	}
	return nil
}

func (f *filter) createBloom(meta *bmeta) error {
	file := fmt.Sprintf(path.Join(f.dataDir, "%s.bloom.%d.dat"),
		f.name, meta.CreateTime)
	bloom, err := bloom.New(maxNumElements, falsePostive,
		bloom.WithMmap(file, maxMmapSize))
	if err != nil {
		return err
	}
	f.blooms = append(f.blooms, bloom)
	f.cur = bloom
	return nil
}

func (f *filter) add(key []byte) error {
	f.RLock()
	defer f.RUnlock()

	if err := f.cur.Add(key); err != nil {
		return err
	}
	atomic.AddUint64(&f.meta.Blooms[len(f.meta.Blooms)-1].Count, 1)
	return nil
}

func (f *filter) mayContain(key []byte) bool {
	f.RLock()
	defer f.RUnlock()

	var (
		wg    sync.WaitGroup
		exist bool
	)
	for _, b := range f.blooms {
		wg.Add(1)
		go func(b *bloom.Bloom) {
			if b.MayContain(key) {
				exist = true
			}
			wg.Done()
		}(b)
	}
	wg.Wait()
	return exist
}

func (f *filter) createLoop() {
	ticker := time.NewTicker(time.Hour)
	for {
		select {
		case <-ticker.C:
			f.Lock()
			now := time.Now()
			if len(f.meta.Blooms) > 0 {
				b := f.meta.Blooms[0]
				if time.Since(time.Unix(b.CreateTime, 0)) < createInterval {
					f.Unlock()
					continue
				}
			}
			meta := &bmeta{
				CreateTime: now.Unix(),
				FormatTime: now.Format("2006-01-02 15:04:05"),
			}
			if err := f.createBloom(meta); err != nil {
				log.Fatal(err)
			}
			f.meta.Blooms = append(f.meta.Blooms, meta)
			f.sync()
			f.Unlock()
		case <-f.exitC:
			goto exit
		}
	}
exit:
	ticker.Stop()
}

func (f *filter) eliminateLoop() {
	ticker := time.NewTicker(time.Hour)
	for {
		select {
		case <-ticker.C:
			f.Lock()
			if len(f.meta.Blooms) > 0 {
				b := f.meta.Blooms[0]
				if time.Since(time.Unix(b.CreateTime, 0)) >= eliminateInterval {
					f.blooms[0].Delete()
					f.blooms = f.blooms[1:]
					f.meta.Blooms = f.meta.Blooms[1:]
				}
			}
			f.Unlock()
		case <-f.exitC:
			goto exit
		}
	}
exit:
	ticker.Stop()
}

func (f *filter) syncLoop() {
	ticker := time.NewTicker(syncInterval)
	for {
		select {
		case <-ticker.C:
			f.Lock()
			if err := f.sync(); err != nil {
				log.Error(err)
			}
			f.Unlock()
		case <-f.exitC:
			goto exit
		}
	}
exit:
	ticker.Stop()
}

func (f *filter) close() error {
	f.Lock()
	defer f.Unlock()

	close(f.exitC)
	f.wg.Wait()
	if err := f.sync(); err != nil {
		return err
	}
	for _, b := range f.blooms {
		b.Close()
	}
	return nil
}

func (f *filter) sync() error {
	for _, b := range f.blooms {
		if err := b.Sync(); err != nil {
			log.Error(err)
			return err
		}
	}
	if err := f.persistMetadata(); err != nil {
		return err
	}
	return nil
}

func (f *filter) persistMetadata() error {
	filename := fmt.Sprintf(path.Join(f.dataDir, "%s.bloom.meta"), f.name)
	tmpFilename := fmt.Sprintf("%s.%d.tmp", filename, rand.Int())
	file, err := os.OpenFile(tmpFilename, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "    ")
	if err := encoder.Encode(f.meta); err != nil {
		return err
	}
	file.Sync()
	file.Close()
	return os.Rename(tmpFilename, filename)
}
