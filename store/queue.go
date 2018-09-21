package store

import (
	"sync"

	"github.com/tddhit/tools/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/tddhit/diskqueue/pb"
)

type Queue struct {
	sync.RWMutex
	dataDir string
	topics  map[string]*topic
}

func NewQueue(dataDir string) *Queue {
	return &Queue{
		dataDir: dataDir,
		topics:  make(map[string]*topic),
	}
}

func (s *Queue) HasTopic(topic string) bool {
	_, ok := s.GetTopic(topic)
	return ok
}

func (s *Queue) MayContain(topic string, key []byte) bool {
	t, exist := s.GetTopic(topic)
	return exist && t.filter.mayContain(key)
}

func (s *Queue) Push(topic string, data, hashKey []byte) error {
	t := s.GetOrCreateTopic(topic)
	if len(hashKey) == 0 {
		hashKey = data
	}
	if err := t.filter.add(hashKey); err != nil {
		return status.Error(codes.Internal, err.Error())
	}
	if err := t.push(data); err != nil {
		return err
	}
	return nil
}

func (s *Queue) GetMessage(topic string) (*pb.Message, int64, error) {
	t := s.GetOrCreateTopic(topic)
	return t.get()
}

func (s *Queue) Advance(topic string, pos int64) {
	t := s.GetOrCreateTopic(topic)
	t.advance(pos)
}

func (s *Queue) GetTopic(name string) (*topic, bool) {
	s.RLock()
	defer s.RUnlock()

	t, ok := s.topics[name]
	return t, ok
}

func (s *Queue) GetOrCreateTopic(name string) *topic {
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
	topic, err := newTopic(s.dataDir, name)
	if err != nil {
		log.Fatal(err)
	}
	s.topics[name] = topic
	s.Unlock()
	return topic
}

func (s *Queue) Close() error {
	log.Debug("Close")
	s.Lock()
	defer s.Unlock()

	for _, t := range s.topics {
		t.Close()
	}
	return nil
}
