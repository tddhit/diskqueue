package main

import (
	"expvar"
	"flag"
	"net"
	"net/http"

	"google.golang.org/grpc"

	"github.com/tddhit/diskqueue/handler"
	pb "github.com/tddhit/diskqueue/pb"
	"github.com/tddhit/tools/log"
)

var (
	listenAddr string
	dataPath   string
	logPath    string
	logLevel   int
)

func init() {
	flag.StringVar(&listenAddr, "listen-addr", ":9010", "listen address")
	flag.StringVar(&dataPath, "datapath", "", "data path")
	flag.StringVar(&logPath, "logpath", "", "log file path")
	flag.IntVar(&logLevel, "loglevel", 1, "log level (Trace:1, Debug:2, Info:3, Error:5)")
	flag.Parse()
}

func main() {
	log.Init(logPath, logLevel)
	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatal(err)
	}
	s := grpc.NewServer(grpc.UnaryInterceptor(handler.UnaryIntercept))
	handler := handler.NewHandler(dataPath)
	pb.RegisterDiskqueueServer(s, handler)
	go func() {
		http.ListenAndServe(":6060", expvar.Handler())
	}()
	s.Serve(lis)
}
