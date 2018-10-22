package store

import (
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/tddhit/tools/log"
	"github.com/tddhit/tools/mmap"

	pb "github.com/tddhit/diskqueue/pb"
)

type topic struct {
	sync.RWMutex
	name         string
	dataDir      string
	writeID      uint64
	segs         segments
	channels     map[string]*channel
	filter       *filter
	writeSeg     *segment
	writeC       chan *pb.Message
	writeRspC    chan error
	syncInterval time.Duration
	exitFlag     int32
	exitC        chan struct{}
	wg           sync.WaitGroup
}

func newTopic(dataDir, name string) (*topic, error) {
	t := &topic{
		name:         name,
		dataDir:      path.Join(dataDir, name, "data"),
		writeC:       make(chan *pb.Message),
		writeRspC:    make(chan error),
		syncInterval: 10 * time.Second,
		exitC:        make(chan struct{}),
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
		if err := t.createSegment(mmap.APPEND, 0); err != nil {
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

func (t *topic) push(data []byte) error {
	t.RLock()
	defer t.RUnlock()

	if atomic.LoadInt32(&t.exitFlag) == 1 {
		return errors.New("diskqueue already close")
	}
	t.writeC <- &pb.Message{Data: data}
	return <-t.writeRspC
}

func (t *topic) seek(msgID uint64) *segment {
	if len(t.segs) == 0 {
		log.Fatal("empty segments")
	}
	index := sort.Search(len(t.segs), func(i int) bool {
		return t.segs[i].minID > msgID
	})
	if index == 0 {
		log.Fatal("not found segment")
	}
	return t.segs[index-1]
}

func (t *topic) createSegment(mode int, minID uint64) error {
	file := fmt.Sprintf(path.Join(t.dataDir, "%s.diskqueue.%d.dat"),
		t.name, minID)
	seg, err := newSegment(file, mode, mmap.SEQUENTIAL, minID)
	if err != nil {
		return err
	}
	t.segs = append(t.segs, seg)
	t.writeSeg = seg
	return nil
}

func (t *topic) writeOne(msg *pb.Message) error {
	if t.writeSeg.full() {
		if err := t.createSegment(mmap.APPEND, msg.ID); err != nil {
			return err
		}
		t.sync()
	}
	if err := t.writeSeg.writeOne(msg); err != nil {
		return err
	}
	atomic.AddUint64(&t.writeID, 1)
	log.Debugf("writeOne\tid=%d\tdata=%s", msg.ID, string(msg.GetData()))
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
				t.segs[0].delete()
				t.segs = t.segs[1:]
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
	meta := &pb.Metadata{
		Topic:   t.name,
		WriteID: t.writeID,
	}
	for _, c := range t.channels {
		meta.Channels = append(meta.Channels, &pb.Metadata_Channel{
			Name:    c.name,
			ReadID:  c.readID,
			ReadPos: c.readPos,
		})
	}
	for _, s := range t.segs {
		meta.Segments = append(meta.Segments, &pb.Metadata_Segment{
			MinID:    s.minID,
			WriteID:  s.writeID,
			WritePos: s.writePos,
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
	meta := &pb.Metadata{}
	if err := proto.Unmarshal(data, meta); err != nil {
		log.Error(err)
		return err
	}
	var mode int
	for i, s := range meta.Segments {
		if i != len(meta.Segments)-1 {
			mode = mmap.RDONLY
		} else {
			mode = mmap.APPEND
		}
		if err := t.createSegment(mode, s.MinID); err != nil {
			return err
		}
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
	c := newChannel(name, t)
	t.channels[name] = c
	t.Unlock()
	return c, true
}

func (t *topic) close() error {
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
