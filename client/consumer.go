package client

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/tddhit/tools/log"
)

type Consumer struct {
	sync.RWMutex

	addr             string
	topic            string
	channel          string
	conn             *Conn
	incomingMessages chan []byte

	wg sync.WaitGroup
}

func NewConsumer(topic string, channel string) *Consumer {
	r := &Consumer{
		topic:            topic,
		channel:          channel,
		incomingMessages: make(chan []byte),
	}
	return r
}

func (r *Consumer) Connect(addr, msgid string) error {
	r.addr = addr
	conn := NewConn(addr, &consumerConnDelegate{r})
	log.Infof("(%s) connecting to nsqd", addr)
	err := conn.Connect()
	if err != nil {
		return err
	}
	cmd := Subscribe(r.topic, r.channel, msgid)
	if err = conn.WriteCommand(cmd); err != nil {
		return fmt.Errorf("[%s] failed to subscribe to %s:%s - %s",
			conn, r.topic, r.channel, err.Error())
	}
	r.conn = conn
	return nil
}

func indexOf(n string, h []string) int {
	for i, a := range h {
		if n == a {
			return i
		}
	}
	return -1
}

func (r *Consumer) onConnMessage(c *Conn, msg []byte) {
	r.incomingMessages <- msg
}

func (r *Consumer) onConnResponse(c *Conn, data []byte) {
	switch {
	case bytes.Equal(data, []byte("CLOSE_WAIT")):
		log.Infof("(%s) received CLOSE_WAIT from nsqd", c.String())
	}
}

func (r *Consumer) onConnError(c *Conn, data []byte) {}

func (r *Consumer) Pull() []byte {
	cmd := Pull(r.topic, r.channel)
	if err := r.conn.WriteCommand(cmd); err != nil {
		log.Error(err)
		return nil
	}
	log.Debug("pull")
	return <-r.incomingMessages
}
