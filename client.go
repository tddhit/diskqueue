package diskqueue

import (
	"bufio"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

const defaultBufferSize = 16 * 1024

const (
	stateInit = iota
	stateDisconnected
	stateConnected
	stateSubscribed
	stateClosing
)

type identifyDataV2 struct {
	ClientID          string `json:"client_id"`
	Hostname          string `json:"hostname"`
	HeartbeatInterval int    `json:"heartbeat_interval"`
	MsgTimeout        int    `json:"msg_timeout"`
}

type identifyEvent struct {
	HeartbeatInterval time.Duration
	MsgTimeout        time.Duration
}

type clientV2 struct {
	net.Conn

	MessageCount uint64
	FinishCount  uint64

	writeLock sync.RWMutex
	metaLock  sync.RWMutex

	ID     int64
	ctx    *context
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

	ReadyStateChan    chan int
	ExitChan          chan int
	IdentifyEventChan chan identifyEvent
	SubEventChan      chan *Channel
}

func newClientV2(id int64, conn net.Conn, ctx *context) *clientV2 {
	var identifier string
	if conn != nil {
		identifier, _, _ = net.SplitHostPort(conn.RemoteAddr().String())
	}
	c := &clientV2{
		ID:     id,
		ctx:    ctx,
		Conn:   conn,
		Reader: bufio.NewReaderSize(conn, defaultBufferSize),
		Writer: bufio.NewWriterSize(conn, defaultBufferSize),

		ExitChan:     make(chan int),
		ConnectTime:  time.Now(),
		State:        stateInit,
		ClientID:     identifier,
		Hostname:     identifier,
		SubEventChan: make(chan *Channel, 1),
	}
	c.lenSlice = c.lenBuf[:]
	return c
}

func (c *clientV2) String() string {
	return c.RemoteAddr().String()
}

func (c *clientV2) StartClose() {
	atomic.StoreInt32(&c.State, stateClosing)
}

func (c *clientV2) Flush() error {
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
