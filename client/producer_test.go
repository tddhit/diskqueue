package client

import (
	"strconv"
	"sync"
	"testing"

	"github.com/tddhit/tools/log"
)

func produce(i int) {
	p := NewProducer("127.0.0.1:18800")
	var j = 0
	for j = i * 10000; j < i*10000+10000; j++ {
		d := "hello" + strconv.Itoa(j)
		err := p.Publish("topic1", []byte(d))
		if err != nil {
			log.Fatal(err)
		}
	}
	p.Stop()
	log.Info(j)
}

func TestProducer(t *testing.T) {
	log.Init("producer.log", log.DEBUG)
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			produce(i)
			wg.Done()
		}(i)
	}
	wg.Wait()
}
