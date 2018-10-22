package store

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"

	pb "github.com/tddhit/diskqueue/pb"
)

type consumer interface {
	Close() error
	ID() string
}

type channel struct {
	sync.RWMutex
	name       string
	topic      *topic
	readSeg    *segment
	readID     uint64
	readPos    int64
	curMsgSize int
	consumers  map[string]consumer
	exitFlag   int32
}

func newChannel(name string, topic *topic) *channel {
	return &channel{
		name:      name,
		topic:     topic,
		consumers: make(map[string]consumer),
	}
}

func (c *channel) addConsumer(co consumer) error {
	c.Lock()
	defer c.Unlock()

	if _, ok := c.consumers[co.ID()]; ok {
		return fmt.Errorf("consumer(%s) already exist", co.ID())
	}
	c.consumers[co.ID()] = co
	return nil
}

func (c *channel) removeConsumer(id string) {
	c.Lock()
	defer c.Unlock()

	delete(c.consumers, id)
}

func (c *channel) get() (*pb.Message, error) {
	for {
		if atomic.LoadInt32(&c.exitFlag) == 1 {
			return nil, fmt.Errorf("channel(%s) already close", c.name)
		}
		if c.readSeg == nil {
			c.readSeg = c.topic.seek(c.readID)
			c.readPos = 0
		}
		if c.readID < atomic.LoadUint64(&c.topic.writeID) {
			if c.readID < atomic.LoadUint64(&c.readSeg.writeID) {
				msg, size := c.readSeg.readOne(c.readID, c.readPos)
				c.readPos += size
				return msg, nil
			} else {
				c.readSeg = nil
				continue
			}
		} else {
			runtime.Gosched()
			continue
		}
	}
}

func (c *channel) advance() {
	c.RLock()
	defer c.RUnlock()

	c.readID++
}

func (c *channel) close() {
	if !atomic.CompareAndSwapInt32(&c.exitFlag, 0, 1) {
		return
	}
	for _, co := range c.consumers {
		co.Close()
	}
}
