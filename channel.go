package diskqueue

import (
	"errors"
	"sync"
	"sync/atomic"
)

type Consumer interface {
	Close() error
}

type Channel struct {
	name      string
	topicName string
	clients   sync.Map
	readChan  <-chan *Message

	exitFlag int32
}

func NewChannel(topicName, channelName string, readChan <-chan *Message) *Channel {
	c := &Channel{
		name:      channelName,
		topicName: topicName,
		readChan:  readChan,
	}
	return c
}

func (c *Channel) Close() error {
	return c.exit()
}

func (c *Channel) exit() error {
	if !atomic.CompareAndSwapInt32(&c.exitFlag, 0, 1) {
		return errors.New("exiting")
	}

	c.clients.Range(func(key, value interface{}) bool {
		value.(Consumer).Close()
		return true
	})
	return nil
}

func (c *Channel) AddClient(clientID int64, client Consumer) {
	c.clients.LoadOrStore(clientID, client)
}

func (c *Channel) RemoveClient(clientID int64) {
	c.clients.Delete(clientID)
}

func (c *Channel) Get() *Message {
	return <-c.readChan
}
