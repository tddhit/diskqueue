package service

import (
	"sync"

	"github.com/tddhit/diskqueue/store"
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

func (c *client) getOrCreateInflight(t *store.Topic) *inflight {
	c.RLock()
	f, ok := c.inflights[t.Name]
	if ok {
		c.RUnlock()
		return f
	}
	c.RUnlock()

	c.Lock()
	f, ok = c.inflights[t.Name]
	if ok {
		c.Unlock()
		return f

	}
	f = newInflight(t)
	c.inflights[t.Name] = f
	c.Unlock()
	return f
}
