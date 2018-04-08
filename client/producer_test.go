package client

import (
	"testing"

	"github.com/tddhit/tools/log"
)

func TestProducer(t *testing.T) {
	p := NewProducer("127.0.0.1:18800")
	err := p.Publish("topic1", []byte("hello20"))
	if err != nil {
		log.Fatal(err)
	}
}
