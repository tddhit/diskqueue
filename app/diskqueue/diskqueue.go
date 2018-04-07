package main

import (
	"flag"

	"github.com/tddhit/diskqueue"
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
	opts := diskqueue.NewOptions()
	opts.TCPAddress = address
	opts.DataPath = dataPath
	dq := diskqueue.New(opts)
	dq.Go()
	dq.Wait()
	dq.Exit()
}
