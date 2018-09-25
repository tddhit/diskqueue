package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tddhit/tools/log"
	"github.com/tddhit/tools/mmap"

	pb "github.com/tddhit/diskqueue/pb"
)

type tmeta struct {
	WriteID  uint64   `json:"writeID"`
	ReadID   uint64   `json:"readID"`
	Segments []*smeta `json:"segments"`
}

type topic struct {
	sync.RWMutex
	Name    string
	dataDir string
	meta    tmeta

	filter   *filter
	readSeg  *segment
	writeSeg *segment
	segs     segments

	writeC    chan *pb.Message
	writeRspC chan error

	syncInterval time.Duration

	exitFlag int32
	exitC    chan struct{}
	wg       sync.WaitGroup
}

func newTopic(dataDir, name string) (*topic, error) {
	t := &topic{
		Name:    name,
		dataDir: path.Join(dataDir, name, "data"),

		writeC:    make(chan *pb.Message),
		writeRspC: make(chan error),

		syncInterval: 10 * time.Second,

		exitC: make(chan struct{}),
	}
	if err := os.MkdirAll(t.dataDir, 0755); err != nil && !os.IsExist(err) {
		log.Error(err)
		return nil, err
	}
	filter, err := newFilter(dataDir, name)
	if err != nil {
		return nil, err
	}
	t.filter = filter
	if err := t.loadMetadata(); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	}
	if len(t.segs) == 0 {
		meta := &smeta{}
		if err := t.createSegment(mmap.APPEND, meta); err != nil {
			return nil, err
		}
		t.meta.Segments = append(t.meta.Segments, meta)
		t.sync()
	}
	t.readSeg, err = t.seek(t.meta.ReadID)
	if err != nil {
		return nil, err
	}
	t.wg.Add(1)
	go func() {
		t.writeLoop()
		t.wg.Done()
	}()
	t.wg.Add(1)
	go func() {
		t.recycleLoop()
		t.wg.Done()
	}()
	t.wg.Add(1)
	go func() {
		t.syncLoop()
		t.wg.Done()
	}()
	return t, nil
}

func (t *topic) push(data []byte) error {
	t.RLock()
	defer t.RUnlock()

	if atomic.LoadInt32(&t.exitFlag) == 1 {
		return errors.New("diskqueue already close")
	}
	t.writeC <- &pb.Message{Data: data}
	return <-t.writeRspC
}

func (t *topic) get() (*pb.Message, int64, error) {
	var (
		msg *pb.Message
		err error
		pos int64
	)
	for {
		if atomic.LoadInt32(&t.exitFlag) == 1 {
			return nil, 0, errors.New("diskqueue already close")
		}
		if t.readSeg == nil {
			t.readSeg, err = t.seek(t.meta.ReadID)
			if err != nil {
				log.Fatal(err)
			}
		}
		if t.meta.ReadID < atomic.LoadUint64(&t.meta.WriteID) {
			if t.meta.ReadID < atomic.LoadUint64(&t.readSeg.meta.WriteID) {
				msg, pos, err = t.readSeg.readOne(t.meta.ReadID)
				if err != nil {
					log.Fatal(err)
				}
			} else {
				t.readSeg = nil
				continue
			}
		} else {
			runtime.Gosched()
			continue
		}
		return msg, pos, nil
	}
}

func (t *topic) advance(pos int64) {
	t.RLock()
	defer t.RUnlock()

	t.meta.ReadID++
	t.readSeg.meta.ReadID++
	t.readSeg.meta.ReadPos = pos
}

func (t *topic) seek(msgID uint64) (*segment, error) {
	if len(t.segs) == 0 {
		return nil, errors.New("empty segments")
	}
	index := sort.Search(len(t.segs), func(i int) bool {
		return t.segs[i].meta.MinID > msgID
	})
	if index == 0 {
		return nil, errors.New("not found segment")
	}
	return t.segs[index-1], nil
}

func (t *topic) createSegment(mode int, meta *smeta) error {
	file := fmt.Sprintf(path.Join(t.dataDir, "%s.diskqueue.%d.dat"),
		t.Name, meta.MinID)
	seg, err := newSegment(file, mode, mmap.SEQUENTIAL, meta)
	if err != nil {
		return err
	}
	t.segs = append(t.segs, seg)
	t.writeSeg = seg
	return nil
}

func (t *topic) writeOne(msg *pb.Message) error {
	if t.writeSeg.full() {
		meta := &smeta{
			MinID:   msg.ID,
			ReadID:  msg.ID,
			WriteID: msg.ID,
		}
		if err := t.createSegment(mmap.APPEND, meta); err != nil {
			return err
		}
		t.meta.Segments = append(t.meta.Segments, meta)
		t.sync()
	}
	if err := t.writeSeg.writeOne(msg); err != nil {
		return err
	}
	atomic.AddUint64(&t.meta.WriteID, 1)
	log.Debugf("writeOne\tid=%d\tdata=%s", msg.ID, string(msg.GetData()))
	return nil
}

func (t *topic) writeLoop() {
	for {
		select {
		case msg := <-t.writeC:
			msg.ID = atomic.LoadUint64(&t.meta.WriteID)
			t.writeRspC <- t.writeOne(msg)
		case <-t.exitC:
			goto exit
		}
	}
exit:
	log.Infof("diskqueue(%s) exit writeLoop.", t.Name)
}

func (t *topic) recycleLoop() {
	ticker := time.NewTicker(time.Hour)
	for {
		select {
		case <-ticker.C:
			t.Lock()
			if len(t.meta.Segments) > 0 {
				seg := t.meta.Segments[0]
				if seg.ReadID == seg.WriteID {
					t.segs[0].delete()
					t.segs = t.segs[1:]
					t.meta.Segments = t.meta.Segments[1:]
				}
			}
			t.Unlock()
		case <-t.exitC:
			goto exit
		}
	}
exit:
	ticker.Stop()
}

func (t *topic) syncLoop() {
	ticker := time.NewTicker(t.syncInterval)
	for {
		select {
		case <-ticker.C:
			t.Lock()
			if err := t.sync(); err != nil {
				log.Error(err)
			}
			t.Unlock()
		case <-t.exitC:
			goto exit
		}
	}
exit:
	ticker.Stop()
}

func (t *topic) persistMetadata() error {
	filename := fmt.Sprintf(path.Join(t.dataDir, "%s.diskqueue.meta"), t.Name)
	tmpFilename := fmt.Sprintf("%s.%d.tmp", filename, rand.Int())
	f, err := os.OpenFile(tmpFilename, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "    ")
	if err := encoder.Encode(t.meta); err != nil {
		return err
	}
	f.Sync()
	f.Close()
	return os.Rename(tmpFilename, filename)
}

func (t *topic) loadMetadata() error {
	filename := fmt.Sprintf(path.Join(t.dataDir, "%s.diskqueue.meta"), t.Name)
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(f)
	if err := decoder.Decode(&t.meta); err != nil {
		return err
	}
	var mode int
	for i, meta := range t.meta.Segments {
		if i != len(t.meta.Segments)-1 {
			mode = mmap.RDONLY
		} else {
			mode = mmap.APPEND
		}
		if err := t.createSegment(mode, meta); err != nil {
			return err
		}
	}
	return nil
}

func (t *topic) Close() error {
	log.Debug("Close")
	t.Lock()
	defer t.Unlock()

	if !atomic.CompareAndSwapInt32(&t.exitFlag, 0, 1) {
		return errors.New("already exit")
	}
	close(t.exitC)
	t.wg.Wait()
	if err := t.sync(); err != nil {
		return err
	}
	for _, seg := range t.segs {
		seg.close()
	}
	return nil
}

func (t *topic) sync() error {
	for _, seg := range t.segs {
		if err := seg.sync(); err != nil {
			return err
		}
	}
	if err := t.persistMetadata(); err != nil {
		return err
	}
	return nil
}
