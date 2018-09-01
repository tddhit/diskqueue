package main

import (
	"context"
	"io"
	"testing"

	"google.golang.org/grpc"

	pb "github.com/tddhit/diskqueue/pb"
	"github.com/tddhit/tools/log"
)

func TestConsumer(t *testing.T) {
	log.Init("consumer.log", log.INFO)
	conn, err := grpc.Dial("127.0.0.1:9010", grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}
	client := pb.NewDiskqueueClient(conn)
	_, err = client.Subscribe(context.Background(), &pb.SubscribeRequest{
		Topic: "Test",
	})
	if err != nil {
		log.Fatal(err)
	}
	streamClient, err := client.KeepAlive(context.Background(), &pb.KeepAliveRequest{})
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		for {
			_, err := streamClient.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Fatal(err)
			}
		}
	}()
	for {
		reply, err := client.Pull(context.Background(), &pb.PullRequest{})
		if err != nil {
			log.Fatal(err)
		}
		id := reply.GetMessage().GetID()
		log.Info(id, string(reply.GetMessage().GetData()))
		_, err = client.Ack(context.Background(), &pb.AckRequest{MsgID: id})
		if err != nil {
			log.Fatal(err)
		}
		if id == 199999 {
			break
		}
	}
	_, err = client.Cancel(context.Background(), &pb.CancelRequest{})
	if err != nil {
		log.Fatal(err)
	}
}