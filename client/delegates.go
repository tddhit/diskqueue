package client

type ConnDelegate interface {
	OnResponse(*Conn, []byte)
	OnMessage(*Conn, []byte)
}

type consumerConnDelegate struct {
	r *Consumer
}

func (d *consumerConnDelegate) OnResponse(c *Conn, data []byte) { d.r.onConnResponse(c, data) }
func (d *consumerConnDelegate) OnMessage(c *Conn, m []byte)     { d.r.onConnMessage(c, m) }

type producerConnDelegate struct {
	w *Producer
}

func (d *producerConnDelegate) OnResponse(c *Conn, data []byte) { d.w.onConnResponse(c, data) }
func (d *producerConnDelegate) OnMessage(c *Conn, m []byte)     {}
