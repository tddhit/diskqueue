package client

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	etcd "github.com/coreos/etcd/clientv3"

	"github.com/tddhit/diskqueue/types"
	"github.com/tddhit/tools/log"
	"github.com/tddhit/wox/naming"
)

type Consumer struct {
	mtx sync.RWMutex

	ec       *etcd.Client
	topic    string
	channel  string
	messages chan *types.Message
	conn     map[string]*Conn
	conns    []string
	addrs    []string
	msgid    map[string]string

	wg          sync.WaitGroup
	counter     uint64
	stopFlag    int32
	stopHandler sync.Once
	exitHandler sync.Once
	subHandler  sync.Once

	StopChan chan struct{}
	exitChan chan struct{}
	subChan  chan struct{}
}

func NewConsumer(
	c *etcd.Client,
	registry string,
	topic string,
	channel string) (*Consumer, error) {
	r := &Consumer{
		ec:       c,
		topic:    topic,
		channel:  channel,
		messages: make(chan *types.Message),
		conn:     make(map[string]*Conn),
		msgid:    make(map[string]string),

		StopChan: make(chan struct{}),
		exitChan: make(chan struct{}),
		subChan:  make(chan struct{}),
	}

	resolver := &naming.Resolver{
		Client:  c,
		Timeout: 2 * time.Second,
	}
	addrs := resolver.Resolve(registry)

	for _, addr := range addrs {
		key := fmt.Sprintf("/msgid/%s/%s/%s", topic, channel, addr)
		ctx, _ := context.WithTimeout(context.Background(), 2*time.Second)
		rsp, err := c.Get(ctx, key)
		if err != nil {
			return nil, err
		}
		msgid := "0"
		if len(rsp.Kvs) > 0 {
			msgid = string(rsp.Kvs[0].Value)
		}
		if err = r.Connect(addr, msgid); err != nil {
			log.Error(err)
		}
	}
	go r.commitMsgid()

	return r, nil
}

func (r *Consumer) Connect(addr, msgid string) error {
	if atomic.LoadInt32(&r.stopFlag) == 1 {
		return errors.New("consumer already stopped")
	}

	conn := NewConn(addr, &consumerConnDelegate{r})
	if err := conn.Connect(); err != nil {
		conn.Close()
		return err
	}
	log.Infof("connecting to diskqueue (%s)", addr)
	cmd := Subscribe(r.topic, r.channel, msgid)
	if err := conn.WriteCommand(cmd); err != nil {
		conn.Close()
		return fmt.Errorf("[%s] failed to subscribe to %s:%s - %s",
			conn, r.topic, r.channel, err.Error())
	}

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

func (r *Consumer) Disconnect(addr string) error {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	idx := indexOf(addr, r.addrs)
	if idx == -1 {
		return ErrNotConnected
	}
	r.addrs = append(r.addrs[:idx], r.addrs[idx+1:]...)
	delete(r.msgid, addr)
	if conn, ok := r.conn[addr]; ok {
		conn.Close()
	}
	return nil
}

func (r *Consumer) onConnMessage(c *Conn, msg *types.Message) {
	r.messages <- msg
}

func (r *Consumer) onConnResponse(c *Conn, data []byte) {
	str := string(data)
	switch {
	case str == "OK":
	case str == "CLOSE_WAIT":
		log.Infof("(%s) received CLOSE_WAIT from diskqueue", c.String())
		c.Close()
	case strings.HasPrefix(str, "SUB_OK-"):
		msgid := strings.Split(str, "-")[1]
		r.mtx.Lock()
		r.conn[c.String()] = c
		r.msgid[c.String()] = msgid
		r.conns = append(r.conns, c.String())
		r.addrs = append(r.addrs, c.String())
		r.mtx.Unlock()
		r.subHandler.Do(func() {
			close(r.subChan)
		})
	}
}

func (r *Consumer) onConnError(c *Conn, data []byte) {
}

func (r *Consumer) onConnIOError(c *Conn, err error) {
	c.Close()
}

func (r *Consumer) onConnClose(c *Conn) {
	r.mtx.Lock()
	delete(r.conn, c.String())
	idx := indexOf(c.String(), r.conns)
	r.conns = append(r.conns[:idx], r.conns[idx+1:]...)
	left := len(r.conns)
	r.mtx.Unlock()

	if atomic.LoadInt32(&r.stopFlag) == 1 {
		if left == 0 {
			r.stopHandlers()
		}
		return
	}

	r.mtx.RLock()
	reconnect := indexOf(c.String(), r.addrs) >= 0
	r.mtx.RUnlock()

	if reconnect {
		go func(addr string) {
			for {
				log.Infof("(%s) re-connecting", addr)
				time.Sleep(1 * time.Second)
				if atomic.LoadInt32(&r.stopFlag) == 1 {
					break
				}
				r.mtx.RLock()
				reconnect := indexOf(addr, r.addrs) >= 0
				msgid, ok := r.msgid[addr]
				r.mtx.RUnlock()
				if !reconnect || !ok {
					log.Infof("(%s) skipped reconnect", addr)
					return
				}
				err := r.Connect(addr, msgid)
				if err != nil && err != ErrAlreadyConnected {
					log.Errorf("(%s) error connecting to diskqueue - %s", addr, err)
					continue
				}
				break
			}
		}(c.String())
	}
}

func (r *Consumer) commitMsgid() {
	ticker := time.NewTicker(2 * time.Second)
	for range ticker.C {
		if atomic.LoadInt32(&r.stopFlag) == 1 {
			break
		}
		conns := r.connections()
		msgid := r.copyMsgid()
		log.Error(msgid)
		for _, conn := range conns {
			key := fmt.Sprintf("/msgid/%s/%s/%s", r.topic, r.channel, conn.String())
			ctx, _ := context.WithTimeout(context.Background(), 100*time.Millisecond)
			_, err := r.ec.Put(ctx, key, msgid[conn.String()])
			if err != nil {
				log.Error(err)
			}
		}
	}
}

func (r *Consumer) Pull() *types.Message {
	<-r.subChan
	counter := atomic.AddUint64(&r.counter, 1)

	r.mtx.RLock()
	index := counter % uint64(len(r.conns))
	addr := r.conns[index]
	conn, _ := r.conn[addr]
	r.mtx.RUnlock()

	cmd := Pull(r.topic, r.channel)
	if err := conn.WriteCommand(cmd); err != nil {
		log.Error(err)
		return nil
	}
	msg := <-r.messages

	r.mtx.Lock()
	r.msgid[conn.String()] = strconv.FormatUint(msg.Id+1, 10)
	r.mtx.Unlock()

	return msg
}

func (r *Consumer) connections() []*Conn {
	r.mtx.RLock()
	conns := make([]*Conn, 0, len(r.conn))
	for _, c := range r.conn {
		conns = append(conns, c)
	}
	r.mtx.RUnlock()
	return conns
}

func (r *Consumer) copyMsgid() map[string]string {
	r.mtx.RLock()
	defer r.mtx.RUnlock()

	msgid := make(map[string]string)
	for k, v := range r.msgid {
		msgid[k] = v
	}
	return msgid
}

func (r *Consumer) Stop() {
	if !atomic.CompareAndSwapInt32(&r.stopFlag, 0, 1) {
		return
	}

	if len(r.connections()) == 0 {
		r.stopHandlers()
	} else {
		for _, conn := range r.connections() {
			err := conn.WriteCommand(StartClose())
			if err != nil {
				log.Errorf("(%s) error sending CLS - %s", conn.String(), err)
			}
		}

		time.AfterFunc(time.Second*30, func() {
			r.exit()
		})
	}
}

func (r *Consumer) stopHandlers() {
	r.stopHandler.Do(func() {
		close(r.messages)
	})
}

func (r *Consumer) exit() {
	r.exitHandler.Do(func() {
		close(r.exitChan)
		r.wg.Wait()
		close(r.StopChan)
	})
}
