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
	seg, pos, err := c.topic.seek(msgid)
	if err != nil {
		return nil, err
	}
	go c.readLoop(seg, pos, msgid)
	return c, nil
}

func (c *Channel) readLoop(seg *segment, pos uint32, msgid uint64) {
	var msg *types.Message
	var err error
	for {
		if atomic.LoadInt32(&c.exitFlag) == 1 {
			goto exit
		}
		curMsgid := atomic.LoadUint64(&c.topic.msgid)
		if msgid < curMsgid {
			msg, pos, err = seg.readOne(msgid, pos)
			if err != nil {
				log.Fatal(err)
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
