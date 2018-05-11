package core

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path"
	"sort"
	"sync/atomic"
	"syscall"
	"unsafe"

	"github.com/tddhit/diskqueue/types"
	"github.com/tddhit/tools/log"
)

const (
	MaxSegmentSize = 1 << 22 // 512M
	MaxMsgSize     = 1 << 22 // 4M
	MaxMapSize     = 1 << 30 // 1G
	IndexInterval  = 1 << 19 // 512K
)

var (
	ErrEmptyIndexs   = errors.New("empty indexs")
	ErrNotFoundIndex = errors.New("not found index")
	ErrNotFoundMsgid = errors.New("not found msgid")
)

type segment struct {
	minMsgid   uint64
	maxMsgid   uint64
	size       uint32
	indexCount uint32

	indexFile *os.File
	logFile   *os.File
	indexBuf  *[MaxMapSize]byte
	logBuf    *[MaxMapSize]byte

	writeBuf bytes.Buffer
	indexs   indexs
}

func newSegment(
	dataPath string,
	name string,
	flag int,
	minMsgid uint64,
	maxMsgid uint64,
	meta ...uint32) (*segment, error) {

	s := &segment{
		minMsgid: minMsgid,
		maxMsgid: maxMsgid,
	}
	fileName := fmt.Sprintf(path.Join(dataPath, "%s.diskqueue.%d.log"),
		name, minMsgid)
	f, err := os.OpenFile(fileName, flag, 0600)
	if err != nil {
		return nil, err
	}
	s.logFile = f
	fileName = fmt.Sprintf(path.Join(dataPath, "%s.diskqueue.%d.idx"),
		name, minMsgid)
	f, err = os.OpenFile(fileName, flag, 0600)
	if err != nil {
		return nil, err
	}
	s.indexFile = f
	if err := s.mmap(); err != nil {
		return nil, err
	}
	if len(meta) == 2 {
		s.size = meta[0]
		s.indexCount = meta[1]
	}
	if err := s.loadIndex(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *segment) mmap() error {
	buf, err := syscall.Mmap(int(s.logFile.Fd()), 0, MaxMapSize, syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		return err
	}
	if _, _, err := syscall.Syscall(syscall.SYS_MADVISE, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)), uintptr(syscall.MADV_RANDOM)); err != 0 {
		return err
	}
	s.logBuf = (*[MaxMapSize]byte)(unsafe.Pointer(&buf[0]))

	buf, err = syscall.Mmap(int(s.indexFile.Fd()), 0, MaxMapSize, syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		return err
	}
	if _, _, err := syscall.Syscall(syscall.SYS_MADVISE, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)), uintptr(syscall.MADV_RANDOM)); err != 0 {
		return err
	}
	s.indexBuf = (*[MaxMapSize]byte)(unsafe.Pointer(&buf[0]))
	return nil
}

func (s *segment) loadIndex() error {
	var (
		offset uint32
		pos    uint32
	)
	indexCount := atomic.LoadUint32(&s.indexCount)
	i := 0
	for indexCount != 0 {
		buf := bytes.NewBuffer(s.indexBuf[i : i+4 : i+4])
		err := binary.Read(buf, binary.BigEndian, &offset)
		if err != nil {
			return err
		}
		i += 4
		buf = bytes.NewBuffer(s.indexBuf[i : i+4 : i+4])
		err = binary.Read(buf, binary.BigEndian, &pos)
		if err != nil {
			return err
		}
		i += 4
		s.indexs = append(s.indexs, &index{offset, pos})
		log.Debug(offset, pos)
		indexCount--
	}
	return nil
}

func (s *segment) full() bool {
	size := atomic.LoadUint32(&s.size)
	if size < MaxSegmentSize {
		return false
	}
	return true
}

func (s *segment) writeLog(msg *types.Message) error {
	dataLen := uint32(len(msg.Data))
	if dataLen > MaxMsgSize {
		return fmt.Errorf("invalid message write size (%d) maxMsgSize=%d", dataLen, MaxMsgSize)
	}
	s.writeBuf.Reset()
	if err := binary.Write(&s.writeBuf, binary.BigEndian, msg.Id); err != nil {
		return err
	}
	if err := binary.Write(&s.writeBuf, binary.BigEndian, dataLen); err != nil {
		return err
	}
	if _, err := s.writeBuf.Write(msg.Data); err != nil {
		return err
	}
	//s.writeBuf.WriteString(strconv.FormatUint(msg.Id, 10))
	//s.writeBuf.WriteString(",")
	//s.writeBuf.Write(msg.Data)
	//s.writeBuf.WriteString("\n")
	//log.Debug(msg.Id, string(msg.Data))
	if _, err := s.logFile.Write(s.writeBuf.Bytes()); err != nil {
		s.logFile.Close()
		s.logFile = nil
		return err
	}
	return nil
}

func (s *segment) writeIndex(offset, pos uint32) error {
	s.writeBuf.Reset()
	if err := binary.Write(&s.writeBuf, binary.BigEndian, offset); err != nil {
		return err
	}
	if err := binary.Write(&s.writeBuf, binary.BigEndian, pos); err != nil {
		return err
	}
	if _, err := s.indexFile.Write(s.writeBuf.Bytes()); err != nil {
		s.indexFile.Close()
		s.indexFile = nil
		return err
	}
	s.indexs = append(s.indexs, &index{offset, pos})
	return nil
}

func (s *segment) writeOne(msg *types.Message) error {
	if err := s.writeLog(msg); err != nil {
		return err
	}
	dataLen := 12 + uint32(len(msg.Data))
	size := atomic.AddUint32(&s.size, dataLen)
	oldSize := size - dataLen
	if (size/IndexInterval)-(oldSize/IndexInterval) > 0 || oldSize == 0 {
		offset := uint32(msg.Id - s.minMsgid)
		if err := s.writeIndex(offset, oldSize); err != nil {
			return err
		}
		atomic.AddUint32(&s.indexCount, 1)
	}
	atomic.AddUint64(&s.maxMsgid, 1)
	return nil
}

func (s *segment) seek(msgid uint64) (pos uint32, err error) {
	if len(s.indexs) == 0 {
		return
	}
	offset := uint32(msgid - s.minMsgid)
	index := sort.Search(len(s.indexs), func(i int) bool {
		return s.indexs[i].offset > offset
	})
	if index == 0 {
		err = ErrNotFoundIndex
		return
	}
	pos = s.indexs[index-1].pos
	log.Debug("seg seek:", index-1, s.indexs[index-1].offset, s.indexs[index-1].pos)
	return
}

func (s *segment) readOne(msgid uint64, pos uint32) (msg *types.Message, nextPos uint32, err error) {
	var (
		id  uint64
		len uint32
	)
	for {
		buf := bytes.NewBuffer(s.logBuf[pos : pos+8 : pos+8])
		err = binary.Read(buf, binary.BigEndian, &id)
		if err != nil {
			return
		}
		pos += 8
		buf = bytes.NewBuffer(s.logBuf[pos : pos+4 : pos+4])
		err = binary.Read(buf, binary.BigEndian, &len)
		if err != nil {
			return
		}
		pos += 4
		log.Debugf("seg readOne\tid=%d\tlen=%d\tmsgid=%d\tpos=%d\n", id, len, msgid, pos)
		if id < msgid {
			pos += len
			continue
		} else if id == msgid {
			nextPos = pos + len
			data := s.logBuf[pos : pos+len : pos+len]
			msg = &types.Message{id, data}
			return
		} else {
			err = ErrNotFoundMsgid
			return
		}
	}
}

func (s *segment) exit() {
	s.sync()
	if s.indexFile != nil {
		s.indexFile.Close()
		s.indexFile = nil
	}

	if s.logFile != nil {
		s.logFile.Close()
		s.logFile = nil
	}
}

func (s *segment) sync() error {
	if s.indexFile != nil {
		err := s.indexFile.Sync()
		if err != nil {
			s.indexFile.Close()
			s.indexFile = nil
			return err
		}
	}
	if s.logFile != nil {
		err := s.logFile.Sync()
		if err != nil {
			s.logFile.Close()
			s.logFile = nil
			return err
		}
	}
	return nil
}

type index struct {
	offset uint32
	pos    uint32
}

type indexs []*index

func (s indexs) Len() int           { return len(s) }
func (s indexs) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s indexs) Less(i, j int) bool { return s[i].offset < s[j].offset }

type segments []*segment

func (s segments) Len() int           { return len(s) }
func (s segments) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s segments) Less(i, j int) bool { return s[i].minMsgid < s[j].minMsgid }
