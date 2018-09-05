package store

import (
	"fmt"
	"os"
	"sync/atomic"

	"github.com/gogo/protobuf/proto"
	"github.com/tddhit/tools/log"
	"github.com/tddhit/tools/mmap"

	pb "github.com/tddhit/diskqueue/pb"
)

const (
	maxSegmentSize = 1 << 25 // 1G
	maxMsgSize     = 1 << 10 // 1k
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
	path string
	file *mmap.MmapFile
}

func newSegment(path string, mode, advise int, m *smeta) (*segment, error) {
	s := &segment{
		path: path,
		meta: m,
	}
	file, err := mmap.New(path, maxSegmentSize+2*maxMsgSize, mode, advise)
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
	len, err := s.file.Uint32At(offset)
	if err != nil {
		return nil, 0, err
	}
	offset += 4
	buf, err := s.file.ReadAt(offset, int64(len))
	if err != nil {
		return nil, 0, err
	}
	offset += int64(len)
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

func (s *segment) delete() error {
	if s.file != nil {
		if err := s.close(); err != nil {
			return err
		}
		if err := os.Remove(s.path); err != nil {
			return err
		}
		s.file = nil
	}
	return nil
}

type segments []*segment

func (s segments) Len() int           { return len(s) }
func (s segments) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s segments) Less(i, j int) bool { return s[i].meta.MinID < s[j].meta.MinID }
