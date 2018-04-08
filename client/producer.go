package client

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/tddhit/tools/log"
)

type producerConn interface {
	String() string
	Connect() error
	WriteCommand(*Command) error
}

type Producer struct {
	addr                string
	state               int32
	concurrentProducers int32

	conn            producerConn
	transactions    []*ProducerTransaction
	transactionChan chan *ProducerTransaction
	responseChan    chan []byte

	errorChan chan []byte
	closeChan chan struct{}
	exitChan  chan struct{}
	stopFlag  int32
	wg        sync.WaitGroup
	guard     sync.Mutex
}

type ProducerTransaction struct {
	cmd      *Command
	doneChan chan *ProducerTransaction
	Error    error
	Args     []interface{}
}

func (t *ProducerTransaction) finish() {
	if t.doneChan != nil {
		t.doneChan <- t
	}
}

func NewProducer(addr string) *Producer {
	p := &Producer{
		addr:            addr,
		transactionChan: make(chan *ProducerTransaction),
		responseChan:    make(chan []byte),
		errorChan:       make(chan []byte),
	}
	return p
}

func (w *Producer) Publish(topic string, body []byte) error {
	return w.sendCommand(Publish(topic, body))
}

func (w *Producer) sendCommand(cmd *Command) error {
	doneChan := make(chan *ProducerTransaction)
	if err := w.sendCommandAsync(cmd, doneChan, nil); err != nil {
		close(doneChan)
		return err
	}
	t := <-doneChan
	return t.Error
}

func (w *Producer) sendCommandAsync(cmd *Command, doneChan chan *ProducerTransaction,
	args []interface{}) error {
	atomic.AddInt32(&w.concurrentProducers, 1)
	defer atomic.AddInt32(&w.concurrentProducers, -1)

	if atomic.LoadInt32(&w.state) != StateConnected {
		if err := w.connect(); err != nil {
			return err
		}
	}
	t := &ProducerTransaction{
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

func (w *Producer) connect() error {
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
		log.Errorf("(%s) error connecting to nsqd - %s", w.addr, err)
		return err
	}
	atomic.StoreInt32(&w.state, StateConnected)
	w.closeChan = make(chan struct{})
	w.wg.Add(1)
	go w.router()
	return nil
}

func (w *Producer) router() {
	for {
		select {
		case t := <-w.transactionChan:
			w.transactions = append(w.transactions, t)
			if err := w.conn.WriteCommand(t.cmd); err != nil {
				log.Errorf("(%s) sending command - %s", w.conn.String(), err)
			}
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

func (w *Producer) popTransaction(frameType int32, data []byte) {
	t := w.transactions[0]
	w.transactions = w.transactions[1:]
	if frameType == FrameTypeError {
		t.Error = ErrProtocol{string(data)}
	}
	t.finish()
}

func (w *Producer) transactionCleanup() {
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
			if atomic.LoadInt32(&w.concurrentProducers) == 0 {
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
	}
}

func (w *Producer) onConnResponse(c *Conn, data []byte) { w.responseChan <- data }
func (w *Producer) onConnError(c *Conn, data []byte)    { w.errorChan <- data }
func (w *Producer) onConnHeartbeat(c *Conn)             {}
func (w *Producer) onConnIOError(c *Conn, err error)    {}
func (w *Producer) onConnClose(c *Conn) {
	w.guard.Lock()
	defer w.guard.Unlock()
	close(w.closeChan)
}
