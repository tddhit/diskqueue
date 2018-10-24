package main

import (
	"bufio"
	"context"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/urfave/cli"

	"github.com/tddhit/box/transport"
	"github.com/tddhit/diskqueue/pb"
	"github.com/tddhit/tools/log"
)

var (
	benchmarkCommand = cli.Command{
		Name:      "benchmark",
		Usage:     "start benchmark",
		Action:    withLog(startBenchmark),
		UsageText: "alexander benchmark [producer|consumer] [arguments...]",
		Flags: []cli.Flag{
			logPathFlag,
			logLevelFlag,
			cli.StringFlag{
				Name:  "addr",
				Usage: "diskqueue addr",
				Value: "grpc://127.0.0.1:9000",
			},
			cli.StringFlag{
				Name:  "topics",
				Usage: "topic&concurrency (used by producer)",
				Value: "topic1&1,topic2&10",
			},
			cli.StringFlag{
				Name:  "channels",
				Usage: "topic&channel&concurrency (used by consumer)",
				Value: "topic1&channel1&1,topic1&channel2&10",
			},
			cli.StringFlag{
				Name:  "src",
				Usage: "data source file (used by producer)",
				Value: "./src.txt",
			},
			cli.IntFlag{
				Name:  "n",
				Usage: "numbers of messages",
				Value: 10000,
			},
			cli.IntFlag{
				Name:  "t",
				Usage: "time(second)",
				Value: 60,
			},
		},
	}
)

func startBenchmark(params *cli.Context) {
	switch params.Args().First() {
	case "producer":
		startProduce(params)
	case "consumer":
		startConsume(params)
	}
}

func startProduce(params *cli.Context) {
	var (
		dataC       = make(chan []byte, 10000)
		wg          sync.WaitGroup
		topics      = strings.Split(params.String("topics"), ",")
		ctx, cancel = context.WithDeadline(
			context.Background(),
			time.Now().Add(time.Duration(params.Int("t"))*time.Second),
		)
	)
	wg.Add(1)
	go func() {
		generate(ctx, params.String("src"), dataC)
		wg.Done()
	}()
	for i, _ := range topics {
		v := strings.Split(topics[i], "&")
		topic := v[0]
		concurrency, _ := strconv.Atoi(v[1])
		for i := 0; i < concurrency; i++ {
			conn, err := transport.Dial(params.String("addr"))
			if err != nil {
				log.Fatal(err)
			}
			client := diskqueuepb.NewDiskqueueGrpcClient(conn)
			wg.Add(1)
			go func(c diskqueuepb.DiskqueueGrpcClient, topic string) {
				defer wg.Done()
				count := 0
				for count < params.Int("n") {
					select {
					case <-ctx.Done():
						return
					case data := <-dataC:
						rsp, err := c.Push(ctx, &diskqueuepb.PushReq{
							Topic: topic,
							Data:  data,
						})
						if err != nil {
							log.Error(err)
							continue
						}
						log.Infof("\t%s\t%d\t%s", topic, rsp.ID, string(data))
					}
				}
			}(client, topic)
		}
	}
	go func() {
		signalC := make(chan os.Signal)
		signal.Notify(signalC)
		<-signalC
		cancel()
	}()
	wg.Wait()
}

func generate(ctx context.Context, file string, dataC chan<- []byte) {
	f, err := os.Open(file)
	if err != nil {
		log.Fatal(err)
	}
	data := make([][]byte, 0)
	buf := make([]byte, 4096)
	scanner := bufio.NewScanner(f)
	scanner.Buffer(buf, cap(buf))
	for scanner.Scan() {
		line := scanner.Text()
		data = append(data, []byte(line))
	}
	for {
		for i, _ := range data {
			select {
			case dataC <- data[i]:
			case <-ctx.Done():
				return
			}
		}
	}
}

func startConsume(params *cli.Context) {
	var (
		wg          sync.WaitGroup
		channels    = strings.Split(params.String("channels"), ",")
		ctx, cancel = context.WithDeadline(
			context.Background(),
			time.Now().Add(time.Duration(params.Int("t"))*time.Second),
		)
	)
	for i, _ := range channels {
		v := strings.Split(channels[i], "&")
		topic := v[0]
		channel := v[1]
		concurrency, _ := strconv.Atoi(v[2])
		for i := 0; i < concurrency; i++ {
			conn, err := transport.Dial(params.String("addr"))
			if err != nil {
				log.Fatal(err)
			}
			client := diskqueuepb.NewDiskqueueGrpcClient(conn)
			wg.Add(1)
			go func(c diskqueuepb.DiskqueueGrpcClient, topic, channel string) {
				defer wg.Done()
				count := 0
				for count < params.Int("n") {
					select {
					case <-ctx.Done():
						return
					default:
						rsp, err := c.Pop(ctx, &diskqueuepb.PopReq{
							Topic:   topic,
							Channel: channel,
						})
						if err != nil {
							log.Error(err)
							continue
						}
						log.Infof("\t%s\t%s\t%d\t%s", topic, channel,
							rsp.Message.ID, string(rsp.Message.Data))
					}
				}
			}(client, topic, channel)
		}
	}
	go func() {
		signalC := make(chan os.Signal)
		signal.Notify(signalC)
		<-signalC
		cancel()
	}()
	wg.Wait()
}
