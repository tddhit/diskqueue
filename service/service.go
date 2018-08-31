package service

import (
	"context"
	"io"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"github.com/gogo/protobuf/proto"
	pb "github.com/tddhit/diskqueue/pb"
	"github.com/tddhit/diskqueue/store"
	"github.com/tddhit/tools/dirlock"
	"github.com/tddhit/tools/log"
)

type service struct {
	sync.Mutex
	topics   sync.Map
	clients  sync.Map
	dataPath string
	dl       *dirlock.DirLock
}

func New(dataPath string) *service {
	s := &service{
		dataPath: dataPath,
		dl:       dirlock.New(dataPath),
	}
	if err := s.dl.Lock(); err != nil {
		log.Fatal(err)
	}
	return s
}

func (s *service) Publish(ctx context.Context,
	in *pb.PublishRequest) (*pb.PublishReply, error) {

	topic := s.getTopic(in.GetTopic())
	if err := topic.PutMessage(in.GetData()); err != nil {
		return nil, status.New(codes.Aborted, err.Error()).Err()
	}
	return &pb.PublishReply{}, nil
}

func (s *service) MPublish(stream pb.Diskqueue_MPublishServer) error {
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&pb.PublishReply{})
		}
		if err != nil {
			return err
		}
		topic := s.getTopic(in.GetTopic())
		if err := topic.PutMessage(in.GetData()); err != nil {
			return status.New(codes.Aborted, err.Error()).Err()
		}
	}
}

func (s *service) Subscribe(ctx context.Context,
	in *pb.SubscribeRequest) (*pb.SubscribeReply, error) {

	p, _ := peer.FromContext(ctx)
	addr := p.Addr.String()
	_, ok := s.clients.Load(addr)
	if ok {
		return nil, status.New(codes.AlreadyExists,
			"client already exists").Err()
	} else {
		topic := s.getTopic(in.GetTopic())
		client := &client{
			addr:  addr,
			state: stateSubscribed,
			topic: topic,
		}
		if err := topic.AddConsumer(client); err != nil {
			return nil, status.New(codes.AlreadyExists,
				"channel already subscribed").Err()
		}
		s.clients.Store(addr, client)
		return &pb.SubscribeReply{}, nil
	}
}

func (s *service) Cancel(ctx context.Context,
	in *pb.CancelRequest) (*pb.CancelReply, error) {

	reply, err := s.execute(ctx, in,
		func(c *client, in proto.Message) (interface{}, error) {
			c.topic.RemoveConsumer(c.String())
			s.clients.Delete(c.String())
			return &pb.CancelReply{}, nil
		},
	)
	return reply.(*pb.CancelReply), err
}

func (s *service) Pull(ctx context.Context,
	in *pb.PullRequest) (*pb.PullReply, error) {

	reply, err := s.execute(ctx, in,
		func(c *client, in proto.Message) (interface{}, error) {
			msg := c.topic.GetMessage()
			return &pb.PullReply{
				Message: msg,
			}, nil
		},
	)
	return reply.(*pb.PullReply), err
}

func (s *service) KeepAlive(in *pb.KeepAliveRequest,
	stream pb.Diskqueue_KeepAliveServer) error {

	_, err := s.execute(stream.Context(), nil,
		func(c *client, in proto.Message) (interface{}, error) {
			ticker := time.NewTicker(time.Second)
			for range ticker.C {
				if err := stream.Send(&pb.KeepAliveReply{}); err != nil {
					c.topic.RemoveConsumer(c.String())
					s.clients.Delete(c.String())
					return nil, err
				}
			}
			return nil, nil
		},
	)
	return err
}

func (s *service) Ack(ctx context.Context,
	in *pb.AckRequest) (*pb.AckReply, error) {

	reply, err := s.execute(ctx, in,
		func(c *client, in proto.Message) (interface{}, error) {
			req := in.(*pb.AckRequest)
			if err := c.topic.Ack(req.GetMsgID()); err != nil {
				return nil, err
			}
			return &pb.AckReply{}, nil
		},
	)
	return reply.(*pb.AckReply), err
}

func (s *service) getTopic(name string) *store.Topic {
	if t, ok := s.topics.Load(name); ok {
		return t.(*store.Topic)
	}
	s.Lock()
	if t, ok := s.topics.Load(name); ok {
		s.Unlock()
		return t.(*store.Topic)
	}
	topic, err := store.NewTopic(s.dataPath, name)
	if err != nil {
		log.Fatal(err)
	}
	s.topics.Store(name, topic)
	s.Unlock()
	return topic
}

func (s *service) execute(ctx context.Context, in proto.Message,
	f func(*client, proto.Message) (interface{}, error)) (interface{}, error) {

	p, _ := peer.FromContext(ctx)
	addr := p.Addr.String()
	c, ok := s.clients.Load(addr)
	if ok {
		client := c.(*client)
		if client.state == stateSubscribed {
			return f(client, in)
		} else {
			return nil, status.New(codes.FailedPrecondition,
				"client state is not stateSubscribed").Err()
		}
	} else {
		return nil, status.New(codes.FailedPrecondition,
			"client has not subscribed").Err()
	}
}
