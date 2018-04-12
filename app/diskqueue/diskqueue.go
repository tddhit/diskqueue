package main

import (
	"flag"

	"net/http"
	_ "net/http/pprof"

	"github.com/tddhit/diskqueue"
	"github.com/tddhit/tools/log"
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
