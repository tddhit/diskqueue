package service

import (
	"context"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/tddhit/box/mw"
	"github.com/tddhit/tools/dirlock"
	"github.com/tddhit/tools/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/tddhit/diskqueue/pb"
)

type queue interface {
	Push(topic string, data, hashKey []byte) error
	Pop(topic, channel string) (*pb.Message, error)
	MayContain(topic string, key []byte) bool
	HasTopic(topic string) bool
	Join(raftAddr, nodeID string) error
	Leave(nodeID string) error
	Snapshot() error
	GetState() uint32
	Close() error
}

type Service struct {
	sync.RWMutex
	queue         queue
	clients       map[string]*client
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

	if !in.GetIgnoreFilter() {
		hashKey := in.GetData()
		if in.GetHashKey() != nil {
			hashKey = in.GetHashKey()
		}
		if s.queue.MayContain(in.GetTopic(), hashKey) {
			s.filterCounter.WithLabelValues().Inc()
			log.Debug("filter:", string(hashKey))
			return &pb.PushReply{}, nil
		}
	}
	if err := s.queue.Push(in.GetTopic(),
		in.GetData(), in.GetHashKey()); err != nil {

		return nil, status.Error(codes.Internal, err.Error())
	}
	return &pb.PushReply{}, nil
}

func (s *Service) Pop(ctx context.Context,
	in *pb.PopRequest) (*pb.PopReply, error) {

	var (
		msg *pb.Message
		err error
	)
	if in.GetNeedAck() {
		client := ctx.Value("client").(*client)
		inflight := client.getOrCreateInflight(in.GetTopic(), s.queue)
		inflight.cond.L.Lock()
		for inflight.full() {
			inflight.cond.Wait()
		}
		msg, err = s.queue.Pop(in.GetTopic(), in.GetChannel())
		if err != nil {
			inflight.cond.L.Unlock()
			return nil, err
		}
		inflight.push(msg)
		inflight.cond.L.Unlock()
	} else {
		msg, err = s.queue.Pop(in.GetTopic(), in.GetChannel())
		if err != nil {
			return nil, err
		}
	}
	return &pb.PopReply{Message: msg}, nil
}

func (s *Service) Ack(ctx context.Context,
	in *pb.AckRequest) (*pb.AckReply, error) {

	client := ctx.Value("client").(*client)
	if !s.queue.HasTopic(in.GetTopic()) {
		return nil, status.Error(codes.FailedPrecondition,
			"not found topic in diskqueue")
	}
	inflight, exist := client.getInflight(in.GetTopic())
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

	if err := s.queue.Join(in.GetRaftAddr(), in.GetNodeID()); err != nil {
		return nil, err
	}
	return &pb.JoinReply{}, nil
}

func (s *Service) Leave(ctx context.Context,
	in *pb.LeaveRequest) (*pb.LeaveReply, error) {

	if err := s.queue.Leave(in.GetNodeID()); err != nil {
		return nil, err
	}
	return &pb.LeaveReply{}, nil
}

func (s *Service) Snapshot(ctx context.Context,
	in *pb.SnapshotRequest) (*pb.SnapshotReply, error) {

	return nil, nil
}

func (s *Service) GetState(ctx context.Context,
	in *pb.GetStateRequest) (*pb.GetStateReply, error) {

	return &pb.GetStateReply{State: s.queue.GetState()}, nil
}

func (s *Service) WatchState(in *pb.WatchStateRequest,
	stream pb.Diskqueue_WatchStateServer) error {

	for {
		err := stream.Send(&pb.WatchStateReply{State: s.queue.GetState()})
		if err != nil {
			return err
		}
	}
	return nil
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

func (s *Service) Close() {
	log.Debug("Close")
	s.Lock()
	defer s.Unlock()

	s.queue.Close()
}
