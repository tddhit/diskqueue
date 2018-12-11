package cluster

import (
	"io"

	"github.com/gogo/protobuf/proto"
	"github.com/hashicorp/raft"

	"github.com/tddhit/diskqueue/pb"
	"github.com/tddhit/tools/log"
)

type queue interface {
	Push(topic string, data, hashKey []byte) (uint64, error)
	Advance(topic, channel string, nextPos int64)
}

type fsm struct {
	queue queue
}

func newFSM(q queue) *fsm {
	return &fsm{
		queue: q,
	}
}

func (f *fsm) Apply(l *raft.Log) interface{} {
	var cmd diskqueuepb.Command
	if err := proto.Unmarshal(l.Data, &cmd); err != nil {
		log.Panic(err)
	}
	switch cmd.Op {
	case diskqueuepb.Command_PUSH:
		var rsp struct {
			ID  uint64
			Err error
		}
		rsp.ID, rsp.Err = f.queue.Push(cmd.Topic, cmd.Data, cmd.HashKey)
		return rsp
	case diskqueuepb.Command_ADVANCE:
		f.queue.Advance(cmd.Topic, cmd.Channel, cmd.NextPos)
	default:
		log.Panic("Invalid Op")
	}
	return nil
}

func (f *fsm) Snapshot() (raft.FSMSnapshot, error) {
	return &snapshot{}, nil
}

func (f *fsm) Restore(rc io.ReadCloser) error {
	return nil
}
