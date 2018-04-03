package diskqueue

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"syscall"
	"unsafe"

	"github.com/tddhit/tools/log"
)

const (
	MaxSegmentSize = 1 << 29 // 512M
	MaxMsgSize     = 1 << 22 // 4M
	MaxMapSize     = 1 << 30 // 1G
)

var (
	ErrEmptyIndexs   = errors.New("empty indexs")
	ErrNotFoundIndex = errors.New("not found index")
	ErrNotFoundMsgid = errors.New("not found msgid")
)

type segment struct {
	sync.RWMutex

	minMsgid   uint64
	pos        uint32
	indexCount uint32
	msgCount   uint32

	indexFile *os.File
	logFile   *os.File
	indexBuf  *[MaxMapSize]byte
	logBuf    *[MaxMapSize]byte

	writeBuf bytes.Buffer
	indexs   indexs
}

func newSegment(name string, msgid uint64, flag int, meta ...uint32) (*segment, error) {
	s := &segment{
		minMsgid: msgid,
	}
	fileName := fmt.Sprintf("%s.diskqueue.%d.log", name, msgid)
	f, err := os.OpenFile(fileName, flag, 0600)
	if err != nil {
		return nil, err
	}
	s.logFile = f
	fileName = fmt.Sprintf("%s.diskqueue.%d.idx", name, msgid)
	f, err = os.OpenFile(fileName, flag, 0600)
	if err != nil {
		return nil, err
	}
	s.indexFile = f
	if err := s.mmap(); err != nil {
		return nil, err
	}
	if len(meta) == 3 {
		s.pos = meta[0]
		s.indexCount = meta[1]
		s.msgCount = meta[2]
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
	for indexCount != 0 {
		buf := bytes.NewBuffer(s.indexBuf[pos : pos+4 : pos+4])
		err := binary.Read(buf, binary.BigEndian, &offset)
		if err != nil {
			return err
		}
		pos += 4
		buf = bytes.NewBuffer(s.indexBuf[pos : pos+4 : pos+4])
		err = binary.Read(buf, binary.BigEndian, &pos)
		if err != nil {
			return err
		}
		s.indexs = append(s.indexs, &index{offset, pos})
		pos += 4
		indexCount--
	}
	return nil
}

func (s *segment) full() bool {
	size := atomic.LoadUint32(&s.pos)
	if size < MaxSegmentSize {
		return false
	}
	return true
}

func (s *segment) writeLog(msg *Message) (uint32, error) {
	dataLen := uint32(len(msg.Data))
	if dataLen > MaxMsgSize {
		return 0, fmt.Errorf("invalid message write size (%d) maxMsgSize=%d", dataLen, MaxMsgSize)
	}
	s.writeBuf.Reset()
	if err := binary.Write(&s.writeBuf, binary.BigEndian, msg.Id); err != nil {
		return 0, err
	}
	if err := binary.Write(&s.writeBuf, binary.BigEndian, dataLen); err != nil {
		return 0, err
	}
	if _, err := s.writeBuf.Write(msg.Data); err != nil {
		return 0, err
	}
	if _, err := s.logFile.Write(s.writeBuf.Bytes()); err != nil {
		s.logFile.Close()
		s.logFile = nil
		return 0, err
	}
	pos := atomic.LoadUint32(&s.pos)
	atomic.AddUint32(&s.pos, 12+dataLen)
	return pos, nil
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
	s.Lock()
	s.indexs = append(s.indexs, &index{offset, pos})
	s.Unlock()
	return nil
}

func (s *segment) writeOne(msg *Message) error {
	pos, err := s.writeLog(msg)
	if err != nil {
		return err
	}
	offset := uint32(msg.Id - s.minMsgid)
	if err = s.writeIndex(offset, pos); err != nil {
		return err
	}
	return nil
}

func (s *segment) seek(msgid uint64) (pos uint32, err error) {
	s.RLock()
	defer s.RUnlock()
	if len(s.indexs) == 0 {
		return
	}
	log.Debugf("seg seek:%#v\n", s.indexs[0])
	offset := uint32(msgid - s.minMsgid)
	index := sort.Search(len(s.indexs), func(i int) bool {
		return s.indexs[i].offset > offset
	})
	if index == 0 {
		log.Fatal(err)
		err = ErrNotFoundIndex
		return
	}
	pos = s.indexs[index-1].pos
	log.Debug("seg seek:", index-1, s.indexs[index-1].offset, s.indexs[index-1].pos)
	return
}

func (s *segment) readOne(msgid uint64, pos uint32) (msg *Message, nextPos uint32, err error) {
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
		log.Debug(id, len, msgid, pos)
		if id < msgid {
			pos += len
			continue
		} else if id == msgid {
			nextPos = pos + len
			data := s.logBuf[pos : pos+len : pos+len]
			msg = &Message{id, data}
			return
		} else {
			err = ErrNotFoundMsgid
			return
		}
	}
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
