package service

import "github.com/tddhit/diskqueue/store"

const (
	stateInit = iota
	stateSubscribed
	stateClosing
)

type Client struct {
	State int32
	Addr  string
	topic *store.Topic
}

func (c *Client) String() string {
	return c.Addr
}
