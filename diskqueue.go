package diskqueue

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tddhit/tools/log"
)

var (
	ErrEmptySegments   = errors.New("empty segments")
	ErrNotFoundSegment = errors.New("not found segment")
	ErrInvalidMsgid    = errors.New("invalid msgid")
	ErrMetaData        = errors.New("incorrect metadata")
	ErrAlreadyClose    = errors.New("diskqueue already close")
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

	needSync   bool
	exitFlag   bool
	exitChan   chan struct{}
	exitSyncWg sync.WaitGroup
}

func New(dataPath, topic string) (*DiskQueue, error) {
	q := &DiskQueue{
		name:     topic,
		dataPath: dataPath,

		writeChan:         make(chan *Message),
		writeResponseChan: make(chan error),

		exitChan: make(chan struct{}),
	}
	if err := q.loadMetaData(); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	}
	if len(q.segs) == 0 {
		if err := q.createSegment(0); err != nil {
			return nil, err
		}
	}
	q.exitSyncWg.Add(1)
	go q.writeLoop()
	return q, nil
}

func (q *DiskQueue) Put(data []byte) (err error) {
	q.RLock()
	defer q.RUnlock()

	if q.exitFlag {
		return ErrAlreadyClose
	}

	msgid := atomic.LoadUint64(&q.msgid)
	msg := &Message{msgid, data}
	q.writeChan <- msg
	return <-q.writeResponseChan
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
	log.Debugf("Seek\tmsgid=%d\tseg=%d\tpos=%d\n", msgid, seg.minMsgid, pos)
	return
}

func (q *DiskQueue) createSegment(msgid uint64) (err error) {
	seg, err := newSegment(q.dataPath, q.name, msgid, os.O_CREATE|os.O_RDWR|os.O_APPEND)
	if err != nil {
		return
	}
	q.segs = append(q.segs, seg)
	q.curSeg = seg
	return
}

func (q *DiskQueue) writeOne(msg *Message) error {
	if q.curSeg.full() {
		if err := q.createSegment(msg.Id); err != nil {
			return err
		}
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
			log.Debug("writeOne:", msg.Id)
			q.writeResponseChan <- q.writeOne(msg)
		case <-q.exitChan:
			goto exit
		}
	}
exit:
	log.Infof("diskqueue(%s) exit writeLoop.", q.name)
	q.exitSyncWg.Done()
}

func (q *DiskQueue) StartRead(msgid uint64) (<-chan *Message, error) {
	readChan := make(chan *Message)
	seg, pos, err := q.seek(msgid)
	if err != nil {
		return nil, err
	}
	q.exitSyncWg.Add(1)
	go q.readLoop(seg, pos, msgid, readChan)
	return readChan, nil
}

func (q *DiskQueue) readLoop(seg *segment, pos uint32, msgid uint64,
	readChan chan<- *Message) {

	var rchan chan<- *Message
	var msg *Message
	var err error

	for {
		curMsgid := atomic.LoadUint64(&q.msgid)
		log.Debugf("readLoop\tcurMsgid=%d\tmsgid=%d\tpos=%d\n", curMsgid, msgid, pos)
		if msgid < curMsgid {
			rchan = readChan
			msg, pos, err = seg.readOne(msgid, pos)
			if err != nil {
				log.Debug(err)
				os.Exit(1)
			}
			log.Debug("readOne:", msg.Id, string(msg.Data))
		} else {
			log.Debug("rchan = nil")
			rchan = nil
		}
		log.Debugf("readLoop\tcurMsgid=%d\tmsgid=%d\tpos=%d\n", curMsgid, msgid, pos)
		select {
		case rchan <- msg:
			msgid++
			continue
		case <-q.exitChan:
			close(readChan)
			goto exit
		}
		log.Debugf("readLoop\tcurMsgid=%d\tmsgid=%d\tpos=%d\n", curMsgid, msgid, pos)
		time.Sleep(100 * time.Millisecond)
	}
exit:
	log.Infof("diskqueue(%s) exit readLoop.", q.name)
	q.exitSyncWg.Done()
}

func (q *DiskQueue) persistMetaData() error {
	fileName := fmt.Sprintf(path.Join(q.dataPath, "%s.diskqueue.meta"), q.name)
	tmpFileName := fmt.Sprintf("%s.%d.tmp", fileName, rand.Int())
	f, err := os.OpenFile(tmpFileName, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0600)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	buf.WriteString(strconv.FormatUint(q.msgid, 10))
	buf.WriteString("\n")
	for _, seg := range q.segs {
		buf.WriteString(strconv.FormatUint(seg.minMsgid, 10))
		buf.WriteString(",")
		buf.WriteString(strconv.FormatUint(uint64(seg.size), 10))
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
	buf, err := ioutil.ReadFile(fileName)
	if err != nil {
		return err
	}
	lines := strings.Split(string(buf), "\n")
	lines = lines[:len(lines)-1]
	for i := 0; i < len(lines); i++ {
		if i == 0 {
			msgid, err := strconv.ParseUint(lines[i], 10, 64)
			if err != nil {
				return err
			}
			q.msgid = msgid
		} else {
			tokens := strings.Split(lines[i], ",")
			log.Debug(tokens)
			if len(tokens) != 4 {
				return ErrMetaData
			}
			msgid, err := strconv.ParseUint(tokens[0], 10, 64)
			if err != nil {
				return err
			}
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
			var flag int
			if i != len(lines)-1 {
				flag = os.O_RDONLY
			} else {
				flag = os.O_CREATE | os.O_RDWR | os.O_APPEND
			}
			seg, err := newSegment(q.dataPath, q.name, msgid, flag,
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

func (q *DiskQueue) Close() error {
	err := q.exit()
	if err != nil {
		return err
	}
	return q.sync()
}

func (q *DiskQueue) exit() error {
	q.Lock()
	defer q.Unlock()

	q.exitFlag = true

	close(q.exitChan)
	q.exitSyncWg.Wait()

	for _, seg := range q.segs {
		seg.exit()
	}
	return nil
}

func (q *DiskQueue) sync() error {
	for _, seg := range q.segs {
		err := seg.sync()
		if err != nil {
			return err
		}
	}
	err := q.persistMetaData()
	if err != nil {
		return err
	}
	q.needSync = false
	return nil
}
