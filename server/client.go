package diskqueue

import (
	"bufio"
	"net"
	"sync"
	"sync/atomic"
	"time"

	. "github.com/tddhit/diskqueue/core"
)

const defaultBufferSize = 16 * 1024

const (
	stateInit = iota
	stateSubscribed
	stateClosing
)

type client struct {
	net.Conn

	MessageCount uint64
	FinishCount  uint64

	writeLock sync.RWMutex
	metaLock  sync.RWMutex

	ID     int64
	Reader *bufio.Reader
	Writer *bufio.Writer

	HeartbeatInterval time.Duration
	MsgTimeout        time.Duration
	ConnectTime       time.Time

	State    int32
	Channel  *Channel
	ClientID string
	Hostname string
	lenBuf   [4]byte
	lenSlice []byte

	ExitChan chan int
}

func newClient(id int64, conn net.Conn) *client {
	var identifier string
	if conn != nil {
		identifier, _, _ = net.SplitHostPort(conn.RemoteAddr().String())
	}
	c := &client{
		ID:     id,
		Conn:   conn,
		Reader: bufio.NewReaderSize(conn, defaultBufferSize),
		Writer: bufio.NewWriterSize(conn, defaultBufferSize),

		HeartbeatInterval: 18 * time.Second,
		ExitChan:          make(chan int),
		ConnectTime:       time.Now(),
		State:             stateInit,
		ClientID:          identifier,
		Hostname:          identifier,
	}
	c.lenSlice = c.lenBuf[:]
	return c
}

func (c *client) String() string {
	return c.RemoteAddr().String()
}

func (c *client) StartClose() {
	atomic.StoreInt32(&c.State, stateClosing)
}

func (c *client) Flush() error {
	var zeroTime time.Time
	if c.HeartbeatInterval > 0 {
		c.SetWriteDeadline(time.Now().Add(c.HeartbeatInterval))
	} else {
		c.SetWriteDeadline(zeroTime)
	}
	if err := c.Writer.Flush(); err != nil {
		return err
	}
	return nil
}
