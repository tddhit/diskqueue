package client

import (
	"bytes"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tddhit/tools/log"
)

type Consumer struct {
	mtx sync.RWMutex

	topic   string
	channel string
	conn    *Conn

	incomingMessages chan []byte

	wg            sync.WaitGroup
	stopFlag      int32
	connectedFlag int32
	stopHandler   sync.Once
	exitHandler   sync.Once

	StopChan chan struct{}
	exitChan chan struct{}
}

func NewConsumer(topic string, channel string) *Consumer {
	r := &Consumer{
		topic:   topic,
		channel: channel,

		incomingMessages: make(chan []byte),

		StopChan: make(chan struct{}),
		exitChan: make(chan struct{}),
	}
	return r
}

func (r *Consumer) Connect(addr, msgid string) error {
	if atomic.LoadInt32(&r.stopFlag) == 1 {
		return errors.New("consumer stopped")
	}
	atomic.StoreInt32(&r.connectedFlag, 1)
	conn := NewConn(addr, &consumerConnDelegate{r})
	log.Infof("(%s) connecting to nsqd", addr)
	err := conn.Connect()
	if err != nil {
		conn.Close()
		return err
	}
	cmd := Subscribe(r.topic, r.channel, msgid)
	if err = conn.WriteCommand(cmd); err != nil {
		conn.Close()
		return fmt.Errorf("[%s] failed to subscribe to %s:%s - %s",
			conn, r.topic, r.channel, err.Error())
	}
	r.conn = conn
	return nil
}

func (r *Consumer) Disconnect() error {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	r.conn.Close()
	r.conn = nil
	return nil
}

func (r *Consumer) onConnMessage(c *Conn, msg []byte) {
	r.incomingMessages <- msg
}

func (r *Consumer) onConnResponse(c *Conn, data []byte) {
	switch {
	case bytes.Equal(data, []byte("CLOSE_WAIT")):
		log.Infof("(%s) received CLOSE_WAIT from nsqd", c.String())
		c.Close()
	}
}

func (r *Consumer) onConnError(c *Conn, data []byte) {}

func (r *Consumer) onConnIOError(c *Conn, err error) {
	c.Close()
}

func (r *Consumer) onConnClose(c *Conn) {
	if atomic.LoadInt32(&r.stopFlag) == 1 {
		r.stopHandlers()
		return
	}
}

func (r *Consumer) Pull() []byte {
	cmd := Pull(r.topic, r.channel)
	if err := r.conn.WriteCommand(cmd); err != nil {
		log.Error(err)
		return nil
	}
	log.Debug("pull")
	return <-r.incomingMessages
}

func (r *Consumer) Stop() {
	if !atomic.CompareAndSwapInt32(&r.stopFlag, 0, 1) {
		return
	}

	err := r.conn.WriteCommand(StartClose())
	if err != nil {
		log.Errorf("(%s) error sending CLS - %s", r.conn.String(), err)
	}

	time.AfterFunc(time.Second*30, func() {
		r.exit()
	})
}

func (r *Consumer) stopHandlers() {
	r.stopHandler.Do(func() {
		close(r.incomingMessages)
	})
}

func (r *Consumer) exit() {
	r.exitHandler.Do(func() {
		close(r.exitChan)
		r.wg.Wait()
		close(r.StopChan)
	})
}
