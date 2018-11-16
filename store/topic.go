package store

import (
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gogo/protobuf/proto"

	"github.com/tddhit/diskqueue/pb"
	"github.com/tddhit/tools/log"
	"github.com/tddhit/tools/mmap"
)

type topic struct {
	sync.RWMutex
	name    string
	dataDir string

	channels map[string]*channel
	segs     segments
	seglock  sync.RWMutex
	writeSeg *segment
	writeID  uint64

	writeC       chan *diskqueuepb.Message
	writeRspC    chan error
	syncInterval time.Duration
	exitFlag     int32
	exitC        chan struct{}
	wg           sync.WaitGroup
}

func newTopic(dataDir, name string) (*topic, error) {
	t := &topic{
		name:    name,
		dataDir: path.Join(dataDir, name, "data"),

		channels: make(map[string]*channel),

		writeC:       make(chan *diskqueuepb.Message),
		writeRspC:    make(chan error),
		syncInterval: 10 * time.Second,
		exitC:        make(chan struct{}),
	}
	if err := os.MkdirAll(t.dataDir, 0755); err != nil && !os.IsExist(err) {
		log.Error(err)
		return nil, err
	}
	if err := t.loadMetadata(); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	}
	if len(t.segs) == 0 {
		err := t.createSegment(mmap.MODE_APPEND, 0, 0, 0, time.Now().Unix())
		if err != nil {
			return nil, err
		}
		t.sync()
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

func (t *topic) push(data []byte) (uint64, error) {
	t.RLock()
	defer t.RUnlock()

	if atomic.LoadInt32(&t.exitFlag) == 1 {
		return 0, errors.New("diskqueue already close")
	}
	t.writeC <- &diskqueuepb.Message{Data: data}
	err := <-t.writeRspC
	if err == nil {
		return t.writeID - 1, nil
	}
	return 0, err
}

func (t *topic) createSegment(
	mode int,
	minID uint64,
	writeID uint64,
	writePos int64,
	ctime int64) error {

	file := fmt.Sprintf(path.Join(t.dataDir, "%s.diskqueue.%d.dat"),
		t.name, minID)
	seg, err := newSegment(file, mode, mmap.ADVISE_SEQUENTIAL,
		minID, writeID, writePos, ctime)
	if err != nil {
		return err
	}

	t.seglock.Lock()
	t.segs = append(t.segs, seg)
	t.seglock.Unlock()

	t.writeSeg = seg
	return nil
}

func (t *topic) writeOne(msg *diskqueuepb.Message) error {
	if t.writeSeg.full() {
		err := t.createSegment(mmap.MODE_APPEND, msg.ID, msg.ID, 0, time.Now().Unix())
		if err != nil {
			return err
		}
		t.sync()
	}
	if err := t.writeSeg.writeOne(msg); err != nil {
		return err
	}
	atomic.AddUint64(&t.writeID, 1)
	return nil
}

func (t *topic) writeLoop() {
	for {
		select {
		case msg := <-t.writeC:
			msg.ID = atomic.LoadUint64(&t.writeID)
			t.writeRspC <- t.writeOne(msg)
		case <-t.exitC:
			goto exit
		}
	}
exit:
	log.Infof("diskqueue(%s) exit writeLoop.", t.name)
}

func (t *topic) recycleLoop() {
	ticker := time.NewTicker(time.Hour)
	for {
		select {
		case <-ticker.C:
			t.Lock()
			if len(t.segs) > 1 {
				if time.Since(time.Unix(t.segs[0].ctime, 0)) >= 7*24*time.Hour {
					t.segs[0].delete()
					t.segs = t.segs[1:]
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
	filename := fmt.Sprintf(path.Join(t.dataDir, "%s.diskqueue.meta"), t.name)
	tmpFilename := fmt.Sprintf("%s.%d.tmp", filename, rand.Int())
	f, err := os.OpenFile(tmpFilename, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	meta := &diskqueuepb.Metadata{
		Topic:   t.name,
		WriteID: t.writeID,
	}
	for _, c := range t.channels {
		meta.Channels = append(meta.Channels, &diskqueuepb.Metadata_Channel{
			Name:    c.name,
			ReadID:  c.readID,
			ReadPos: c.readPos,
		})
	}
	for _, s := range t.segs {
		meta.Segments = append(meta.Segments, &diskqueuepb.Metadata_Segment{
			MinID:      s.minID,
			WriteID:    s.writeID,
			WritePos:   s.writePos,
			CreateTime: s.ctime,
		})
	}
	data, err := proto.Marshal(meta)
	if err != nil {
		log.Error(err)
		return err
	}
	n, err := f.Write(data)
	if n != len(data) {
		return errors.New("persist metadata fail.")
	}
	if err != nil {
		log.Error(err)
		return err
	}
	f.Sync()
	f.Close()
	return os.Rename(tmpFilename, filename)
}

func (t *topic) loadMetadata() error {
	filename := fmt.Sprintf(path.Join(t.dataDir, "%s.diskqueue.meta"), t.name)
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Error(err)
		return err
	}
	meta := &diskqueuepb.Metadata{}
	if err := proto.Unmarshal(data, meta); err != nil {
		log.Error(err)
		return err
	}
	t.writeID = meta.WriteID
	var mode int
	for i, s := range meta.Segments {
		if i != len(meta.Segments)-1 {
			mode = mmap.MODE_RDONLY
		} else {
			mode = mmap.MODE_APPEND
		}
		err := t.createSegment(mode, s.MinID, s.WriteID, s.WritePos, s.CreateTime)
		if err != nil {
			return err
		}
	}
	for _, c := range meta.Channels {
		t.channels[c.Name] = newChannel(c.Name, t, c.ReadID, c.ReadPos)
	}
	return nil
}

func (t *topic) getOrCreateChannel(name string) (_ *channel, created bool) {
	t.RLock()
	if c, ok := t.channels[name]; ok {
		t.RUnlock()
		return c, false
	}
	t.RUnlock()

	t.Lock()
	if c, ok := t.channels[name]; ok {
		t.Unlock()
		return c, false
	}
	c := &channel{
		name:  name,
		topic: t,
	}
	t.channels[name] = c
	t.Unlock()
	return c, true
}

func (t *topic) close() error {
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
