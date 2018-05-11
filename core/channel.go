package core

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tddhit/diskqueue/types"
	"github.com/tddhit/tools/log"
)

var (
	ErrAlreadyExit = errors.New("already exit")
)

type Consumer interface {
	Close() error
}

type Channel struct {
	sync.RWMutex

	name     string
	topic    *Topic
	client   Consumer
	readChan chan *types.Message

	count    int32
	exitFlag int32
	exitChan chan struct{}
}

func NewChannel(
	channelName string,
	topic *Topic,
	msgid uint64) (*Channel, error) {

	c := &Channel{
		name:     channelName,
		topic:    topic,
		readChan: make(chan *types.Message),

		exitChan: make(chan struct{}),
	}
	go c.readLoop(msgid)
	return c, nil
}

func (c *Channel) readLoop(msgid uint64) {
	var (
		seg *segment
		pos uint32
		msg *types.Message
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
		curMsgid := atomic.LoadUint64(&c.topic.msgid)
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
	return nil
}

func (c *Channel) RemoveClient() {
	c.topic.RemoveChannel(c.name)
}

func (c *Channel) GetMessage() *types.Message {
	return <-c.readChan
}

func (c *Channel) close() error {
	return c.exit()
}

func (c *Channel) exit() error {
	if !atomic.CompareAndSwapInt32(&c.exitFlag, 0, 1) {
		return ErrAlreadyExit
	}

	c.client.Close()
	return nil
}
