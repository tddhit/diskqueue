package service

import (
	"sync"

	"github.com/tddhit/tools/log"

	pb "github.com/tddhit/diskqueue/pb"
	"github.com/tddhit/diskqueue/store"
)

type standaloneStore struct {
	sync.RWMutex
	dataDir string
	topics  map[string]*store.Topic
}

func newStandaloneStore(dataDir string) *standaloneStore {
	return &standaloneStore{
		dataDir: dataDir,
		topics:  make(map[string]*store.Topic),
	}
}

func (s *standaloneStore) Push(topic string, data []byte) error {
	t := s.getOrCreateTopic(topic)
	if err := t.PutMessage(data); err != nil {
		return err
	}
	return nil
}

func (s *standaloneStore) Pop(topic string) *pb.Message {
	t := s.getOrCreateTopic(topic)
	return t.GetMessage()
}

func (s *standaloneStore) getTopic(name string) (*store.Topic, bool) {
	s.RLock()
	defer s.RUnlock()

	t, ok := s.topics[name]
	return t, ok
}

func (s *standaloneStore) getOrCreateTopic(name string) *store.Topic {
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
	topic, err := store.NewTopic(s.dataDir, name)
	if err != nil {
		log.Fatal(err)
	}
	s.topics[name] = topic
	s.Unlock()
	return topic
}

func (s *standaloneStore) join(raftAddr, nodeID string) error {
	return nil
}
func (s *standaloneStore) leave(nodeID string) error {
	return nil
}
func (s *standaloneStore) snapshot() error {
	return nil
}

func (s *standaloneStore) close() error {
	return nil
}
