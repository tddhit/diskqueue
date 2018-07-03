package core

import (
	"errors"
	"expvar"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	pb "github.com/tddhit/diskqueue/pb"
	"github.com/tddhit/tools/log"
)

var (
	ErrAlreadyExit = errors.New("already exit")
)

type Consumer interface {
	String() string
}

type Channel struct {
	sync.RWMutex

	name     string
	topic    *Topic
	client   Consumer
	readChan chan *pb.Message

	outQueue *expvar.Int

	exitFlag int32
	exitChan chan struct{}
}

func NewChannel(
	channelName string,
	topic *Topic,
	msgid uint64) *Channel {

	c := &Channel{
		name:     channelName,
		topic:    topic,
		readChan: make(chan *pb.Message),

		exitChan: make(chan struct{}),
	}
	go c.readLoop(msgid)
	return c
}

func (c *Channel) readLoop(msgid uint64) {
	var (
		seg *segment
		pos uint32
		msg *pb.Message
		err error
	)
	for {
		if atomic.LoadInt32(&c.exitFlag) == 1 {
			goto exit
		}
		if seg == nil {
			seg, pos, err = c.topic.seek(msgid)
			if err != nil {
				log.Fatal(err)
			}
		}
		curMsgid := atomic.LoadUint64((*uint64)(unsafe.Pointer(&c.topic.msgid)))
		if msgid < curMsgid {
			segMaxMsgid := atomic.LoadUint64(&seg.maxMsgid)
			if msgid < segMaxMsgid {
				msg, pos, err = seg.readOne(msgid, pos)
				if err != nil {
					log.Fatal(err)
				}
			} else {
				seg = nil
				continue
			}
		} else {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		select {
		case c.readChan <- msg:
			msgid++
			c.outQueue.Add(1)
		case <-c.exitChan:
			log.Info("receive exitChan")
			goto exit
		}
	}
exit:
	close(c.readChan)
	log.Infof("channel(%s:%s) exit readLoop.", c.topic.name, c.name)
}

func (c *Channel) AddClient(client Consumer) error {
	c.Lock()
	defer c.Unlock()

	if c.client != nil {
		return errors.New("already subscribe")
	}
	c.client = client
	c.outQueue = expvar.NewInt(fmt.Sprintf("%s-%s-%s-outQueue",
		c.topic.name, c.name, c.client))
	return nil
}

func (c *Channel) RemoveClient() {
	c.topic.RemoveChannel(c.name)
}

func (c *Channel) GetMessage() *pb.Message {
	return <-c.readChan
}

func (c *Channel) close() error {
	return c.exit()
}

func (c *Channel) exit() error {
	if !atomic.CompareAndSwapInt32(&c.exitFlag, 0, 1) {
		return ErrAlreadyExit
	}
	return nil
}
