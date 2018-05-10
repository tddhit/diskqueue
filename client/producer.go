package client

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"

	etcd "github.com/coreos/etcd/clientv3"

	"github.com/tddhit/tools/log"
	"github.com/tddhit/wox/naming"
)

type producerConn interface {
	String() string
	Connect() error
	Close() error
	WriteCommand(*Command) error
}

type producer struct {
	addr string
	conn producerConn

	responseChan chan []byte
	errorChan    chan []byte
	closeChan    chan struct{}

	transactionChan chan *producerTransaction
	transactions    []*producerTransaction
	state           int32

	concurrentproducers int32
	stopFlag            int32
	exitChan            chan struct{}
	wg                  sync.WaitGroup
	guard               sync.Mutex
}

func (w *producer) String() string {
	return w.addr
}

type producerTransaction struct {
	cmd      *Command
	doneChan chan *producerTransaction
	Error    error
	Args     []interface{}
}

func (t *producerTransaction) finish() {
	if t.doneChan != nil {
		t.doneChan <- t
	}
}

func (w *producer) Stop() {
	w.guard.Lock()
	if !atomic.CompareAndSwapInt32(&w.stopFlag, 0, 1) {
		w.guard.Unlock()
		return
	}
	close(w.exitChan)
	w.close()
	w.guard.Unlock()
	w.wg.Wait()
}

func (w *producer) close() {
	if !atomic.CompareAndSwapInt32(&w.state, StateConnected, StateDisconnected) {
		return
	}
	w.conn.Close()
	go func() {
		w.wg.Wait()
		atomic.StoreInt32(&w.state, StateInit)
	}()
}

func (w *producer) Publish(topic string, body []byte) error {
	return w.sendCommand(Publish(topic, body))
}

func (w *producer) sendCommand(cmd *Command) error {
	doneChan := make(chan *producerTransaction)
	if err := w.sendCommandAsync(cmd, doneChan, nil); err != nil {
		close(doneChan)
		return err
	}
	t := <-doneChan
	return t.Error
}

func (w *producer) sendCommandAsync(cmd *Command, doneChan chan *producerTransaction,
	args []interface{}) error {
	atomic.AddInt32(&w.concurrentproducers, 1)
	defer atomic.AddInt32(&w.concurrentproducers, -1)

	if atomic.LoadInt32(&w.state) != StateConnected {
		if err := w.connect(); err != nil {
			return err
		}
	}
	t := &producerTransaction{
		cmd:      cmd,
		doneChan: doneChan,
		Args:     args,
	}
	select {
	case w.transactionChan <- t:
	case <-w.exitChan:
		return ErrStopped
	}
	return nil
}

func (w *producer) connect() error {
	w.guard.Lock()
	defer w.guard.Unlock()

	if atomic.LoadInt32(&w.stopFlag) == 1 {
		return ErrStopped
	}
	switch state := atomic.LoadInt32(&w.state); state {
	case StateInit:
	case StateConnected:
		return nil
	default:
		return ErrNotConnected
	}
	log.Infof("(%s) connecting to nsqd", w.addr)
	w.conn = NewConn(w.addr, &producerConnDelegate{w})
	if err := w.conn.Connect(); err != nil {
		w.conn.Close()
		log.Errorf("(%s) error connecting to nsqd - %s", w.addr, err)
		return err
	}
	atomic.StoreInt32(&w.state, StateConnected)
	w.closeChan = make(chan struct{})
	w.wg.Add(1)
	go w.router()
	return nil
}

func (w *producer) router() {
	for {
		select {
		case t := <-w.transactionChan:
			w.transactions = append(w.transactions, t)
			if err := w.conn.WriteCommand(t.cmd); err != nil {
				log.Errorf("(%s) sending command - %s", w.conn.String(), err)
				w.close()
			}
			log.Debug("WriteCommand", string(t.cmd.Body))
		case data := <-w.responseChan:
			w.popTransaction(FrameTypeResponse, data)
		case data := <-w.errorChan:
			w.popTransaction(FrameTypeError, data)
		case <-w.closeChan:
			goto exit
		case <-w.exitChan:
			goto exit
		}
	}
exit:
	w.transactionCleanup()
	w.wg.Done()
	log.Info("exiting router")
}

func (w *producer) popTransaction(frameType int32, data []byte) {
	t := w.transactions[0]
	w.transactions = w.transactions[1:]
	if frameType == FrameTypeError {
		t.Error = ErrProtocol{string(data)}
	}
	t.finish()
}

func (w *producer) transactionCleanup() {
	for _, t := range w.transactions {
		t.Error = ErrNotConnected
		t.finish()
	}
	w.transactions = w.transactions[:0]
	for {
		select {
		case t := <-w.transactionChan:
			t.Error = ErrNotConnected
			t.finish()
		default:
			if atomic.LoadInt32(&w.concurrentproducers) == 0 {
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
	}
}

func (w *producer) onConnResponse(c *Conn, data []byte) { w.responseChan <- data }
func (w *producer) onConnError(c *Conn, data []byte)    { w.errorChan <- data }
func (w *producer) onConnIOError(c *Conn, err error)    { w.close() }
func (w *producer) onConnClose(c *Conn) {
	w.guard.Lock()
	defer w.guard.Unlock()
	close(w.closeChan)
}

type Producer struct {
	producers []*producer
	counter   uint64
	wg        sync.WaitGroup
}

func NewProducer(
	c *etcd.Client,
	registry string) (p *Producer, err error) {

	resolver := &naming.Resolver{
		Client:  c,
		Timeout: 2 * time.Second,
	}
	addrs := resolver.Resolve(registry)

	p = &Producer{}
	for _, addr := range addrs {
		p.producers = append(p.producers, &producer{
			addr:            addr,
			transactionChan: make(chan *producerTransaction),
			exitChan:        make(chan struct{}),
			responseChan:    make(chan []byte),
			errorChan:       make(chan []byte),
		})
	}
	return
}

func (p *Producer) Publish(topic string, body []byte) (err error) {
	if len(p.producers) == 0 {
		err = errors.New("Unavailable Producer")
		return
	}
	counter := atomic.AddUint64(&p.counter, 1)
	index := counter % uint64(len(p.producers))
	producer := p.producers[index]
	retryTimes := 0
	if err = producer.Publish(topic, body); err != nil {
		for retryTimes < 2 {
			retryTimes++
			if err = producer.Publish(topic, body); err != nil {
				continue
			}
			break
		}
		if err != nil {
			log.Errorf("type=msgqueue\tvendor=diskqueue\taddr=%s\ttopic=%s\tmsg=%s\tretry=%d\terr=%s\n",
				producer, topic, string(body), retryTimes, err)
			return
		}
	}
	log.Infof("type=msgqueue\tvendor=diskqueue\taddr=%s\ttopic=%s\tmsg=%s\tretry=%d\n",
		producer, topic, string(body), retryTimes)
	return
}

func (p *Producer) Stop() {
	for _, producer := range p.producers {
		p.wg.Add(1)
		go func() {
			producer.Stop()
			p.wg.Done()
		}()
	}
	p.wg.Wait()
}
