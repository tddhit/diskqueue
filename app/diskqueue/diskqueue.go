package main

import (
	"flag"

	"net/http"
	_ "net/http/pprof"

	etcd "github.com/coreos/etcd/clientv3"

	"github.com/tddhit/diskqueue"
	"github.com/tddhit/tools/log"
	"github.com/tddhit/wox/naming"
)

var (
	address  string
	dataPath string
)

func init() {
	flag.StringVar(&address, "address", "0.0.0.0:18800", "tcp listen address")
	flag.StringVar(&dataPath, "dataPath", "./default.data", "data path")
}

func main() {
	etcdClient, err := etcd.New(etcd.Config{
		Endpoints: []string{"localhost:2379"},
	})
	if err != nil {
		log.Fatal(err)
	}
	r := &naming.Registry{
		Client:     etcdClient,
		Timeout:    2000,
		TTL:        1,
		Target:     "/nlpservice/diskqueue",
		ListenAddr: ":18800",
	}
	r.Register()
	log.Init("diskqueue.log", log.INFO)
	opts := diskqueue.NewOptions()
	opts.TCPAddress = address
	opts.DataPath = dataPath
	dq := diskqueue.New(opts)
	go func() {
		log.Debug(http.ListenAndServe("localhost:6060", nil))
	}()
	dq.Go()
}
