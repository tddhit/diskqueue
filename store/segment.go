package store

import (
	"fmt"
	"os"
	"sync/atomic"

	"github.com/gogo/protobuf/proto"
	"github.com/tddhit/tools/log"
	"github.com/tddhit/tools/mmap"

	"github.com/tddhit/diskqueue/pb"
)

const (
	maxSegmentSize = 1 << 27 // 128M
	maxMsgSize     = 1 << 20 // 1M
)

type segment struct {
	minID    uint64
	writeID  uint64
	writePos int64
	ctime    int64
	path     string
	file     *mmap.MmapFile
}

func newSegment(
	path string,
	mode int,
	advise int,
	minID uint64,
	writeID uint64,
	writePos int64,
	ctime int64) (*segment, error) {

	s := &segment{
		minID:    minID,
		writeID:  writeID,
		writePos: writePos,
		ctime:    ctime,
		path:     path,
	}
	file, err := mmap.New(path, maxSegmentSize+2*maxMsgSize, mode, advise)
	if err != nil {
		log.Fatal(err)
	}
	s.file = file
	return s, nil
}

func (s *segment) full() bool {
	size := atomic.LoadInt64(&s.writePos)
	if size < maxSegmentSize {
		return false
	}
	return true
}

func (s *segment) writeOne(msg *diskqueuepb.Message) error {
	buf, err := proto.Marshal(msg)
	if err != nil {
		log.Error(err)
		return err
	}
	len := uint32(len(buf))
	if len > maxMsgSize {
		return fmt.Errorf("invalid msg:size(%d)>maxSize(%d)", len, maxMsgSize)
	}
	offset := atomic.LoadInt64(&s.writePos)
	if err := s.file.PutUint32At(offset, len); err != nil {
		return err
	}
	offset += 4
	if err := s.file.WriteAt(buf, offset); err != nil {
		return err
	}
	offset += int64(len)
	atomic.StoreInt64(&s.writePos, offset)
	atomic.AddUint64(&s.writeID, 1)
	return nil
}

func (s *segment) readOne(msgID uint64, pos int64) (*diskqueuepb.Message, int64) {
	msg := &diskqueuepb.Message{}
	len, err := s.file.Uint32At(pos)
	if err != nil {
		log.Fatal(err)
	}
	pos += 4
	buf, err := s.file.ReadAt(pos, int64(len))
	if err != nil {
		log.Fatal(err)
	}
	pos += int64(len)
	if err := proto.Unmarshal(buf, msg); err != nil {
		log.Fatal(err)
	}
	if msg.ID != msgID {
		log.Fatalf("msg.ID(%d) != msgID(%d)", msg.ID, msgID)
	}
	return msg, pos
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
		log.Warnf("delete segment(%d-%d)", s.minID, s.writeID)
		s.file = nil
	}
	return nil
}

type segments []*segment

func (s segments) Len() int           { return len(s) }
func (s segments) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s segments) Less(i, j int) bool { return s[i].minID < s[j].minID }
