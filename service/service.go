package service

import (
	"context"
	"io"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	pb "github.com/tddhit/diskqueue/pb"
	"github.com/tddhit/diskqueue/store"
	"github.com/tddhit/tools/dirlock"
	"github.com/tddhit/tools/log"
)

type Service struct {
	sync.Mutex
	topics   sync.Map
	clients  sync.Map
	dataPath string
	dl       *dirlock.DirLock
}

func New(dataPath string) *Service {
	s := &Service{
		dataPath: dataPath,
		dl:       dirlock.New(dataPath),
	}
	if err := s.dl.Lock(); err != nil {
		log.Fatal(err)
	}
	return s
}

func (s *Service) Publish(ctx context.Context,
	in *pb.PublishRequest) (*pb.PublishReply, error) {

	topic := s.getTopic(in.GetTopic())
	if err := topic.PutMessage(in.GetData()); err != nil {
		return nil, status.New(codes.Aborted, err.Error()).Err()
	}
	return &pb.PublishReply{}, nil
}

func (s *Service) MPublish(stream pb.Diskqueue_MPublishServer) error {
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&pb.PublishReply{})
		}
		if err != nil {
			log.Error(err)
			return err
		}
		topic := s.getTopic(in.GetTopic())
		if err := topic.PutMessage(in.GetData()); err != nil {
			return status.New(codes.Aborted, err.Error()).Err()
		}
	}
}

func (s *Service) Subscribe(ctx context.Context,
	in *pb.SubscribeRequest) (*pb.SubscribeReply, error) {

	p, _ := peer.FromContext(ctx)
	addr := p.Addr.String()
	_, ok := s.clients.Load(addr)
	if ok {
		return nil, status.New(codes.AlreadyExists,
			"client already exists").Err()
	} else {
		topic := s.getTopic(in.GetTopic())
		c := newClient(addr, topic)
		if err := topic.AddConsumer(c); err != nil {
			return nil, status.New(codes.AlreadyExists,
				"channel already subscribed").Err()
		}
		s.clients.Store(addr, c)
		return &pb.SubscribeReply{}, nil
	}
}

func (s *Service) Cancel(ctx context.Context,
	in *pb.CancelRequest) (*pb.CancelReply, error) {

	c := ctx.Value("client").(*client)
	c.topic.RemoveConsumer(c.String())
	s.clients.Delete(c.String())
	return &pb.CancelReply{}, nil
}

func (s *Service) Pull(ctx context.Context,
	in *pb.PullRequest) (*pb.PullReply, error) {

	c := ctx.Value("client").(*client)
	c.inFlightCond.L.Lock()
	for !c.canPull() {
		c.inFlightCond.Wait()
	}
	msg := c.topic.GetMessage()
	c.pushInFlight(msg)
	c.inFlightCond.L.Unlock()
	if msg != nil {
		return &pb.PullReply{
			Message: msg,
		}, nil
	}
	return nil, status.Error(codes.Unavailable, "msg is nil")
}

func (s *Service) KeepAlive(in *pb.KeepAliveRequest,
	stream pb.Diskqueue_KeepAliveServer) error {

	p, _ := peer.FromContext(stream.Context())
	addr := p.Addr.String()
	c, _ := s.clients.Load(addr)
	cc := c.(*client)
	ticker := time.NewTicker(time.Second)
	for range ticker.C {
		if err := stream.Send(&pb.KeepAliveReply{}); err != nil {
			log.Error(err)
			cc.topic.RemoveConsumer(cc.String())
			s.clients.Delete(cc.String())
			return err
		}
	}
	return nil
}

func (s *Service) Ack(ctx context.Context,
	in *pb.AckRequest) (*pb.AckReply, error) {

	c := ctx.Value("client").(*client)
	if err := c.removeFromInFlight(in.GetMsgID()); err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	return &pb.AckReply{}, nil
}

func (s *Service) getTopic(name string) *store.Topic {
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

func (s *Service) Close() {
	s.topics.Range(func(key, value interface{}) bool {
		t := value.(*store.Topic)
		t.Close()
		return true
	})
}
