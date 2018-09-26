package resolver

import (
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/raft"
	"github.com/tddhit/box/transport"
	"github.com/tddhit/tools/log"
	"golang.org/x/net/context"
	"google.golang.org/grpc/resolver"

	pb "github.com/tddhit/diskqueue/pb"
)

func init() {
	resolver.Register(NewBuilder())
}

func NewBuilder() resolver.Builder {
	return &dqBuilder{}
}

type dqBuilder struct {
}

func (b *dqBuilder) Scheme() string {
	return "diskqueue"
}

func (b *dqBuilder) Build(target resolver.Target,
	cc resolver.ClientConn, opts resolver.BuildOption) (resolver.Resolver, error) {

	if target.Authority == "" {
		return nil, errors.New("no diskqueue endpoints")
	}
	endpoints := strings.Split(target.Authority, ",")
	r := &dqResolver{
		cc:        cc,
		rn:        make(chan struct{}, 1),
		endpoints: endpoints,
	}
	r.ctx, r.cancel = context.WithCancel(context.Background())
	r.wg.Add(2)
	go r.watchLeader()
	go r.watcher()
	return r, nil
}

type dqResolver struct {
	ctx       context.Context
	cancel    context.CancelFunc
	endpoints []string
	cc        resolver.ClientConn
	rn        chan struct{}
	wg        sync.WaitGroup
	exitFlag  int32
}

func (r *dqResolver) ResolveNow(opt resolver.ResolveNowOption) {
	select {
	case r.rn <- struct{}{}:
	default:
	}
}

func (r *dqResolver) getLeader() (string, error) {
	log.Info("Try to get the leader of diskqueue.")
	var (
		wg             sync.WaitGroup
		replies        = make(map[string]*pb.GetStateReply)
		leaderEndpoint string
	)
	for _, endpoint := range r.endpoints {
		wg.Add(1)
		go func(endpoint string) {
			defer wg.Done()
			conn, err := transport.Dial("grpc://" + endpoint)
			if err != nil {
				log.Error(err)
				return
			}
			client := pb.NewDiskqueueGrpcClient(conn)
			reply, err := client.GetState(r.ctx, &pb.GetStateRequest{})
			if err != nil {
				log.Error(err)
				return
			}
			replies[endpoint] = reply
		}(endpoint)
	}
	wg.Wait()
	for endpoint, reply := range replies {
		if reply.GetState() == uint32(raft.Leader) {
			if leaderEndpoint == "" {
				leaderEndpoint = endpoint
			} else {
				return "", errors.New("more than one leader")
			}
		}
	}
	if leaderEndpoint == "" {
		return "", errors.New("no leader")
	}
	return leaderEndpoint, nil
}

func (r *dqResolver) watchLeader() {
	defer r.wg.Done()
	for {
		if atomic.LoadInt32(&r.exitFlag) == 1 {
			goto exit
		}
		leaderEndpoint, err := r.getLeader()
		if err != nil {
			time.Sleep(time.Second)
			continue
		}
		r.ResolveNow(resolver.ResolveNowOption{})
		conn, err := transport.Dial("grpc://" + leaderEndpoint)
		if err != nil {
			log.Error(err)
			time.Sleep(time.Second)
			continue
		}
		client := pb.NewDiskqueueGrpcClient(conn)
		streamClient, err := client.WatchState(r.ctx, &pb.WatchStateRequest{})
		if err != nil {
			log.Error(err)
			time.Sleep(time.Second)
			continue
		}
		for {
			reply, err := streamClient.Recv()
			if err != nil || reply.GetState() != uint32(raft.Leader) {
				log.Error("Disconnect the leader of the diskqueue", err, reply.GetState())
				break
			}
		}
	}
exit:
}

func (r *dqResolver) watcher() {
	defer r.wg.Done()
	for {
		select {
		case <-r.ctx.Done():
			return
		case <-r.rn:
		}
		leaderEndpoint, err := r.getLeader()
		if err != nil {
			log.Error(err)
		} else {
			r.cc.NewAddress([]resolver.Address{
				{Addr: leaderEndpoint},
			})
		}
	}
}

func (r *dqResolver) Close() {
	if !atomic.CompareAndSwapInt32(&r.exitFlag, 0, 1) {
		log.Info("already closed")
		return
	}
	r.cancel()
	r.wg.Wait()
}
