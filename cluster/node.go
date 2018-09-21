package cluster

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
	box "github.com/tddhit/box/transport"
	"github.com/tddhit/tools/log"

	pb "github.com/tddhit/diskqueue/pb"
)

type RaftNode struct {
	*raft.Raft
	fsm *fsm
}

func NewRaftNode(dir, addr, id, leaderAddr string, q queue) (*RaftNode, error) {
	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(id)
	fsm := newFSM(q)
	if err := os.MkdirAll(filepath.Join(dir, "wal"), 0755); err != nil {
		return nil, err
	}
	boltdb, err := raftboltdb.NewBoltStore(filepath.Join(dir, "wal", id+".log"))
	if err != nil {
		return nil, err
	}
	ss, err := raft.NewFileSnapshotStore(dir, 2, nil)
	if err != nil {
		return nil, err
	}
	netAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil, err
	}
	transport, err := raft.NewTCPTransport(addr, netAddr, 3, 10*time.Second, nil)
	if err != nil {
		return nil, err
	}
	r, err := raft.NewRaft(config, fsm, boltdb, boltdb, ss, transport)
	if err != nil {
		return nil, err
	}
	node := &RaftNode{
		Raft: r,
		fsm:  fsm,
	}
	bootstrap := leaderAddr == ""
	if bootstrap {
		configuration := raft.Configuration{
			Servers: []raft.Server{
				{
					ID:      config.LocalID,
					Address: transport.LocalAddr(),
				},
			},
		}
		node.Raft.BootstrapCluster(configuration)
	} else {
		conn, err := box.Dial(leaderAddr)
		if err != nil {
			log.Fatal(err)
		}
		client := pb.NewDiskqueueGrpcClient(conn)
		_, err = client.Join(context.Background(), &pb.JoinRequest{
			RaftAddr: addr,
			NodeID:   id,
		})
		if err != nil {
			log.Fatal(err)
		}
	}
	return node, nil
}

func (r *RaftNode) Close() error {
	return nil
}
