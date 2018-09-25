package main

import (
	"context"
	"strconv"
	"testing"

	"github.com/tddhit/box/transport"
	pb "github.com/tddhit/diskqueue/pb"
	_ "github.com/tddhit/diskqueue/resolver"
	"github.com/tddhit/tools/log"
)

func TestProducer(t *testing.T) {
	log.Init("producer.log", log.INFO)
	conn, err := transport.Dial("diskqueue://127.0.0.1:9010,127.0.0.1:9020,127.0.0.1:9030/leader")
	if err != nil {
		log.Fatal(err)
	}
	client := pb.NewDiskqueueGrpcClient(conn)
	for i := 10; i < 20; i++ {
		_, err := client.Push(context.Background(), &pb.PushRequest{
			Topic: "Test",
			Data:  []byte("test_" + strconv.Itoa(i)),
		})
		if err != nil {
			log.Fatal(err)
		}
	}
}
