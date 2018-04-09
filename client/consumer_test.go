package client

import (
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/tddhit/tools/log"
)

func consume(i int) {
	c := NewConsumer("topic1", "channel"+strconv.Itoa(i))
	err := c.Connect("127.0.0.1:18800", strconv.Itoa(i*500))
	if err != nil {
		log.Fatal(err)
	}
	for {
		msg := c.Pull()
		log.Debug(i, string(msg))
	}
}

func TestConsumer(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			consume(i)
			wg.Done()
		}(i)
	}
	wg.Wait()
	time.Sleep(10 * time.Second)
}
