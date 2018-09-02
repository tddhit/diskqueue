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
	stateInit = iota
	stateSubscribed
	stateClosing

	maxInFlight = 100
)

type client struct {
	state           int32
	addr            string
	topic           *store.Topic
	inFlightTimeout int64
	inFlightQueue   *pqueue
	inFlightMap     map[uint64]*pb.Message
	inFlightMutex   sync.Mutex
	inFlightCond    *sync.Cond
	exitC           chan struct{}
}

func newClient(addr string, topic *store.Topic) *client {
	c := &client{
		state:           stateSubscribed,
		addr:            addr,
		topic:           topic,
		inFlightTimeout: int64(10 * time.Second),
		inFlightQueue:   &pqueue{},
		inFlightMap:     make(map[uint64]*pb.Message, maxInFlight),
		inFlightCond:    sync.NewCond(&sync.Mutex{}),
		exitC:           make(chan struct{}),
	}
	heap.Init(c.inFlightQueue)
	go c.scanInFlightLoop()
	return c
}

func (c *client) String() string {
	return c.addr
}

func (c *client) scanInFlightLoop() {
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ticker.C:
			c.processInFlight()
		case <-c.exitC:
			goto exit
		}
	}
exit:
	ticker.Stop()
}

func (c *client) processInFlight() {
	now := time.Now().UnixNano()
	for {
		msg, err := c.popInFlight(now)
		if err != nil {
			return
		}
		if msg != nil {
			log.Info("requeue", msg.GetID())
			c.topic.PutMessage(msg.GetData())
		}
	}
}

func (c *client) pushInFlight(msg *pb.Message) {
	if msg == nil {
		return
	}
	msg.Timestamp = time.Now().UnixNano()
	c.inFlightMutex.Lock()
	heap.Push(c.inFlightQueue, msg)
	log.Info("push", msg.GetID())
	c.inFlightMap[msg.GetID()] = msg
	c.inFlightMutex.Unlock()
}

func (c *client) popInFlight(now int64) (*pb.Message, error) {
	c.inFlightMutex.Lock()
	defer c.inFlightMutex.Unlock()

	if c.inFlightQueue.Len() == 0 {
		return nil, errors.New("inFlightQueue is empty.")
	}
	msg := heap.Pop(c.inFlightQueue).(*pb.Message)
	log.Debug("comp", now, msg.GetTimestamp()+c.inFlightTimeout)
	if now < msg.GetTimestamp()+c.inFlightTimeout {
		heap.Push(c.inFlightQueue, msg)
		return nil, errors.New("no timeout message.")
	}
	if _, ok := c.inFlightMap[msg.GetID()]; ok {
		log.Debug("delete", msg.GetID())
		delete(c.inFlightMap, msg.GetID())
		c.inFlightCond.Signal()
		return msg, nil
	}
	return nil, nil
}

func (c *client) removeFromInFlight(msgID uint64) error {
	c.inFlightMutex.Lock()
	defer c.inFlightMutex.Unlock()

	if _, ok := c.inFlightMap[msgID]; ok {
		log.Info("delete", msgID)
		delete(c.inFlightMap, msgID)
		c.inFlightCond.Signal()
		return nil
	}
	return fmt.Errorf("msgID(%d) does not exist in InFlightMap", msgID)
}

func (c *client) canPull() bool {
	c.inFlightMutex.Lock()
	defer c.inFlightMutex.Unlock()

	return len(c.inFlightMap) < maxInFlight
}

type pqueue []*pb.Message

func (q pqueue) Len() int { return len(q) }
func (q pqueue) Less(i, j int) bool {
	return q[i].GetTimestamp() < q[j].GetTimestamp()
}
func (q pqueue) Swap(i, j int) { q[i], q[j] = q[j], q[i] }

func (q *pqueue) Push(x interface{}) {
	*q = append(*q, x.(*pb.Message))
}

func (q *pqueue) Pop() interface{} {
	old := *q
	n := len(old)
	x := old[n-1]
	*q = old[0 : n-1]
	return x
}
