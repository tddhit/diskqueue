package service

import (
	"sync"

	pb "github.com/tddhit/diskqueue/pb"
	"github.com/tddhit/diskqueue/store"
)

type standaloneStore struct {
	sync.RWMutex
	*store.Queue
	topicLocks map[string]*sync.RWMutex
}

func newStandaloneStore(dataDir string) *standaloneStore {
	return &standaloneStore{
		Queue:      store.NewQueue(dataDir),
		topicLocks: make(map[string]*sync.RWMutex),
	}
}

func (s *standaloneStore) getOrCreateLock(topic string) *sync.RWMutex {
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

func (s *standaloneStore) Pop(topic, channel string) (*pb.Message, error) {
	mutex := s.getOrCreateLock(topic)
	mutex.Lock()
	defer mutex.Unlock()

	msg, err := s.Queue.GetMessage(topic, channel)
	if err != nil {
		return nil, err
	}
	s.Queue.Advance(topic, channel)
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

func (s *standaloneStore) GetState() uint32 {
	return 0
}

func (s *standaloneStore) Close() error {
	return nil
}
