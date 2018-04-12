package client

import (
	"strconv"
	"sync"
	"testing"

	"github.com/tddhit/tools/log"
)

func consume(i int) {
	c := NewConsumer("topic1", "channel"+strconv.Itoa(i))
	err := c.Connect("127.0.0.1:18800", strconv.Itoa(i))
	if err != nil {
		log.Fatal(err)
	}
	for {
		msg := c.Pull()
		if msg == nil {
			break
		}
		log.Info("Consume", i, string(msg))
	}
}

func TestConsumer(t *testing.T) {
	log.Init("consumer.log", log.INFO)
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			consume(i)
			wg.Done()
		}(i)
	}
	wg.Wait()
}
