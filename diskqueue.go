package diskqueue

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/tddhit/tools/log"
)

var (
	ErrEmptySegments   = errors.New("empty segments")
	ErrNotFoundSegment = errors.New("not found segment")
	ErrInvalidMsgid    = errors.New("invalid msgid")
	ErrMetaData        = errors.New("incorrect metadata")
)

type Message struct {
	Id   uint64
	Data []byte
}

type DiskQueue struct {
	sync.RWMutex

	name     string
	dataPath string
	msgid    uint64

	curSeg *segment
	segs   segments

	writeChan         chan *Message
	writeResponseChan chan error
	notifyChans       []chan struct{}
}

func NewDiskQueue(topic string) *DiskQueue {
	q := &DiskQueue{
		name: topic,

		writeChan:         make(chan *Message),
		writeResponseChan: make(chan error),
	}
	go q.writeLoop()
	return q
}

func (q *DiskQueue) Put(data []byte) (err error) {
	msgid := atomic.LoadUint64(&q.msgid)
	msg := &Message{msgid, data}
	q.writeChan <- msg
	err = <-q.writeResponseChan
	if err != nil {
		return
	}
	atomic.AddUint64(&q.msgid, 1)
	return
}

func (q *DiskQueue) seek(msgid uint64) (seg *segment, pos uint32, err error) {
	q.RLock()
	defer q.RUnlock()
	if len(q.segs) == 0 {
		err = ErrEmptySegments
		return
	}
	index := sort.Search(len(q.segs), func(i int) bool {
		return q.segs[i].minMsgid > msgid
	})
	if index == 0 {
		err = ErrNotFoundSegment
		return
	}
	seg = q.segs[index-1]
	pos, err = seg.seek(msgid)
	return
}

func (q *DiskQueue) writeOne(msg *Message) error {
	if q.curSeg == nil || q.curSeg.full() {
		seg, err := newSegment(q.name, msg.Id, os.O_CREATE|os.O_RDWR|os.O_APPEND)
		if err != nil {
			return err
		}
		q.Lock()
		q.segs = append(q.segs, seg)
		q.Unlock()
		q.curSeg = seg
	}
	if err := q.curSeg.writeOne(msg); err != nil {
		return err
	}
	atomic.AddUint64(&q.msgid, 1)
	return nil
}

func (q *DiskQueue) writeLoop() {
	for {
		select {
		case msg := <-q.writeChan:
			q.writeResponseChan <- q.writeOne(msg)
			q.RLock()
			for _, ch := range q.notifyChans {
				select {
				case ch <- struct{}{}:
				default:
				}
			}
			q.RUnlock()
		}
	}
}

func (q *DiskQueue) StartRead(msgid uint64) (<-chan *Message, error) {
	readChan := make(chan *Message)
	notifyChan := make(chan struct{})
	q.Lock()
	q.notifyChans = append(q.notifyChans, notifyChan)
	q.Unlock()
	seg, pos, err := q.seek(msgid)
	if err != nil {
		return nil, ErrInvalidMsgid
	}
	go q.readLoop(seg, pos, msgid, readChan, notifyChan)
	return readChan, nil
}

func (q *DiskQueue) readLoop(seg *segment, pos uint32, msgid uint64,
	readChan chan<- *Message, notifyChan <-chan struct{}) {

	var rchan chan<- *Message
	var nchan <-chan struct{}
	var msg *Message
	var err error

	for {
		curMsgid := atomic.LoadUint64(&q.msgid)
		if msgid < curMsgid {
			rchan = readChan
			nchan = nil
			msg, pos, err = seg.readOne(msgid, pos)
			if err != nil {
				log.Error(err)
				os.Exit(1)
			}
		} else {
			rchan = nil
			nchan = notifyChan
		}
		select {
		case rchan <- msg:
			msgid++
		case <-nchan:
		}
	}
}

func (q *DiskQueue) persistMetaData() error {
	fileName := fmt.Sprintf(path.Join(q.dataPath, "%s.diskqueue.meta"), q.name)
	tmpFileName := fmt.Sprintf("%s.%d.tmp", fileName, rand.Int())
	f, err := os.OpenFile(tmpFileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	buf.WriteString(strconv.FormatUint(q.msgid, 10))
	buf.WriteString("\n")
	for _, seg := range q.segs {
		buf.WriteString(strconv.FormatUint(seg.minMsgid, 10))
		buf.WriteString(",")
		buf.WriteString(strconv.FormatUint(uint64(seg.pos), 10))
		buf.WriteString(",")
		buf.WriteString(strconv.FormatUint(uint64(seg.indexCount), 10))
		buf.WriteString(",")
		buf.WriteString(strconv.FormatUint(uint64(seg.msgCount), 10))
		buf.WriteString("\n")
	}
	_, err = f.Write(buf.Bytes())
	if err != nil {
		return err
	}
	f.Sync()
	f.Close()
	return os.Rename(tmpFileName, fileName)
}

func (q *DiskQueue) loadMetaData() error {
	fileName := fmt.Sprintf(path.Join(q.dataPath, "%s.diskqueue.meta"), q.name)
	f, err := os.OpenFile(fileName, os.O_RDONLY, 0600)
	if err != nil {
		return err
	}
	firstLine := true
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if firstLine {
			msgid, err := strconv.ParseUint(line, 10, 64)
			if err != nil {
				return err
			}
			q.msgid = msgid
		} else {
			tokens := strings.Split(line, ",")
			if len(tokens) != 4 {
				return ErrMetaData
			}
			msgid, _ := strconv.ParseUint(tokens[0], 10, 64)
			t, err := strconv.ParseUint(tokens[1], 10, 32)
			if err != nil {
				return err
			}
			pos := uint32(t)
			t, err = strconv.ParseUint(tokens[2], 10, 32)
			if err != nil {
				return err
			}
			indexCount := uint32(t)
			t, err = strconv.ParseUint(tokens[3], 10, 32)
			if err != nil {
				return err
			}
			msgCount := uint32(t)
			seg, err := newSegment(q.name, msgid, os.O_RDONLY,
				pos, indexCount, msgCount)
			if err != nil {
				return err
			}
			q.segs = append(q.segs, seg)
		}
	}
	if len(q.segs) > 0 {
		q.curSeg = q.segs[len(q.segs)-1]
	}
	return nil
}
