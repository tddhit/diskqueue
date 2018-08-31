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

	"github.com/tddhit/diskqueue/filter"
	pb "github.com/tddhit/diskqueue/pb"
)

type metadata struct {
	WriteID  uint64 `json:"writeID"`
	ReadID   uint64 `json:"readID"`
	Segments []struct {
		MinID      uint64 `json:"minID"`
		MaxID      uint64 `json:"maxID"`
		Size       uint32 `json:"size"`
		IndexCount uint32 `json:"indexCount"`
	} `json:"segments"`
}

type Consumer interface {
	String() string
}

type Topic struct {
	sync.RWMutex
	name          string
	dataPath      string
	meta          metadata
	timeout       int64
	curSeg        *segment
	segs          segments
	consumers     map[string]Consumer
	filter        *filter.Bloom
	inFlightQueue []*pb.Message
	inFlightMap   map[uint64]*pb.Message
	inFlightMutex sync.Mutex
	writeC        chan *pb.Message
	writeRspC     chan error
	syncEvery     int
	syncInterval  time.Duration
	needSync      bool
	readC         chan *pb.Message
	exitFlag      int32
	exitC         chan struct{}
	wg            sync.WaitGroup
}

func NewTopic(dataPath, topic string) (*Topic, error) {
	t := &Topic{
		name:          topic,
		dataPath:      dataPath,
		timeout:       int64(10 * time.Second),
		consumers:     make(map[string]Consumer),
		filter:        filter.New(),
		inFlightQueue: make([]*pb.Message, 0, 100),
		inFlightMap:   make(map[uint64]*pb.Message),
		writeC:        make(chan *pb.Message),
		writeRspC:     make(chan error),
		syncEvery:     10000,
		syncInterval:  30 * time.Second,
		readC:         make(chan *pb.Message),
		exitC:         make(chan struct{}),
	}
	if err := t.loadMetadata(); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	}
	if len(t.segs) == 0 {
		if err := t.createSegment(0); err != nil {
			return nil, err
		}
	}
	t.wg.Add(3)
	go func() {
		t.writeLoop()
		t.wg.Done()
	}()
	go func() {
		t.readLoop()
		t.wg.Done()
	}()
	go func() {
		t.scanInFlightLoop()
		t.wg.Done()
	}()
	return t, nil
}

func (t *Topic) PutMessage(data []byte) error {
	t.RLock()
	defer t.RUnlock()

	if atomic.LoadInt32(&t.exitFlag) == 1 {
		return errors.New("diskqueue already close")
	}
	t.writeC <- &pb.Message{Data: data}
	return <-t.writeRspC
}

func (t *Topic) GetMessage() *pb.Message {
	msg := <-t.readC
	t.pushInFlight(msg)
	return msg
}

func (t *Topic) Ack(msgID uint64) error {
	return t.removeFromInFlight(msgID)
}

func (t *Topic) AddConsumer(consumer Consumer) error {
	t.Lock()
	defer t.Unlock()

	if _, ok := t.consumers[consumer.String()]; ok {
		return errors.New("already subscribe.")
	} else {
		t.consumers[consumer.String()] = consumer
	}
	return nil
}

func (t *Topic) RemoveConsumer(name string) {
	t.Lock()
	defer t.Unlock()

	delete(t.consumers, name)
}

func (t *Topic) seek(msgID uint64) (seg *segment, pos uint32, err error) {
	if len(t.segs) == 0 {
		err = errors.New("empty segments")
		return
	}
	index := sort.Search(len(t.segs), func(i int) bool {
		return t.segs[i].minID > msgID
	})
	if index == 0 {
		err = errors.New("not found segment")
		return
	}
	seg = t.segs[index-1]
	pos, err = seg.seek(msgID)
	log.Debugf("Seek\tmsgID=%d\tseg=%d\tpos=%d\n", msgID, seg.minID, pos)
	return
}

func (t *Topic) createSegment(msgID uint64) (err error) {
	seg, err := newSegment(t.dataPath, t.name,
		os.O_CREATE|os.O_RDWR|os.O_APPEND, msgID, msgID)
	if err != nil {
		return
	}
	t.segs = append(t.segs, seg)
	t.curSeg = seg
	return
}

func (t *Topic) writeOne(msg *pb.Message) error {
	//if t.filter.CheckOrAdd(msg.GetData()) {
	//	log.Debug(msg.ID, string(msg.GetData()), "filter")
	//	return nil
	//}
	if t.curSeg.full() {
		if err := t.createSegment(msg.ID); err != nil {
			return err
		}
	}
	if err := t.curSeg.writeOne(msg); err != nil {
		return err
	}
	atomic.AddUint64(&t.meta.WriteID, 1)
	log.Debugf("writeOne\tid=%d\tdata=%s", msg.ID, string(msg.Data))
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
			msg.Timestamp = time.Now().UnixNano()
			t.writeRspC <- t.writeOne(msg)
		case <-syncTicker.C:
			if count == 0 {
				continue
			}
			t.needSync = true
		case <-t.exitC:
			goto exit
		}
	}
exit:
	log.Infof("diskqueue(%s) exit writeLoop.", t.name)
}

func (t *Topic) readLoop() {
	var (
		msg *pb.Message
		seg *segment
		pos uint32
		err error
	)
	for {
		if atomic.LoadInt32(&t.exitFlag) == 1 {
			goto exit
		}
		readID := atomic.LoadUint64(&t.meta.ReadID)
		if seg == nil {
			seg, pos, err = t.seek(readID)
			if err != nil {
				log.Fatal(err)
			}
		}
		if readID < atomic.LoadUint64(&t.meta.WriteID) {
			if readID < atomic.LoadUint64(&seg.maxID) {
				msg, pos, err = seg.readOne(readID, pos)
				if err != nil {
					log.Fatal(err)
				}
			} else {
				seg = nil
				continue
			}
		} else {
			runtime.Gosched()
			continue
		}
		select {
		case t.readC <- msg:
			atomic.AddUint64(&t.meta.ReadID, 1)
		case <-t.exitC:
			goto exit
		}
	}
exit:
	close(t.readC)
	log.Infof("topic(%s) exit readLoop.", t.name)
}

func (t *Topic) scanInFlightLoop() {
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ticker.C:
			t.processInFlight()
		case <-t.exitC:
			goto exit
		}
	}
exit:
	ticker.Stop()
}

func (t *Topic) processInFlight() {
	now := time.Now().UnixNano()
	for {
		msg, err := t.popInFlight(now)
		if err != nil {
			return
		}
		log.Info("requeue", msg.GetID())
		t.PutMessage(msg.GetData())
	}
}

func (t *Topic) pushInFlight(msg *pb.Message) {
	t.inFlightMutex.Lock()
	defer t.inFlightMutex.Unlock()

	t.inFlightQueue = append(t.inFlightQueue, msg)
	t.inFlightMap[msg.GetID()] = msg
}

func (t *Topic) popInFlight(now int64) (*pb.Message, error) {
	t.inFlightMutex.Lock()
	defer t.inFlightMutex.Unlock()

	if len(t.inFlightQueue) == 0 {
		return nil, errors.New("inFlightQueue is empty.")
	}
	msg := t.inFlightQueue[0]
	log.Info("comp", now, msg.GetTimestamp()+t.timeout)
	if now < msg.GetTimestamp()+t.timeout {
		return nil, errors.New("no timeout message.")
	}
	t.inFlightQueue = t.inFlightQueue[1:]
	delete(t.inFlightMap, msg.GetID())
	return msg, nil
}

func (t *Topic) removeFromInFlight(msgID uint64) error {
	t.inFlightMutex.Lock()
	defer t.inFlightMutex.Unlock()

	delete(t.inFlightMap, msgID)
	return nil
}

func (t *Topic) persistMetadata() error {
	filename := fmt.Sprintf(path.Join(t.dataPath, "%s.diskqueue.meta"), t.name)
	tmpFilename := fmt.Sprintf("%s.%d.tmp", filename, rand.Int())
	f, err := os.OpenFile(tmpFilename, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(f)
	if err := encoder.Encode(t.meta); err != nil {
		return err
	}
	f.Sync()
	f.Close()
	return os.Rename(tmpFilename, filename)
}

func (t *Topic) loadMetadata() error {
	filename := fmt.Sprintf(path.Join(t.dataPath, "%s.diskqueue.meta"), t.name)
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(f)
	if err := decoder.Decode(t.meta); err != nil {
		return err
	}
	var flag int
	for i, segment := range t.meta.Segments {
		if i != len(t.meta.Segments)-1 {
			flag = os.O_RDONLY
		} else {
			flag = os.O_CREATE | os.O_RDWR | os.O_APPEND
		}
		seg, err := newSegment(t.dataPath, t.name, flag,
			segment.MinID, segment.MaxID, segment.Size, segment.IndexCount)
		if err != nil {
			return err
		}
		t.segs = append(t.segs, seg)
	}
	if len(t.segs) > 0 {
		t.curSeg = t.segs[len(t.segs)-1]
	}
	return nil
}

func (t *Topic) Close() error {
	if err := t.exit(); err != nil {
		return err
	}
	return t.sync()
}

func (t *Topic) exit() error {
	t.Lock()
	defer t.Unlock()

	if !atomic.CompareAndSwapInt32(&t.exitFlag, 0, 1) {
		return errors.New("already exit")
	}
	close(t.exitC)
	t.wg.Wait()
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
