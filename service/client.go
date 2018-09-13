package service

import (
	"sync"
)

type client struct {
	sync.RWMutex
	addr      string
	inflights map[string]*inflight
}

func (c *client) String() string {
	return c.addr
}

func (c *client) getInflight(topic string) (*inflight, bool) {
	c.RLock()
	defer c.RUnlock()

	f, ok := c.inflights[topic]
	return f, ok
}

func (c *client) getOrCreateInflight(topic string, q queue) *inflight {
	c.RLock()
	f, ok := c.inflights[topic]
	if ok {
		c.RUnlock()
		return f
	}
	c.RUnlock()

	c.Lock()
	f, ok = c.inflights[topic]
	if ok {
		c.Unlock()
		return f

	}
	f = newInflight(topic, q)
	c.inflights[topic] = f
	c.Unlock()
	return f
}
