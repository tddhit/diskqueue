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
	listenAddr  string
	clusterAddr string
	leaderAddr  string
	nodeID      string
	dataDir     string
	mode        string
	pidPath     string
	logPath     string
	logLevel    int
)

func init() {
	flag.StringVar(&listenAddr, "listen-addr", "grpc://:9010",
		"client communication address")
	flag.StringVar(&mode, "mode", "standalone", "[standalone|cluster]")
	flag.StringVar(&clusterAddr, "cluster-addr", ":8010",
		"raft communication address")
	flag.StringVar(&leaderAddr, "leader-addr", "", "leader address in raft cluster")
	flag.StringVar(&nodeID, "id", "", "node id in raft cluster")
	flag.StringVar(&pidPath, "pidpath", "/var/diskqueue.pid", "pid path")
	flag.StringVar(&dataDir, "datadir", "", "data directory")
	flag.StringVar(&logPath, "logpath", "", "log file path")
	flag.IntVar(&logLevel, "loglevel", 1,
		"log level (Trace:1, Debug:2, Info:3, Error:5)")
	flag.Parse()
}

func main() {
	log.Init(logPath, logLevel)
	svc := service.New(dataDir, mode, clusterAddr, nodeID, leaderAddr)
	grpcServer, err := transport.Listen(
		listenAddr,
		tropt.WithUnaryServerMiddleware(
			service.CheckPeerWithUnary(svc),
			tracing.ServerMiddleware,
			metrics.Middleware,
		),
		tropt.WithBeforeClose(svc.Close),
	)
	if err != nil {
		log.Fatal(err)
	}
	grpcServer.Register(pb.DiskqueueGrpcServiceDesc, svc)
	mw.Run(mw.WithServer(grpcServer), mw.WithPIDPath(pidPath))
}
