package service

import (
	"context"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/tddhit/box/mw"
	"github.com/tddhit/tools/dirlock"
	"github.com/tddhit/tools/log"
	"github.com/urfave/cli"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/tddhit/diskqueue/pb"
)

type queue interface {
	Push(topic string, data, hashKey []byte) (uint64, error)
	Pop(topic, channel string) (*pb.Message, error)
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
	exitC         chan struct{}
}

func NewService(ctx *cli.Context) *Service {
	dataDir := ctx.String("datadir")
	mode := ctx.String("mode")
	raftAddr := ctx.String("cluster-addr")
	nodeID := ctx.String("id")
	leaderAddr := ctx.String("leader")
	if mode == "cluster" {
		if raftAddr == "" || nodeID == "" {
			log.Fatal("invalid params")
		}
	}
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
		//q = newStandaloneStore(dataDir)
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
		exitC: make(chan struct{}),
	}
	prometheus.MustRegister(s.filterCounter)
	if err := s.dl.Lock(); err != nil {
		log.Fatal(err)
	}
	return s
}

func (s *Service) Push(ctx context.Context,
	in *pb.PushReq) (*pb.PushRsp, error) {

	id, err := s.queue.Push(in.GetTopic(), in.GetData(), in.GetHashKey())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	log.Debugf("push\t%s\t%d\t%d\t%s", in.Topic, id, len(in.Data), string(in.Data))
	return &pb.PushRsp{ID: id}, nil
}

func (s *Service) Pop(ctx context.Context,
	in *pb.PopReq) (*pb.PopRsp, error) {

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
	log.Debugf("pop\t%s\t%s\t%d\t%s", in.Topic, in.Channel, msg.ID, string(msg.Data))
	return &pb.PopRsp{Message: msg}, nil
}

func (s *Service) Ack(ctx context.Context,
	in *pb.AckReq) (*pb.AckRsp, error) {

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
	return &pb.AckRsp{}, nil
}

func (s *Service) Join(ctx context.Context,
	in *pb.JoinReq) (*pb.JoinRsp, error) {

	if err := s.queue.Join(in.GetRaftAddr(), in.GetNodeID()); err != nil {
		return nil, err
	}
	return &pb.JoinRsp{}, nil
}

func (s *Service) Leave(ctx context.Context,
	in *pb.LeaveReq) (*pb.LeaveRsp, error) {

	if err := s.queue.Leave(in.GetNodeID()); err != nil {
		return nil, err
	}
	return &pb.LeaveRsp{}, nil
}

func (s *Service) Snapshot(ctx context.Context,
	in *pb.SnapshotReq) (*pb.SnapshotRsp, error) {

	return nil, nil
}

func (s *Service) GetState(ctx context.Context,
	in *pb.GetStateReq) (*pb.GetStateRsp, error) {

	return &pb.GetStateRsp{State: s.queue.GetState()}, nil
}

func (s *Service) WatchState(in *pb.WatchStateReq,
	stream pb.Diskqueue_WatchStateServer) error {

	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ticker.C:
			err := stream.Send(&pb.WatchStateRsp{State: s.queue.GetState()})
			if err != nil {
				return err
			}
		case <-s.exitC:
			goto exit
		}
	}
exit:
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
	s.Lock()
	defer s.Unlock()

	close(s.exitC)
	s.queue.Close()
}
