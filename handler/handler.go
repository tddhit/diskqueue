package handler

import (
	"context"
	"io"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"github.com/tddhit/diskqueue/core"
	pb "github.com/tddhit/diskqueue/pb"
	"github.com/tddhit/tools/dirlock"
	"github.com/tddhit/tools/log"
)

type handler struct {
	sync.Mutex
	topic    sync.Map
	client   sync.Map
	dataPath string
	dl       *dirlock.DirLock
}

func NewHandler(dataPath string) *handler {
	h := &handler{
		dataPath: dataPath,
		dl:       dirlock.New(dataPath),
	}
	if err := h.dl.Lock(); err != nil {
		log.Fatal(err)
	}
	return h
}

func (h *handler) Publish(ctx context.Context,
	in *pb.PublishRequest) (*pb.PublishReply, error) {

	topic := h.getTopic(in.GetTopic())
	if err := topic.PutMessage(in.GetData()); err != nil {
		return nil, status.New(codes.Aborted, err.Error()).Err()
	}
	return &pb.PublishReply{}, nil
}

func (h *handler) MPublish(stream pb.Diskqueue_MPublishServer) error {
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&pb.PublishReply{})
		}
		if err != nil {
			return err
		}
		topic := h.getTopic(in.GetTopic())
		if err := topic.PutMessage(in.GetData()); err != nil {
			return status.New(codes.Aborted, err.Error()).Err()
		}
	}
}

func (h *handler) Subscribe(ctx context.Context,
	in *pb.SubscribeRequest) (*pb.SubscribeReply, error) {

	p, _ := peer.FromContext(ctx)
	addr := p.Addr.String()
	_, ok := h.client.Load(addr)
	if ok {
		return nil, status.New(codes.AlreadyExists,
			"client already exists").Err()
	} else {
		topic := h.getTopic(in.GetTopic())
		channel := topic.GetChannel(in.GetChannel(), in.GetMsgId())
		client := &Client{
			Addr:    addr,
			State:   stateSubscribed,
			Channel: channel,
		}
		err := channel.AddClient(client)
		if err != nil {
			return nil, status.New(codes.AlreadyExists,
				"channel already subscribed").Err()
		}
		h.client.Store(addr, client)
		return &pb.SubscribeReply{}, nil
	}
}

func (h *handler) Cancel(ctx context.Context,
	in *pb.CancelRequest) (*pb.CancelReply, error) {

	reply, err := h.execute(ctx, func(client *Client) (interface{}, error) {
		client.Channel.RemoveClient()
		h.client.Delete(client.String())
		return &pb.CancelReply{}, nil
	})
	return reply.(*pb.CancelReply), err
}

func (h *handler) Pull(ctx context.Context,
	in *pb.PullRequest) (*pb.PullReply, error) {

	reply, err := h.execute(ctx, func(client *Client) (interface{}, error) {
		msg := client.Channel.GetMessage()
		return &pb.PullReply{
			Message: msg,
		}, nil
	})
	return reply.(*pb.PullReply), err
}

func (h *handler) KeepAlive(in *pb.KeepAliveRequest,
	stream pb.Diskqueue_KeepAliveServer) error {

	_, err := h.execute(stream.Context(), func(client *Client) (interface{}, error) {
		ticker := time.NewTicker(time.Second)
		for range ticker.C {
			if err := stream.Send(&pb.KeepAliveReply{}); err != nil {
				client.Channel.RemoveClient()
				h.client.Delete(client.String())
				return nil, err
			}
		}
		return nil, nil
	})
	return err
}

func (h *handler) getTopic(name string) *core.Topic {
	if t, ok := h.topic.Load(name); ok {
		return t.(*core.Topic)
	}
	h.Lock()
	if t, ok := h.topic.Load(name); ok {
		h.Unlock()
		return t.(*core.Topic)
	}
	topic, err := core.NewTopic(h.dataPath, name)
	if err != nil {
		log.Fatal(err)
	}
	h.topic.Store(name, topic)
	h.Unlock()
	return topic
}

func (h *handler) execute(ctx context.Context,
	f func(*Client) (interface{}, error)) (interface{}, error) {

	p, _ := peer.FromContext(ctx)
	addr := p.Addr.String()
	c, ok := h.client.Load(addr)
	if ok {
		client := c.(*Client)
		if client.State == stateSubscribed {
			return f(client)
		} else {
			return nil, status.New(codes.FailedPrecondition,
				"client state is not stateSubscribed").Err()
		}
	} else {
		return nil, status.New(codes.FailedPrecondition,
			"client has not subscribed").Err()
	}
}
