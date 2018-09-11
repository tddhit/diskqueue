package service

import (
	"context"
	"io"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/tddhit/box/mw"
	"github.com/tddhit/tools/dirlock"
	"github.com/tddhit/tools/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/tddhit/diskqueue/pb"
	"github.com/tddhit/diskqueue/store"
)

type queue interface {
	Push(topic string, data []byte) error
	Pop(topic string) *pb.Message
	getTopic(name string) (*store.Topic, bool)
	getOrCreateTopic(name string) *store.Topic
	join(raftAddr, nodeID string) error
	leave(nodeID string) error
	snapshot() error
	close() error
}

type Service struct {
	sync.RWMutex
	queue         queue
	clients       map[string]*client
	filters       map[string]*filter
	dataDir       string
	dl            *dirlock.DirLock
	filterCounter *prometheus.CounterVec
}

func New(dataDir, mode, raftAddr, nodeID, leaderAddr string) *Service {
	if !mw.IsWorker() {
		return nil
	}
	var q queue
	switch mode {
	case "cluster":
		cs, err := newClusterStore(dataDir, raftAddr, nodeID, leaderAddr)
		if err != nil {
			log.Fatal(err)
		}
		q = cs
	case "standalone":
		fallthrough
	default:
		q = newStandaloneStore(dataDir)
	}
	s := &Service{
		queue:   q,
		filters: make(map[string]*filter),
		clients: make(map[string]*client),
		dataDir: dataDir,
		dl:      dirlock.New(dataDir),
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

func (s *Service) Push(ctx context.Context,
	in *pb.PushRequest) (*pb.PushReply, error) {

	filter := s.getOrCreateFilter(in.GetTopic())
	if !in.GetIgnoreFilter() {
		key := in.GetData()
		if in.GetHashKey() != nil {
			key = in.GetHashKey()
		}
		if filter.mayContain(key) {
			s.filterCounter.WithLabelValues().Inc()
			log.Debug("filter:", string(key))
			return nil, status.Error(codes.AlreadyExists, "filter by bloom")
		} else {
			log.Debug("add:", string(key))
			if err := filter.add(key); err != nil {
				return nil, status.Error(codes.Internal, err.Error())
			}
		}
	}
	if err := s.queue.Push(in.Topic, in.Data); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.PushReply{}, nil
}

func (s *Service) MPush(stream pb.Diskqueue_MPushServer) error {
	for {
		in, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&pb.PushReply{})
		}
		if err != nil {
			log.Error(err)
			return err
		}
		if err := s.queue.Push(in.Topic, in.Data); err != nil {
			return status.Error(codes.Internal, err.Error())
		}
	}
}

func (s *Service) Pop(ctx context.Context,
	in *pb.PopRequest) (*pb.PopReply, error) {

	topic := s.queue.getOrCreateTopic(in.GetTopic())
	var msg *pb.Message
	if in.GetNeedAck() {
		client := ctx.Value("client").(*client)
		inflight := client.getOrCreateInflight(topic)
		inflight.cond.L.Lock()
		for inflight.full() {
			inflight.cond.Wait()
		}
		msg = s.queue.Pop(in.GetTopic())
		inflight.push(msg)
		inflight.cond.L.Unlock()
	} else {
		msg = s.queue.Pop(in.GetTopic())
	}
	if msg != nil {
		return &pb.PopReply{
			Message: msg,
		}, nil
	}
	return nil, status.Error(codes.Unavailable, "msg is nil")
}

func (s *Service) Ack(ctx context.Context,
	in *pb.AckRequest) (*pb.AckReply, error) {

	client := ctx.Value("client").(*client)
	topic, exist := s.queue.getTopic(in.GetTopic())
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

func (s *Service) Join(ctx context.Context,
	in *pb.JoinRequest) (*pb.JoinReply, error) {

	if err := s.queue.join(in.GetRaftAddr(), in.GetNodeID()); err != nil {
		return nil, err
	}
	return &pb.JoinReply{}, nil
}

func (s *Service) Leave(ctx context.Context,
	in *pb.LeaveRequest) (*pb.LeaveReply, error) {

	if err := s.queue.leave(in.GetNodeID()); err != nil {
		return nil, err
	}
	return &pb.LeaveReply{}, nil
}

func (s *Service) Snapshot(ctx context.Context,
	in *pb.SnapshotRequest) (*pb.SnapshotReply, error) {

	return nil, nil
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
	filter, err := newFilter(s.dataDir, topic)
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

	s.queue.close()
}
