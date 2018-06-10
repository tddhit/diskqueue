package client

import (
	"sync"
	"testing"
	"time"

	etcd "github.com/coreos/etcd/clientv3"
	"github.com/tddhit/tools/log"
)

func consume(i int, ec *etcd.Client) {
	var channel string
	if i%2 == 0 {
		channel = "channel0"
	} else {
		channel = "channel1"
	}
	c, err := NewConsumer(ec, "/diskqueue", "topic1", channel)
	if err != nil {
		log.Error(err)
		return
	}
	for {
		msg := c.Pull()
		if msg == nil {
			break
		}
		log.Info("Consume", i, msg.Id, string(msg.Data))
	}
}

func TestConsumer(t *testing.T) {
	log.Init("consumer.log", log.INFO)

	cfg := etcd.Config{
		Endpoints:   []string{"127.0.0.1:2379"},
		DialTimeout: 2000 * time.Millisecond,
	}
	etcdClient, err := etcd.New(cfg)
	if err != nil {
		log.Fatal(err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 1; i++ {
		wg.Add(1)
		go func(i int, c *etcd.Client) {
			consume(i, c)
			wg.Done()
		}(i, etcdClient)
	}
	wg.Wait()
}
