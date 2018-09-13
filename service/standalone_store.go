package service

import (
	"sync"

	pb "github.com/tddhit/diskqueue/pb"
	"github.com/tddhit/diskqueue/store"
)

type standaloneStore struct {
	sync.RWMutex
	*store.Queue
}

func newStandaloneStore(dataDir string) *standaloneStore {
	return &standaloneStore{
		Queue: store.NewQueue(dataDir),
	}
}

func (s *standaloneStore) Pop(topic string) (*pb.Message, error) {
	s.Lock()
	defer s.Unlock()

	msg, pos, err := s.Queue.GetMessage(topic)
	if err != nil {
		return nil, err
	}
	s.Queue.Advance(topic, pos)
	return msg, nil
}

func (s *standaloneStore) Join(raftAddr, nodeID string) error {
	return nil
}
func (s *standaloneStore) Leave(nodeID string) error {
	return nil
}
func (s *standaloneStore) Snapshot() error {
	return nil
}

func (s *standaloneStore) Close() error {
	return nil
}
