package diskqueue

import (
	"errors"
	"sync"
	"sync/atomic"
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
	clients  sync.Map
	readChan <-chan *Message

	count    int32
	exitFlag int32
}

func NewChannel(channelName string, topic *Topic, readChan <-chan *Message) *Channel {
	c := &Channel{
		name:     channelName,
		topic:    topic,
		readChan: readChan,
	}
	return c
}

func (c *Channel) Close() error {
	return c.exit()
}

func (c *Channel) exit() error {
	if !atomic.CompareAndSwapInt32(&c.exitFlag, 0, 1) {
		return ErrAlreadyExit
	}

	c.clients.Range(func(key, value interface{}) bool {
		value.(Consumer).Close()
		return true
	})

	return nil
}

func (c *Channel) AddClient(clientID int64, client Consumer) {
	atomic.AddInt32(&c.count, 1)
	c.clients.LoadOrStore(clientID, client)
}

func (c *Channel) RemoveClient(clientID int64) {
	count := atomic.AddInt32(&c.count, -1)
	c.clients.Delete(clientID)
	if count == 0 {
		c.topic.RemoveChannel(c.name)
	}
}

func (c *Channel) GetMessage() *Message {
	//c.RLock()
	//defer c.RUnlock()

	return <-c.readChan
}
