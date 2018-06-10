package main

import (
	"expvar"
	"flag"
	"strings"
	"time"

	"net/http"
	_ "net/http/pprof"

	etcd "github.com/coreos/etcd/clientv3"

	"github.com/tddhit/diskqueue/server"
	"github.com/tddhit/tools/log"
	"github.com/tddhit/wox/naming"
)

var (
	etcdAddrs  string
	registry   string
	listenAddr string
	profAddr   string
	dataPath   string
	logPath    string
	logLevel   int
)

func init() {
	flag.StringVar(&etcdAddrs, "etcd-addrs", "127.0.0.1:2379", "etcd addrs")
	flag.StringVar(&registry, "registry", "", "registry name")
	flag.StringVar(&listenAddr, "listen-addr", "0.0.0.0:18800", "tcp listen address")
	flag.StringVar(&profAddr, "prof-addr", "0.0.0.0:6060", "http pprof address")
	flag.StringVar(&dataPath, "data-path", "./default.data", "data path")
	flag.StringVar(&logPath, "log-path", "diskqueue.log", "log path")
	flag.IntVar(&logLevel, "log-level", 1, "log level")
	flag.Parse()
}

func main() {
	endpoints := strings.Split(etcdAddrs, ",")
	cfg := etcd.Config{
		Endpoints:   endpoints,
		DialTimeout: 2000 * time.Millisecond,
	}
	etcdClient, err := etcd.New(cfg)
	if err != nil {
		log.Fatal(err)
	}
	addr := naming.GetLocalAddr(listenAddr)
	r := &naming.Registry{
		Client:     etcdClient,
		Timeout:    2 * time.Second,
		TTL:        3,
		Target:     registry,
		ListenAddr: addr,
	}
	r.Register()
	log.Init(logPath, logLevel)
	opts := diskqueue.NewOptions()
	opts.TCPAddress = addr
	opts.DataPath = dataPath
	dq := diskqueue.New(opts)
	profAddr = naming.GetLocalAddr(profAddr)
	go func() {
		s := &http.Server{
			Addr:    profAddr,
			Handler: expvar.Handler(),
		}
		s.ListenAndServe()
	}()
	dq.Go()
}
