package cluster

import (
	"io"

	"github.com/gogo/protobuf/proto"
	"github.com/hashicorp/raft"
	"github.com/tddhit/tools/log"

	pb "github.com/tddhit/diskqueue/pb"
)

type queue interface {
	Push(topic string, data []byte) error
	Pop(topic string) *pb.Message
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
	var cmd pb.Command
	if err := proto.Unmarshal(l.Data, &cmd); err != nil {
		log.Panic(err)
	}
	switch cmd.Op {
	case pb.Command_PUSH:
		return f.queue.Push(cmd.Topic, cmd.Data)
	case pb.Command_POP:
		return f.queue.Pop(cmd.Topic)
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
