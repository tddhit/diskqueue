package main

import (
	"context"
	"strconv"
	"testing"

	"google.golang.org/grpc"

	pb "github.com/tddhit/diskqueue/pb"
	"github.com/tddhit/tools/log"
)

func TestProducer(t *testing.T) {
	conn, err := grpc.Dial("127.0.0.1:9010", grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}
	client := pb.NewDiskqueueClient(conn)
	streamClient, err := client.MPublish(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	for i := 0; i < 100000; i++ {
		_, err := client.Publish(context.Background(), &pb.PublishRequest{
			Topic: "Test",
			Data:  []byte("test_" + strconv.Itoa(i)),
		})
		if err != nil {
			log.Fatal(err)
		}
	}
	for i := 10000; i < 20000; i++ {
		for j := 0; j < 10; j++ {
			err := streamClient.Send(&pb.PublishRequest{
				Topic: "Test",
				Data:  []byte("mtest_" + strconv.Itoa(i*10+j)),
			})
			if err != nil {
				log.Fatal(err)
			}
		}
	}
	_, err = streamClient.CloseAndRecv()
	if err != nil {
		log.Fatal(err)
	}
}
