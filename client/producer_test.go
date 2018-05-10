package client

import (
	"strconv"
	"sync"
	"testing"
	"time"

	etcd "github.com/coreos/etcd/clientv3"
	"github.com/tddhit/tools/log"
)

func produce(i int, ec *etcd.Client) {
	p, err := NewProducer(ec, "/diskqueue")
	if err != nil {
		log.Fatal(err)
	}

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
	log.Init("producer.log", log.INFO)

	cfg := etcd.Config{
		Endpoints:   []string{"127.0.0.1:2379"},
		DialTimeout: 2000 * time.Millisecond,
	}
	ec, err := etcd.New(cfg)
	if err != nil {
		log.Fatal(err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int, ec *etcd.Client) {
			produce(i, ec)
			wg.Done()
		}(i, ec)
	}
	wg.Wait()
}
