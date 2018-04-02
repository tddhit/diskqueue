package diskqueue

import (
	"errors"
	"os"
	"sort"
	"sync"
	"sync/atomic"

	"github.com/tddhit/tools/log"
)

type DiskQueue struct {
	sync.RWMutex
	name   string
	curSeg *segment
	segs   segments
	offset uint64
}

func NewDiskQueue(topic string) *DiskQueue {
	q := &DiskQueue{
		name: topic,
	}
	return q
}

func (q *DiskQueue) Put(data []byte) error {
	q.Lock()
	defer q.Unlock()
	offset := atomic.LoadUint64(&q.offset)
	if q.curSeg == nil || q.curSeg.full() {
		seg, err := newSegment(q.name, offset, os.O_CREATE|os.O_RDWR|os.O_APPEND)
		if err != nil {
			return err
		}
		q.segs = append(q.segs, seg)
		q.curSeg = seg
	}
	if err := q.curSeg.writeOne(offset, data); err != nil {
		log.Error(err)
		return err
	}
	atomic.AddUint64(&q.offset, 1)
	return nil
}

func (q *DiskQueue) Pop(offset uint64) ([]byte, error) {
	q.Lock()
	defer q.Unlock()
	if len(q.segs) == 0 {
		return nil, errors.New("not found offset")
	}
	index := sort.Search(len(q.segs), func(i int) bool {
		return q.segs[i].minOffset >= offset
	})
	if index == 0 && q.segs[0].minOffset > offset {
		return nil, errors.New("not found offset")
	} else if index == 0 {
		index = 1
	}
	seg := q.segs[index-1]
	return seg.readOne(offset)
}
