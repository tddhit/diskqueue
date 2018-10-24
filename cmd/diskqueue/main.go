package main

import (
	"os"

	"github.com/urfave/cli"

	"github.com/tddhit/tools/log"
)

var (
	logPathFlag = cli.StringFlag{
		Name:  "logpath",
		Usage: "(default: stderr)",
	}
	logLevelFlag = cli.IntFlag{
		Name:  "loglevel",
		Usage: "trace:1, debug:2, info:3, warn:4, error:5",
		Value: 2,
	}
)

func main() {
	app := cli.NewApp()
	app.Name = "diskqueue"
	app.Usage = "distributed log storage system"
	app.Version = "0.0.2"
	app.Commands = []cli.Command{
		serviceCommand,
		benchmarkCommand,
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func withLog(f func(ctx *cli.Context)) func(ctx *cli.Context) {
	return func(ctx *cli.Context) {
		logPath := ctx.String("logpath")
		logLevel := ctx.Int("loglevel")
		log.Init(logPath, logLevel)
		f(ctx)
	}
}
