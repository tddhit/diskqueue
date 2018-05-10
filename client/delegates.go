package client

import "github.com/tddhit/diskqueue/types"

type ConnDelegate interface {
	OnResponse(*Conn, []byte)
	OnError(*Conn, []byte)
	OnMessage(*Conn, *types.Message)
	OnIOError(*Conn, error)
	OnClose(*Conn)
}

type consumerConnDelegate struct {
	r *Consumer
}

func (d *consumerConnDelegate) OnResponse(c *Conn, data []byte)     { d.r.onConnResponse(c, data) }
func (d *consumerConnDelegate) OnError(c *Conn, data []byte)        { d.r.onConnError(c, data) }
func (d *consumerConnDelegate) OnMessage(c *Conn, m *types.Message) { d.r.onConnMessage(c, m) }
func (d *consumerConnDelegate) OnIOError(c *Conn, err error)        { d.r.onConnIOError(c, err) }
func (d *consumerConnDelegate) OnClose(c *Conn)                     { d.r.onConnClose(c) }

type producerConnDelegate struct {
	w *producer
}

func (d *producerConnDelegate) OnResponse(c *Conn, data []byte)     { d.w.onConnResponse(c, data) }
func (d *producerConnDelegate) OnError(c *Conn, data []byte)        { d.w.onConnError(c, data) }
func (d *producerConnDelegate) OnMessage(c *Conn, m *types.Message) {}
func (d *producerConnDelegate) OnIOError(c *Conn, err error)        { d.w.onConnIOError(c, err) }
func (d *producerConnDelegate) OnClose(c *Conn)                     { d.w.onConnClose(c) }
