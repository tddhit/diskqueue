package main

import (
	"flag"

	"github.com/tddhit/box/metrics"
	"github.com/tddhit/box/mw"
	"github.com/tddhit/box/tracing"
	"github.com/tddhit/box/transport"
	tropt "github.com/tddhit/box/transport/option"
	"github.com/tddhit/tools/log"

	pb "github.com/tddhit/diskqueue/pb"
	"github.com/tddhit/diskqueue/service"
)

var (
	listenAddr string
	dataPath   string
	logPath    string
	logLevel   int
)

func init() {
	flag.StringVar(&listenAddr, "listen-addr", "grpc://:9010", "grpc listen address")
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
		listenAddr,
		tropt.WithUnaryServerMiddleware(
			service.CheckPeerWithUnary(svc),
			tracing.ServerMiddleware,
			metrics.Middleware,
		),
		tropt.WithStreamServerMiddleware(service.CheckPeerWithStream(svc)),
		tropt.WithBeforeClose(svc.Close),
	)
	if err != nil {
		log.Fatal(err)
	}
	grpcServer.Register(pb.DiskqueueGrpcServiceDesc, svc)
	mw.Run(mw.WithServer(grpcServer))
}
