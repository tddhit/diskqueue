package client

import (
	"strconv"
	"sync"
	"testing"

	etcd "github.com/coreos/etcd/clientv3"
	"github.com/tddhit/tools/log"
)

func consume(i int) {
	etcdClient, err := etcd.New(etcd.Config{
		Endpoints: []string{"localhost:2379"},
	})
	if err != nil {
		log.Fatal(err)
	}
	c := NewConsumer("topic1", "channel"+strconv.Itoa(i))
	c.ConnectByEtcd(etcdClient, "/nlpservice/diskqueue", strconv.Itoa(i))
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
