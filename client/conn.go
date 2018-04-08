package client

import (
	"bytes"
	"io"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tddhit/tools/log"
)

type Conn struct {
	mtx             sync.Mutex
	conn            *net.TCPConn
	addr            string
	delegate        ConnDelegate
	r               io.Reader
	w               io.Writer
	cmdChan         chan *Command
	wg              sync.WaitGroup
	readLoopRunning int32
}

func NewConn(addr string, delegate ConnDelegate) *Conn {
	return &Conn{
		addr:     addr,
		delegate: delegate,
		cmdChan:  make(chan *Command),
	}
}

func (c *Conn) Connect() error {
	dialer := &net.Dialer{}
	conn, err := dialer.Dial("tcp", c.addr)
	if err != nil {
		return err
	}
	c.conn = conn.(*net.TCPConn)
	c.r = conn
	c.w = conn
	c.wg.Add(2)
	atomic.StoreInt32(&c.readLoopRunning, 1)
	go c.readLoop()
	go c.writeLoop()
	return nil
}

func (c *Conn) String() string {
	return c.addr
}

func (c *Conn) Read(p []byte) (int, error) {
	return c.r.Read(p)
}

func (c *Conn) Write(p []byte) (int, error) {
	return c.w.Write(p)
}

func (c *Conn) WriteCommand(cmd *Command) error {
	c.mtx.Lock()
	_, err := cmd.WriteTo(c)
	if err != nil {
		goto exit
	}
	err = c.Flush()
exit:
	c.mtx.Unlock()
	if err != nil {
		log.Errorf("IO error - %s", err)
	}
	return err
}

type flusher interface {
	Flush() error
}

func (c *Conn) Flush() error {
	if f, ok := c.w.(flusher); ok {
		return f.Flush()
	}
	return nil
}

func (c *Conn) readLoop() {
	for {
		frameType, data, err := ReadUnpackedResponse(c)
		if err != nil {
			if !strings.Contains(err.Error(), "use of closed network connection") {
				log.Errorf("IO error - %s", err)
			}
			goto exit
		}
		if frameType == FrameTypeResponse && bytes.Equal(data, []byte("_heartbeat_")) {
			log.Debug("heartbeat received")
			if err := c.WriteCommand(Nop()); err != nil {
				log.Errorf("IO error - %s", err)
				goto exit
			}
			continue
		}
		switch frameType {
		case FrameTypeResponse:
			c.delegate.OnResponse(c, data)
		case FrameTypeMessage:
			c.delegate.OnMessage(c, data)
		case FrameTypeError:
			log.Errorf("protocol error - %s", data)
		default:
			log.Errorf("IO error - %s", err)
		}
	}
exit:
	atomic.StoreInt32(&c.readLoopRunning, 0)
	c.wg.Done()
	log.Info("readLoop exiting")
}

func (c *Conn) writeLoop() {
	for {
		select {
		case cmd := <-c.cmdChan:
			if err := c.WriteCommand(cmd); err != nil {
				log.Errorf("error sending command %s - %s", cmd, err)
				continue
			}
		}
	}

	c.wg.Done()
	log.Info("writeLoop exiting")
}

func (c *Conn) cleanup() {
	ticker := time.NewTicker(100 * time.Millisecond)
	lastWarning := time.Now()
	for {
		if atomic.LoadInt32(&c.readLoopRunning) == 1 {
			if time.Now().Sub(lastWarning) > time.Second {
				log.Warnf("draining... readLoop still running")
				lastWarning = time.Now()
			}
			continue
		}
		goto exit
	}

exit:
	ticker.Stop()
	c.wg.Done()
	log.Info("finished draining, cleanup exiting")
}

func (c *Conn) waitForCleanup() {
	c.wg.Wait()
	c.conn.CloseWrite()
	log.Info("clean close complete")
}
