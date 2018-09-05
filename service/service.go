package service

import (
	"context"
	"io"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/tddhit/tools/dirlock"
	"github.com/tddhit/tools/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/tddhit/diskqueue/pb"
	"github.com/tddhit/diskqueue/store"
)

type Service struct {
	sync.RWMutex
	topics   map[string]*store.Topic
	clients  map[string]*client
	filters  map[string]*filter
	dataPath string
	dl       *dirlock.DirLock

	filterCounter *prometheus.CounterVec
}

func New(dataPath string) *Service {
	s := &Service{
		topics:   make(map[string]*store.Topic),
		filters:  make(map[string]*filter),
		clients:  make(map[string]*client),
		dataPath: dataPath,
		dl:       dirlock.New(dataPath),

		filterCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "filter_count",
				Help: "filter by bloom",
			},
			[]string{},
		),
	}
	prometheus.MustRegister(s.filterCounter)
	if err := s.dl.Lock(); err != nil {
		log.Fatal(err)
	}
	return s
}

func (s *Service) Publish(ctx context.Context,
	in *pb.PublishRequest) (*pb.PublishReply, error) {

	topic := s.getOrCreateTopic(in.GetTopic())
	filter := s.getOrCreateFilter(in.GetTopic())
	if !in.GetIgnoreFilter() {
		key := in.GetData()
		if in.GetHashKey() != nil {
			key = in.GetHashKey()
		}
		if filter.mayContain(key) {
			s.filterCounter.WithLabelValues().Inc()
			return nil, status.Error(codes.AlreadyExists, "filter by bloom")
		} else {
			if err := filter.add(key); err != nil {
				return nil, status.Error(codes.Internal, err.Error())
			}
		}
	}
	if err := topic.PutMessage(in.GetData()); err != nil {
		return nil, status.Error(codes.Aborted, err.Error())
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
		topic := s.getOrCreateTopic(in.GetTopic())
		if err := topic.PutMessage(in.GetData()); err != nil {
			return status.Error(codes.Aborted, err.Error())
		}
	}
}

func (s *Service) Pull(ctx context.Context,
	in *pb.PullRequest) (*pb.PullReply, error) {

	topic := s.getOrCreateTopic(in.GetTopic())
	var msg *pb.Message
	if in.GetNeedAck() {
		client := ctx.Value("client").(*client)
		inflight := client.getOrCreateInflight(topic)
		inflight.cond.L.Lock()
		for inflight.full() {
			inflight.cond.Wait()
		}
		msg = topic.GetMessage()
		inflight.push(msg)
		inflight.cond.L.Unlock()
	} else {
		msg = topic.GetMessage()
	}
	if msg != nil {
		return &pb.PullReply{
			Message: msg,
		}, nil
	}
	return nil, status.Error(codes.Unavailable, "msg is nil")
}

func (s *Service) Ack(ctx context.Context,
	in *pb.AckRequest) (*pb.AckReply, error) {

	client := ctx.Value("client").(*client)
	log.Info(s.topics, ",", in.GetTopic(), ",")
	topic, exist := s.getTopic(in.GetTopic())
	if !exist {
		return nil, status.Error(codes.FailedPrecondition,
			"not found topic in diskqueue")
	}
	inflight, exist := client.getInflight(topic.Name)
	if !exist {
		return nil, status.Error(codes.FailedPrecondition,
			"not found inflight in client")
	}
	if err := inflight.remove(in.GetMsgID()); err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}
	return &pb.AckReply{}, nil
}

func (s *Service) getTopic(name string) (*store.Topic, bool) {
	s.RLock()
	defer s.RUnlock()

	t, ok := s.topics[name]
	return t, ok
}

func (s *Service) getOrCreateTopic(name string) *store.Topic {
	s.RLock()
	if t, ok := s.topics[name]; ok {
		s.RUnlock()
		return t
	}
	s.RUnlock()

	s.Lock()
	if t, ok := s.topics[name]; ok {
		s.Unlock()
		return t
	}
	topic, err := store.NewTopic(s.dataPath, name)
	if err != nil {
		log.Fatal(err)
	}
	s.topics[name] = topic
	s.Unlock()
	return topic
}

func (s *Service) getClient(addr string) (*client, bool) {
	s.RLock()
	defer s.RUnlock()

	c, ok := s.clients[addr]
	return c, ok
}

func (s *Service) getOrCreateClient(addr string) *client {
	s.RLock()
	if t, ok := s.clients[addr]; ok {
		s.RUnlock()
		return t
	}
	s.RUnlock()

	s.Lock()
	if t, ok := s.clients[addr]; ok {
		s.Unlock()
		return t
	}
	client := &client{
		addr:      addr,
		inflights: make(map[string]*inflight),
	}
	s.clients[addr] = client
	s.Unlock()
	return client
}

func (s *Service) getFilter(topic string) (*filter, bool) {
	s.RLock()
	defer s.RUnlock()

	f, ok := s.filters[topic]
	return f, ok
}

func (s *Service) getOrCreateFilter(topic string) *filter {
	s.RLock()
	if f, ok := s.filters[topic]; ok {
		s.RUnlock()
		return f
	}
	s.RUnlock()

	s.Lock()
	if f, ok := s.filters[topic]; ok {
		s.Unlock()
		return f
	}
	filter, err := newFilter(s.dataPath, topic)
	if err != nil {
		log.Fatal(err)
	}
	s.filters[topic] = filter
	s.Unlock()
	return filter
}

func (s *Service) Close() {
	s.Lock()
	defer s.Unlock()

	for _, t := range s.topics {
		t.Close()
	}
}
