package store

import (
	"sync"

	"github.com/tddhit/diskqueue/pb"
	"github.com/tddhit/tools/log"
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

func (s *Queue) Push(topic string, data, hashKey []byte) (uint64, error) {
	t, err := s.getOrCreateTopic(topic)
	if err != nil {
		return 0, err
	}
	id, err := t.push(data)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (s *Queue) Get(
	topic string,
	channel string) (*diskqueuepb.Message, int64, error) {

	t, err := s.getOrCreateTopic(topic)
	if err != nil {
		return nil, 0, err
	}
	c, _ := t.getOrCreateChannel(channel)
	return c.get()
}

func (s *Queue) Advance(topic, channel string, nextPos int64) {
	t, _ := s.getOrCreateTopic(topic)
	c, _ := t.getOrCreateChannel(channel)
	c.advance(nextPos)
}

func (s *Queue) HasTopic(name string) bool {
	_, ok := s.getTopic(name)
	return ok
}

func (s *Queue) getTopic(name string) (*topic, bool) {
	s.RLock()
	defer s.RUnlock()

	t, ok := s.topics[name]
	return t, ok
}

func (s *Queue) getOrCreateTopic(name string) (*topic, error) {
	s.RLock()
	if t, ok := s.topics[name]; ok {
		s.RUnlock()
		return t, nil
	}
	s.RUnlock()

	s.Lock()
	if t, ok := s.topics[name]; ok {
		s.Unlock()
		return t, nil
	}
	topic, err := newTopic(s.dataDir, name)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	s.topics[name] = topic
	s.Unlock()
	return topic, nil
}

func (s *Queue) Close() {
	s.Lock()
	defer s.Unlock()

	for _, t := range s.topics {
		t.close()
	}
}
