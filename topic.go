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

type Topic struct {
	sync.RWMutex

	name     string
	dataPath string
	msgid    uint64

	channelMap sync.Map

	curSeg *segment
	segs   segments

	writeChan         chan *Message
	writeResponseChan chan error

	needSync   bool
	exitFlag   bool
	exitChan   chan struct{}
	exitSyncWg sync.WaitGroup
}

func NewTopic(dataPath, topic string) (*Topic, error) {
	t := &Topic{
		name:     topic,
		dataPath: dataPath,

		writeChan:         make(chan *Message),
		writeResponseChan: make(chan error),

		exitChan: make(chan struct{}),
	}
	if err := t.loadMetaData(); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
	}
	if len(t.segs) == 0 {
		if err := t.createSegment(0); err != nil {
			return nil, err
		}
	}
	t.exitSyncWg.Add(1)
	go t.writeLoop()
	return t, nil
}

func (t *Topic) GetChannel(channelName, msgid string) *Channel {
	if c, ok := t.channelMap.Load(channelName); ok {
		return c.(*Channel)
	}

	t.Lock()
	if c, ok := t.channelMap.Load(channelName); ok {
		t.Unlock()
		return c.(*Channel)
	}
	id, _ := strconv.ParseUint(msgid, 10, 64)
	readChan, _ := t.StartRead(id)
	channel := NewChannel(t.name, channelName, readChan)
	t.channelMap.Store(channelName, channel)
	t.Unlock()

	log.Debugf("CreateChannel\tTopic=%s\tChannel=%s\n", t.name, channel.name)
	return channel
}

func (t *Topic) PutMessage(data []byte) (err error) {
	t.RLock()
	defer t.RUnlock()

	if t.exitFlag {
		return ErrAlreadyClose
	}

	msg := &Message{Data: data}
	t.writeChan <- msg
	return <-t.writeResponseChan
}

func (t *Topic) seek(msgid uint64) (seg *segment, pos uint32, err error) {
	t.RLock()
	defer t.RUnlock()
	if len(t.segs) == 0 {
		err = ErrEmptySegments
		return
	}
	index := sort.Search(len(t.segs), func(i int) bool {
		return t.segs[i].minMsgid > msgid
	})
	if index == 0 {
		err = ErrNotFoundSegment
		return
	}
	seg = t.segs[index-1]
	pos, err = seg.seek(msgid)
	log.Debugf("Seek\tmsgid=%d\tseg=%d\tpos=%d\n", msgid, seg.minMsgid, pos)
	return
}

func (t *Topic) createSegment(msgid uint64) (err error) {
	seg, err := newSegment(t.dataPath, t.name, msgid, os.O_CREATE|os.O_RDWR|os.O_APPEND)
	if err != nil {
		return
	}
	t.segs = append(t.segs, seg)
	t.curSeg = seg
	return
}

func (t *Topic) writeOne(msg *Message) error {
	if t.curSeg.full() {
		if err := t.createSegment(msg.Id); err != nil {
			return err
		}
	}
	if err := t.curSeg.writeOne(msg); err != nil {
		return err
	}
	atomic.AddUint64(&t.msgid, 1)
	return nil
}

func (t *Topic) writeLoop() {
	for {
		select {
		case msg := <-t.writeChan:
			msg.Id = atomic.LoadUint64(&t.msgid)
			log.Debugf("writeOne\tMsgid=%d", msg.Id)
			t.writeResponseChan <- t.writeOne(msg)
		case <-t.exitChan:
			goto exit
		}
	}
exit:
	log.Infof("diskqueue(%s) exit writeLoop.", t.name)
	t.exitSyncWg.Done()
}

func (t *Topic) StartRead(msgid uint64) (<-chan *Message, error) {
	readChan := make(chan *Message)
	seg, pos, err := t.seek(msgid)
	if err != nil {
		return nil, err
	}
	t.exitSyncWg.Add(1)
	go t.readLoop(seg, pos, msgid, readChan)
	return readChan, nil
}

func (t *Topic) readLoop(seg *segment, pos uint32, msgid uint64,
	readChan chan<- *Message) {

	var rchan chan<- *Message
	var msg *Message
	var err error

	for {
		curMsgid := atomic.LoadUint64(&t.msgid)
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
		case <-t.exitChan:
			close(readChan)
			goto exit
		}
		log.Debugf("readLoop\tcurMsgid=%d\tmsgid=%d\tpos=%d\n", curMsgid, msgid, pos)
		time.Sleep(100 * time.Millisecond)
	}
exit:
	log.Infof("diskqueue(%s) exit readLoop.", t.name)
	t.exitSyncWg.Done()
}

func (t *Topic) persistMetaData() error {
	fileName := fmt.Sprintf(path.Join(t.dataPath, "%s.diskqueue.meta"), t.name)
	tmpFileName := fmt.Sprintf("%s.%d.tmp", fileName, rand.Int())
	f, err := os.OpenFile(tmpFileName, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0600)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	buf.WriteString(strconv.FormatUint(t.msgid, 10))
	buf.WriteString("\n")
	for _, seg := range t.segs {
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

func (t *Topic) loadMetaData() error {
	fileName := fmt.Sprintf(path.Join(t.dataPath, "%s.diskqueue.meta"), t.name)
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
			t.msgid = msgid
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
			r, err := strconv.ParseUint(tokens[1], 10, 32)
			if err != nil {
				return err
			}
			pos := uint32(r)
			r, err = strconv.ParseUint(tokens[2], 10, 32)
			if err != nil {
				return err
			}
			indexCount := uint32(r)
			r, err = strconv.ParseUint(tokens[3], 10, 32)
			if err != nil {
				return err
			}
			msgCount := uint32(r)
			var flag int
			if i != len(lines)-1 {
				flag = os.O_RDONLY
			} else {
				flag = os.O_CREATE | os.O_RDWR | os.O_APPEND
			}
			seg, err := newSegment(t.dataPath, t.name, msgid, flag,
				pos, indexCount, msgCount)
			if err != nil {
				return err
			}
			t.segs = append(t.segs, seg)
		}
	}
	if len(t.segs) > 0 {
		t.curSeg = t.segs[len(t.segs)-1]
	}
	return nil
}

func (t *Topic) Close() error {
	err := t.exit()
	if err != nil {
		return err
	}
	return t.sync()
}

func (t *Topic) exit() error {
	t.Lock()
	defer t.Unlock()

	t.exitFlag = true

	close(t.exitChan)
	t.exitSyncWg.Wait()

	for _, seg := range t.segs {
		seg.exit()
	}

	t.channelMap.Range(func(key, value interface{}) bool {
		channel := value.(*Channel)
		channel.Close()
		return true
	})
	return nil
}

func (t *Topic) sync() error {
	for _, seg := range t.segs {
		err := seg.sync()
		if err != nil {
			return err
		}
	}
	err := t.persistMetaData()
	if err != nil {
		return err
	}
	t.needSync = false
	return nil
}
