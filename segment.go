package diskqueue

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"sort"
	"syscall"
	"unsafe"

	"github.com/tddhit/tools/log"
)

const (
	MaxSegmentSize = 1 << 29 // 512M
	MaxMsgSize     = 1 << 20 // 1M
	MaxMapSize     = 1 << 30 //1G
)

type segment struct {
	size         uint32
	minOffset    uint64
	indexFile    *os.File
	logFile      *os.File
	writeBuf     bytes.Buffer
	indexEntries indexs
	logBuf       *[MaxMapSize]byte
	indexBuf     *[MaxMapSize]byte
}

func newSegment(name string, offset uint64, flag int) (*segment, error) {
	s := &segment{
		minOffset: offset,
	}
	fileName := fmt.Sprintf("%s.diskqueue.%d.log", name, offset)
	f, err := os.OpenFile(fileName, flag, 0600)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	s.logFile = f
	fileName = fmt.Sprintf("%s.diskqueue.%d.idx", name, offset)
	f, err = os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	s.indexFile = f
	s.mmap()
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
	for {
		var (
			off uint32
			pos uint32
		)
		buf := bytes.NewBuffer(s.indexBuf[pos : pos+4 : pos+4])
		err := binary.Read(buf, binary.BigEndian, &off)
		if err != nil {
			return err
		}
		pos += 4
		buf = bytes.NewBuffer(s.indexBuf[pos : pos+4 : pos+4])
		err = binary.Read(buf, binary.BigEndian, &pos)
		if err != nil {
			return err
		}
		s.indexEntries = append(s.indexEntries, &index{off, pos})
		pos += 4
	}
}

func (s *segment) full() bool {
	if s.size < MaxSegmentSize {
		return false
	}
	return true
}

func (s *segment) writeLog(offset uint64, data []byte) (pos int, err error) {
	dataLen := uint32(len(data))
	if dataLen > MaxMsgSize {
		return -1, fmt.Errorf("invalid message write size (%d) maxMsgSize=%d", dataLen, MaxMsgSize)
	}
	s.writeBuf.Reset()
	if err = binary.Write(&s.writeBuf, binary.BigEndian, offset); err != nil {
		return
	}
	if err = binary.Write(&s.writeBuf, binary.BigEndian, dataLen); err != nil {
		return
	}
	if _, err = s.writeBuf.Write(data); err != nil {
		return
	}
	if _, err = s.logFile.Write(s.writeBuf.Bytes()); err != nil {
		s.logFile.Close()
		s.logFile = nil
		return
	}
	return
}

func (s *segment) writeIndex(offset, pos uint32) (err error) {
	s.writeBuf.Reset()
	if err = binary.Write(&s.writeBuf, binary.BigEndian, offset); err != nil {
		return
	}
	if err = binary.Write(&s.writeBuf, binary.BigEndian, pos); err != nil {
		return
	}
	if _, err = s.indexFile.Write(s.writeBuf.Bytes()); err != nil {
		s.indexFile.Close()
		s.indexFile = nil
		return
	}
	s.indexEntries = append(s.indexEntries, &index{offset, pos})
	return
}

func (s *segment) writeOne(offset uint64, data []byte) error {
	pos, err := s.writeLog(offset, data)
	if err != nil {
		log.Error(err)
		return err
	}
	diff := int32(offset - s.minOffset)
	if diff < 0 {
		err = fmt.Errorf("offset(%d) < minOffset(%d)", offset, s.minOffset)
		return err
	}
	if err = s.writeIndex(uint32(diff), uint32(pos)); err != nil {
		log.Error(err)
		return err
	}
	return nil
}

func (s *segment) readOne(offset uint64) ([]byte, error) {
	if len(s.indexEntries) == 0 {
		return nil, errors.New("not found offset")
	}
	diff := int32(offset - s.minOffset)
	if diff < 0 {
		return nil, fmt.Errorf("offset(%d) < minOffset(%d)", offset, s.minOffset)
	}
	index := sort.Search(len(s.indexEntries), func(i int) bool {
		return s.indexEntries[i].offset >= uint32(diff)
	})
	if index == 0 && s.indexEntries[0].offset > uint32(diff) {
		return nil, errors.New("not found offset")
	} else {
		index = 1
	}
	pos := s.indexEntries[index-1].pos
	for {
		var (
			off     uint64
			dataLen uint32
		)
		buf := bytes.NewBuffer(s.logBuf[pos : pos+8 : pos+8])
		err := binary.Read(buf, binary.BigEndian, &off)
		if err != nil {
			return nil, err
		}
		pos += 8
		buf = bytes.NewBuffer(s.logBuf[pos : pos+4 : pos+4])
		err = binary.Read(buf, binary.BigEndian, &dataLen)
		if err != nil {
			return nil, err
		}
		pos += 4
		if off < offset {
			pos += dataLen
			continue
		} else if off == offset {
			msgBuf := s.logBuf[pos : pos+dataLen : pos+dataLen]
			return msgBuf, nil
		} else {
			return nil, fmt.Errorf("offset(%d) not found", offset)
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
func (s segments) Less(i, j int) bool { return s[i].minOffset < s[j].minOffset }
