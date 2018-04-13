package client

import (
	"bytes"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	etcd "github.com/coreos/etcd/clientv3"
	"github.com/tddhit/tools/log"
	"github.com/tddhit/wox/naming"
)

type Consumer struct {
	mtx sync.RWMutex

	topic            string
	channel          string
	conns            []*Conn
	incomingMessages chan []byte

	wg          sync.WaitGroup
	counter     uint64
	stopFlag    int32
	stopHandler sync.Once
	exitHandler sync.Once

	StopChan chan struct{}
	exitChan chan struct{}
}

func NewConsumer(topic string, channel string) *Consumer {
	r := &Consumer{
		topic:            topic,
		channel:          channel,
		incomingMessages: make(chan []byte),

		StopChan: make(chan struct{}),
		exitChan: make(chan struct{}),
	}
	return r
}

func (r *Consumer) ConnectByEtcd(c *etcd.Client, key, msgid string) {
	resolver := &naming.Resolver{
		Client:  c,
		Timeout: 2000 * time.Millisecond,
	}
	addrs := resolver.Resolve(key)
	log.Info("Addrs:", addrs)
	for _, addr := range addrs {
		conn, err := r.Connect(addr, msgid)
		if err != nil {
			log.Error(err)
			continue
		}
		r.conns = append(r.conns, conn)
	}
}

func (r *Consumer) Connect(addr, msgid string) (*Conn, error) {
	if atomic.LoadInt32(&r.stopFlag) == 1 {
		return nil, errors.New("consumer stopped")
	}
	conn := NewConn(addr, &consumerConnDelegate{r})
	log.Infof("(%s) connecting to nsqd", addr)
	err := conn.Connect()
	if err != nil {
		conn.Close()
		return nil, err
	}
	cmd := Subscribe(r.topic, r.channel, msgid)
	if err = conn.WriteCommand(cmd); err != nil {
		conn.Close()
		return nil, fmt.Errorf("[%s] failed to subscribe to %s:%s - %s",
			conn, r.topic, r.channel, err.Error())
	}
	return conn, nil
}

func (r *Consumer) Disconnect() error {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	for _, conn := range r.conns {
		conn.Close()
	}
	r.conns = make([]*Conn, 0)
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
	counter := atomic.AddUint64(&r.counter, 1)
	index := counter % uint64(len(r.conns))
	conn := r.conns[index]
	cmd := Pull(r.topic, r.channel)
	if err := conn.WriteCommand(cmd); err != nil {
		log.Error(err)
		return nil
	}
	return <-r.incomingMessages
}

func (r *Consumer) Stop() {
	if !atomic.CompareAndSwapInt32(&r.stopFlag, 0, 1) {
		return
	}

	for _, conn := range r.conns {
		err := conn.WriteCommand(StartClose())
		if err != nil {
			log.Errorf("(%s) error sending CLS - %s", conn.String(), err)
		}
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
