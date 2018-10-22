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
	*store.Queue
	raftNode   *cluster.RaftNode
	topicLocks map[string]*sync.RWMutex
}

func newClusterStore(dataDir, raftAddr string,
	nodeID, leaderAddr string) (*clusterStore, error) {

	queue := store.NewQueue(dataDir)
	node, err := cluster.NewRaftNode(dataDir, raftAddr, nodeID, leaderAddr, queue)
	if err != nil {
		return nil, err
	}
	return &clusterStore{
		Queue:      queue,
		raftNode:   node,
		topicLocks: make(map[string]*sync.RWMutex),
	}, nil
}

func (s *clusterStore) getOrCreateLock(topic string) *sync.RWMutex {
	s.RLock()
	if m, ok := s.topicLocks[topic]; ok {
		s.RUnlock()
		return m
	}
	s.RUnlock()

	s.Lock()
	if m, ok := s.topicLocks[topic]; ok {
		s.Unlock()
		return m
	}
	m := &sync.RWMutex{}
	s.topicLocks[topic] = m
	s.Unlock()
	return m
}

func (s *clusterStore) Push(topic string, data, hashKey []byte) error {
	if s.raftNode.State() != raft.Leader {
		return errNotLeader
	}
	cmd, err := proto.Marshal(&pb.Command{
		Op:      pb.Command_PUSH,
		Topic:   topic,
		Data:    data,
		HashKey: hashKey,
	})
	if err != nil {
		log.Error(err)
		return err
	}
	f := s.raftNode.Apply(cmd, 10*time.Second)
	return f.Error()
}

func (s *clusterStore) Pop(topic, channel string) (*pb.Message, error) {
	if s.raftNode.State() != raft.Leader {
		return nil, errNotLeader
	}
	mutex := s.getOrCreateLock(topic)
	mutex.Lock()
	defer mutex.Unlock()

	msg, err := s.GetMessage(topic, channel)
	if err != nil {
		return nil, err
	}
	cmd, err := proto.Marshal(&pb.Command{
		Op:      pb.Command_ADVANCE,
		Topic:   topic,
		Channel: channel,
	})
	if err != nil {
		return nil, err
	}
	f := s.raftNode.Apply(cmd, 10*time.Second)
	if err := f.Error(); err != nil {
		return nil, err
	}
	return msg, nil
}

func (s *clusterStore) Join(raftAddr, nodeID string) error {
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

func (s *clusterStore) Leave(nodeID string) error {
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

func (s *clusterStore) Snapshot() error {
	return nil
}

func (s *clusterStore) GetState() uint32 {
	return uint32(s.raftNode.State())
}

func (s *clusterStore) Close() error {
	log.Debug("Close")
	s.raftNode.Close()
	s.Queue.Close()
	return nil
}
