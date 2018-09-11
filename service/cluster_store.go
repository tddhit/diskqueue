package service

import (
	"errors"
	"sync"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/hashicorp/raft"
	"github.com/tddhit/tools/log"

	"github.com/tddhit/diskqueue/cluster"
	pb "github.com/tddhit/diskqueue/pb"
	"github.com/tddhit/diskqueue/store"
)

var (
	errNotLeader = errors.New("not a leader")
)

type clusterStore struct {
	sync.RWMutex
	raftNode *cluster.RaftNode
	ss       *standaloneStore
	conds    map[string]*sync.Cond
}

func newClusterStore(dataDir, raftAddr string,
	nodeID, leaderAddr string) (*clusterStore, error) {

	ss := newStandaloneStore(dataDir)
	node, err := cluster.NewRaftNode(dataDir, raftAddr, nodeID, leaderAddr, ss)
	if err != nil {
		return nil, err
	}
	return &clusterStore{
		raftNode: node,
		ss:       ss,
		conds:    make(map[string]*sync.Cond),
	}, nil
}

func (s *clusterStore) getOrCreateCond(topic string) *sync.Cond {
	s.RLock()
	cond, ok := s.conds[topic]
	if ok {
		s.RUnlock()
		return cond
	}
	s.RUnlock()

	s.Lock()
	cond, ok = s.conds[topic]
	if ok {
		s.Unlock()
		return cond
	}
	cond = sync.NewCond(&sync.Mutex{})
	s.conds[topic] = cond
	s.Unlock()
	return cond
}

func (s *clusterStore) Push(topic string, data []byte) error {
	if s.raftNode.State() != raft.Leader {
		return errNotLeader
	}
	cond := s.getOrCreateCond(topic)
	cond.Signal()
	cmd, err := proto.Marshal(&pb.Command{
		Topic: topic,
		Data:  data,
		Op:    pb.Command_PUSH,
	})
	if err != nil {
		log.Error(err)
		return err
	}
	f := s.raftNode.Apply(cmd, 10*time.Second)
	return f.Error()
}

func (s *clusterStore) Pop(topic string) *pb.Message {
	if s.raftNode.State() != raft.Leader {
		return nil
	}
	cond := s.getOrCreateCond(topic)
	t := s.getOrCreateTopic(topic)
	cond.L.Lock()
	for t.Empty() {
		cond.Wait()
	}
	cond.L.Unlock()
	cmd, err := proto.Marshal(&pb.Command{
		Topic: topic,
		Op:    pb.Command_POP,
	})
	if err != nil {
		log.Error(err)
		return nil
	}
	f := s.raftNode.Apply(cmd, 10*time.Second)
	log.Debug("pop...")
	if err := f.Error(); err != nil {
		return nil
	}
	log.Debug("pop...end")
	return f.Response().(*pb.Message)
}

func (s *clusterStore) getTopic(name string) (*store.Topic, bool) {
	return s.ss.getTopic(name)
}

func (s *clusterStore) getOrCreateTopic(name string) *store.Topic {
	return s.ss.getOrCreateTopic(name)
}

func (s *clusterStore) join(raftAddr, nodeID string) error {
	config := s.raftNode.GetConfiguration()
	if err := config.Error(); err != nil {
		log.Error(err)
		return err
	}
	for _, server := range config.Configuration().Servers {
		if server.ID == raft.ServerID(nodeID) {
			return nil
		}
	}
	f := s.raftNode.AddVoter(raft.ServerID(nodeID),
		raft.ServerAddress(raftAddr), 0, 0)
	if err := f.Error(); err != nil {
		return err
	}
	return nil
}

func (s *clusterStore) leave(nodeID string) error {
	config := s.raftNode.GetConfiguration()
	if err := config.Error(); err != nil {
		return err
	}
	for _, server := range config.Configuration().Servers {
		if server.ID == raft.ServerID(nodeID) {
			f := s.raftNode.RemoveServer(server.ID, 0, 0)
			if err := f.Error(); err != nil {
				log.Error(err)
				return err
			}
			return nil
		}
	}
	return nil
}

func (s *clusterStore) snapshot() error {
	return nil
}

func (s *clusterStore) close() error {
	s.raftNode.Close()
	s.ss.close()
	return nil
}
