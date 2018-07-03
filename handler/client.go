package handler

import (
	"github.com/tddhit/diskqueue/core"
)

const (
	stateInit = iota
	stateSubscribed
	stateClosing
)

type Client struct {
	State   int32
	Addr    string
	Channel *core.Channel
}

func (c *Client) String() string {
	return c.Addr
}
