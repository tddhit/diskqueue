package service

import (
	"context"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/urfave/cli"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/tddhit/box/mw"
	"github.com/tddhit/diskqueue/pb"
	"github.com/tddhit/tools/dirlock"
	"github.com/tddhit/tools/log"
)

type Service struct {
	sync.RWMutex
	queue         *clusterStore
	clients       map[string]*client
	dataDir       string
	dl            *dirlock.DirLock
	filterCounter *prometheus.CounterVec
	exitC         chan struct{}
}

func NewService(ctx *cli.Context) *Service {
	dataDir := ctx.String("datadir")
	raftAddr := ctx.String("cluster-addr")
	nodeID := ctx.String("id")
	leaderAddr := ctx.String("leader")
	if raftAddr == "" || nodeID == "" {
		log.Fatal("invalid params")
	}
	if !mw.IsWorker() {
		return nil
	}
	cs, err := newClusterStore(dataDir, raftAddr, nodeID, leaderAddr)
	if err != nil {
		log.Fatal(err)
	}
	s := &Service{
		queue:   cs,
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
	in *diskqueuepb.PushReq) (*diskqueuepb.PushRsp, error) {

	id, err := s.queue.Push(in.GetTopic(), in.GetData(), in.GetHashKey())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	log.Debugf("push\t%s\t%d\t%d\t%s", in.Topic, id, len(in.Data), string(in.Data))
	return &diskqueuepb.PushRsp{ID: id}, nil
}

func (s *Service) Pop(ctx context.Context,
	in *diskqueuepb.PopReq) (*diskqueuepb.PopRsp, error) {

	var (
		msg *diskqueuepb.Message
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
	return &diskqueuepb.PopRsp{Message: msg}, nil
}

func (s *Service) Ack(ctx context.Context,
	in *diskqueuepb.AckReq) (*diskqueuepb.AckRsp, error) {

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
	return &diskqueuepb.AckRsp{}, nil
}

func (s *Service) Join(ctx context.Context,
	in *diskqueuepb.JoinReq) (*diskqueuepb.JoinRsp, error) {

	if err := s.queue.Join(in.GetRaftAddr(), in.GetNodeID()); err != nil {
		return nil, err
	}
	return &diskqueuepb.JoinRsp{}, nil
}

func (s *Service) Leave(ctx context.Context,
	in *diskqueuepb.LeaveReq) (*diskqueuepb.LeaveRsp, error) {

	if err := s.queue.Leave(in.GetNodeID()); err != nil {
		return nil, err
	}
	return &diskqueuepb.LeaveRsp{}, nil
}

func (s *Service) Snapshot(ctx context.Context,
	in *diskqueuepb.SnapshotReq) (*diskqueuepb.SnapshotRsp, error) {

	return nil, nil
}

func (s *Service) GetState(ctx context.Context,
	in *diskqueuepb.GetStateReq) (*diskqueuepb.GetStateRsp, error) {

	return &diskqueuepb.GetStateRsp{State: s.queue.GetState()}, nil
}

func (s *Service) WatchState(in *diskqueuepb.WatchStateReq,
	stream diskqueuepb.Diskqueue_WatchStateServer) error {

	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ticker.C:
			err := stream.Send(&diskqueuepb.WatchStateRsp{State: s.queue.GetState()})
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
