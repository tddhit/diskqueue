package cluster

import "github.com/hashicorp/raft"

type snapshot struct {
}

func (s *snapshot) Persist(sink raft.SnapshotSink) error {
	return nil
}

func (s *snapshot) Release() {
}
