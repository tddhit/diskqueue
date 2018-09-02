package main

import (
	"expvar"
	"flag"
	"net/http"

	"github.com/tddhit/box/mw"
	"github.com/tddhit/box/transport"
	tropt "github.com/tddhit/box/transport/option"
	"github.com/tddhit/tools/log"

	pb "github.com/tddhit/diskqueue/pb"
	"github.com/tddhit/diskqueue/service"
)

var (
	grpcAddr string
	dataPath string
	logPath  string
	logLevel int
)

func init() {
	flag.StringVar(&grpcAddr, "grpcAddr", "grpc://:9010", "grpc listen address")
	flag.StringVar(&dataPath, "datapath", "", "data path")
	flag.StringVar(&logPath, "logpath", "", "log file path")
	flag.IntVar(&logLevel, "loglevel", 1, "log level (Trace:1, Debug:2, Info:3, Error:5)")
	flag.Parse()
}

func main() {
	log.Init(logPath, logLevel)
	var svc *service.Service
	if mw.IsWorker() {
		svc = service.New(dataPath)
	}
	grpcServer, err := transport.Listen(
		grpcAddr,
		tropt.WithUnaryServerMiddleware(service.CheckPeerWithUnary(svc)),
		tropt.WithStreamServerMiddleware(service.CheckPeerWithStream(svc)),
	)
	if err != nil {
		log.Fatal(err)
	}
	grpcServer.Register(pb.DiskqueueGrpcServiceDesc, svc)
	go func() {
		http.ListenAndServe(":6060", expvar.Handler())
	}()
	mw.Run(mw.WithServer(grpcServer))
	if mw.IsWorker() {
		svc.Close()
	}
}
