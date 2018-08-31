package service

import "github.com/tddhit/diskqueue/store"

const (
	stateInit = iota
	stateSubscribed
	stateClosing
)

type client struct {
	state int32
	addr  string
	topic *store.Topic
}

func (c *client) String() string {
	return c.addr
}
