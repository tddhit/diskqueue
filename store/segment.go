package store

import (
	"fmt"
	"sync/atomic"

	"github.com/gogo/protobuf/proto"
	"github.com/tddhit/tools/log"
	"github.com/tddhit/tools/mmap"

	pb "github.com/tddhit/diskqueue/pb"
)

const (
	maxSegmentSize = 1<<30 + 1<<13 // 1G + 8K
	maxMsgSize     = 1 << 10       // 1k
)

type smeta struct {
	MinID    uint64 `json:"minID"`
	ReadID   uint64 `json:"readID"`
	WriteID  uint64 `json:"writeID"`
	ReadPos  int64  `json:"readPos"`
	WritePos int64  `json:"writePos"`
}

type segment struct {
	meta *smeta
	file *mmap.MmapFile
}

func newSegment(path string, mode, advise int, m *smeta) (*segment, error) {
	s := &segment{
		meta: m,
	}
	file, err := mmap.New(path, maxSegmentSize, mode, advise)
	if err != nil {
		log.Fatal(err)
	}
	s.file = file
	return s, nil
}

func (s *segment) full() bool {
	size := atomic.LoadInt64(&s.meta.WritePos)
	if size < maxSegmentSize {
		return false
	}
	return true
}

func (s *segment) writeOne(msg *pb.Message) error {
	buf, err := proto.Marshal(msg)
	if err != nil {
		log.Error(err)
		return err
	}
	len := uint32(len(buf))
	if len > maxMsgSize {
		return fmt.Errorf("invalid msg:size(%d)>maxSize(%d)", len, maxMsgSize)
	}
	offset := atomic.LoadInt64(&s.meta.WritePos)
	if err := s.file.PutUint32At(offset, len); err != nil {
		return err
	}
	offset += 4
	if err := s.file.WriteAt(buf, offset); err != nil {
		return err
	}
	offset += int64(len)
	atomic.StoreInt64(&s.meta.WritePos, offset)
	atomic.AddUint64(&s.meta.WriteID, 1)
	return nil
}

func (s *segment) readOne(msgID uint64) (*pb.Message, int64, error) {
	msg := &pb.Message{}
	offset := s.meta.ReadPos
	len := int64(s.file.Uint32At(offset))
	offset += 4
	buf := s.file.ReadAt(offset, len)
	offset += len
	if err := proto.Unmarshal(buf, msg); err != nil {
		log.Error(err)
		return nil, 0, err
	}
	if msg.GetID() != msgID {
		return nil, 0, fmt.Errorf("msg.GetID(%d) != msgID(%d)", msg.GetID(), msgID)
	}
	log.Debugf("seg readOne\tid=%d", msg.GetID())
	return msg, offset, nil
}

func (s *segment) sync() error {
	return s.file.Sync()
}

func (s *segment) close() error {
	return s.file.Close()
}

type segments []*segment

func (s segments) Len() int           { return len(s) }
func (s segments) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s segments) Less(i, j int) bool { return s[i].meta.MinID < s[j].meta.MinID }
