package store

import (
	"fmt"
	"sort"
	"sync"
	"sync/atomic"

	"github.com/tddhit/diskqueue/pb"
	"github.com/tddhit/tools/log"
)

type channel struct {
	name     string
	topic    *topic
	readSeg  *segment
	readID   uint64
	readPos  int64
	cond     *sync.Cond
	exitFlag int32
}

func newChannel(
	name string,
	t *topic,
	readID uint64,
	readPos int64,
	cond *sync.Cond) *channel {

	c := &channel{
		name:    name,
		topic:   t,
		readID:  readID,
		readPos: readPos,
		cond:    cond,
	}
	c.readSeg = c.seek(c.readID)
	return c
}

func (c *channel) get() (*diskqueuepb.Message, int64, error) {
	for {
		if atomic.LoadInt32(&c.exitFlag) == 1 {
			return nil, 0, fmt.Errorf("channel(%s.%s) already close",
				c.topic.name, c.name)
		}
		if c.readSeg == nil {
			c.readSeg = c.seek(c.readID)
			c.readPos = 0
		}
		if c.readID < atomic.LoadUint64(&c.topic.writeID) {
			if c.readID < atomic.LoadUint64(&c.readSeg.writeID) {
				msg, nextPos := c.readSeg.readOne(c.readID, c.readPos)
				return msg, nextPos, nil
			} else {
				c.readSeg = nil
				continue
			}
		} else {
			c.cond.L.Lock()
			c.cond.Wait()
			c.cond.L.Unlock()
		}
	}
}

func (c *channel) advance(nextPos int64) {
	c.readID++
	c.readPos = nextPos
}

func (c *channel) seek(msgID uint64) *segment {
	c.topic.seglock.RLock()
	defer c.topic.seglock.RUnlock()

	if len(c.topic.segs) == 0 {
		log.Fatal("empty segments")
	}
	index := sort.Search(len(c.topic.segs), func(i int) bool {
		return c.topic.segs[i].minID > msgID
	})
	if index == 0 {
		log.Fatal("not found segment")
	}
	return c.topic.segs[index-1]
}

func (c *channel) close() {
	if !atomic.CompareAndSwapInt32(&c.exitFlag, 0, 1) {
		return
	}
	c.cond.Broadcast()
}
