package main

import (
	"github.com/urfave/cli"

	"github.com/tddhit/box/metrics"
	"github.com/tddhit/box/mw"
	"github.com/tddhit/box/tracing"
	"github.com/tddhit/box/transport"
	tropt "github.com/tddhit/box/transport/option"
	"github.com/tddhit/diskqueue/pb"
	"github.com/tddhit/diskqueue/service"
	"github.com/tddhit/tools/log"
)

var serviceCommand = cli.Command{
	Name:      "service",
	Usage:     "start diskqueue service",
	Action:    withLog(startService),
	UsageText: "diskqueue service [arguments...]",
	Flags: []cli.Flag{
		logPathFlag,
		logLevelFlag,
		cli.StringFlag{
			Name:  "pidpath",
			Usage: "(default: ./xxx.pid)",
		},
		cli.StringFlag{
			Name:  "addr",
			Usage: "client communication address",
			Value: "127.0.0.1:9000",
		},
		cli.StringFlag{
			Name:  "cluster-addr",
			Usage: "raft communication address",
			Value: ":9010",
		},
		cli.StringFlag{
			Name:  "leader",
			Usage: "leader address in raft cluster",
		},
		cli.StringFlag{
			Name:  "id",
			Usage: "node id in raft cluster",
		},
		cli.StringFlag{
			Name:  "datadir",
			Usage: "data directory",
			Value: "./data",
		},
	},
}

func startService(ctx *cli.Context) {
	addr := ctx.String("addr")
	pidPath := ctx.String("pidpath")
	svc := service.NewService(ctx)
	server, err := transport.Listen(
		"grpc://"+addr,
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
	server.Register(diskqueuepb.DiskqueueGrpcServiceDesc, svc)
	mw.Run(mw.WithServer(server), mw.WithPIDPath(pidPath))
}
