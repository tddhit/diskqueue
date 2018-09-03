package service

import (
	"container/heap"
	"errors"
	"fmt"
	"sync"
	"time"

	pb "github.com/tddhit/diskqueue/pb"
	"github.com/tddhit/diskqueue/store"
	"github.com/tddhit/tools/log"
)

const (
	maxInflight = 100
)

type inflight struct {
	sync.RWMutex
	topic   *store.Topic
	queue   *pqueue
	m       map[uint64]*pb.Message
	cond    *sync.Cond
	timeout int64
	exitC   chan struct{}
}

func newInflight(t *store.Topic) *inflight {
	f := &inflight{
		topic:   t,
		queue:   &pqueue{},
		m:       make(map[uint64]*pb.Message, maxInflight),
		cond:    sync.NewCond(&sync.Mutex{}),
		timeout: int64(10 * time.Second),
		exitC:   make(chan struct{}),
	}
	heap.Init(f.queue)
	go f.scanLoop()
	return f
}

func (f *inflight) scanLoop() {
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ticker.C:
			f.process()
		case <-f.exitC:
			goto exit
		}
	}
exit:
	ticker.Stop()
}

func (f *inflight) process() {
	now := time.Now().UnixNano()
	for {
		msg, err := f.pop(now)
		if err != nil {
			return
		}
		if msg != nil {
			log.Info("requeue", msg.GetID())
			f.topic.PutMessage(msg.GetData())
		}
	}
}

func (f *inflight) push(msg *pb.Message) {
	if msg == nil {
		return
	}
	f.Lock()
	defer f.Unlock()

	msg.Timestamp = time.Now().UnixNano()
	heap.Push(f.queue, msg)
	f.m[msg.GetID()] = msg
	log.Info("push", msg.GetID())
}

func (f *inflight) pop(now int64) (*pb.Message, error) {
	f.Lock()
	defer f.Unlock()

	if f.queue.Len() == 0 {
		return nil, errors.New("inFlightQueue is empty.")
	}
	msg := heap.Pop(f.queue).(*pb.Message)
	if now < msg.GetTimestamp()+f.timeout {
		heap.Push(f.queue, msg)
		return nil, errors.New("no timeout message.")
	}
	if _, ok := f.m[msg.GetID()]; ok {
		delete(f.m, msg.GetID())
		f.cond.Signal()
		return msg, nil
	}
	return nil, nil
}

func (f *inflight) remove(msgID uint64) error {
	f.Lock()
	defer f.Unlock()

	if _, ok := f.m[msgID]; ok {
		delete(f.m, msgID)
		f.cond.Signal()
		return nil
	}
	return fmt.Errorf("msgID(%d) does not exist in InflightMap", msgID)
}

func (f *inflight) full() bool {
	f.RLock()
	defer f.RUnlock()

	return len(f.m) >= maxInflight
}
