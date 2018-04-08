package client

import (
	"testing"

	"github.com/tddhit/tools/log"
)

func TestConsumer(t *testing.T) {
	c := NewConsumer("topic1", "channel2")
	err := c.Connect("127.0.0.1:18800", "0")
	if err != nil {
		log.Fatal(err)
	}
	msg := c.Pull()
	log.Debug(string(msg))
}
