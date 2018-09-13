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

type Topic struct {
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

	syncEvery    int
	syncInterval time.Duration
	needSync     bool

	exitFlag int32
	exitC    chan struct{}
	wg       sync.WaitGroup
}

func NewTopic(dataDir, topic string) (*Topic, error) {
	t := &Topic{
		Name:    topic,
		dataDir: dataDir,

		writeC:    make(chan *pb.Message),
		writeRspC: make(chan error),

		syncEvery:    10000,
		syncInterval: 10 * time.Second,

		exitC: make(chan struct{}),
	}
	filter, err := newFilter(dataDir, topic)
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
	return t, nil
}

func (t *Topic) push(data []byte) error {
	t.RLock()
	defer t.RUnlock()

	if atomic.LoadInt32(&t.exitFlag) == 1 {
		return errors.New("diskqueue already close")
	}
	t.writeC <- &pb.Message{Data: data}
	return <-t.writeRspC
}

func (t *Topic) get() (*pb.Message, int64, error) {
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

func (t *Topic) advance(pos int64) {
	t.meta.ReadID++
	t.readSeg.meta.ReadID++
	t.readSeg.meta.ReadPos = pos
}

func (t *Topic) seek(msgID uint64) (*segment, error) {
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

func (t *Topic) createSegment(mode int, meta *smeta) error {
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

func (t *Topic) writeOne(msg *pb.Message) error {
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

func (t *Topic) writeLoop() {
	var count int
	syncTicker := time.NewTicker(t.syncInterval)
	for {
		if count == t.syncEvery {
			t.needSync = true
		}
		if t.needSync {
			if err := t.sync(); err != nil {
				log.Error(err)
			}
			count = 0
		}
		select {
		case msg := <-t.writeC:
			count++
			msg.ID = atomic.LoadUint64(&t.meta.WriteID)
			t.writeRspC <- t.writeOne(msg)
		case <-syncTicker.C:
			t.needSync = true
		case <-t.exitC:
			goto exit
		}
	}
exit:
	log.Infof("diskqueue(%s) exit writeLoop.", t.Name)
}

func (t *Topic) recycleLoop() {
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

func (t *Topic) persistMetadata() error {
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

func (t *Topic) loadMetadata() error {
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

func (t *Topic) Close() error {
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

func (t *Topic) sync() error {
	for _, seg := range t.segs {
		if err := seg.sync(); err != nil {
			return err
		}
	}
	if err := t.persistMetadata(); err != nil {
		return err
	}
	t.needSync = false
	return nil
}
