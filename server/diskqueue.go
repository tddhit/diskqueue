package diskqueue

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"

	. "github.com/tddhit/diskqueue/core"
	"github.com/tddhit/tools/dirlock"
	"github.com/tddhit/tools/log"
)

type DiskQueue struct {
	sync.Mutex

	tcpServer *TCPServer
	topicMap  sync.Map

	opts     atomic.Value
	clientID int64
	dl       *dirlock.DirLock
	wg       sync.WaitGroup
}

func New(opts *Options) *DiskQueue {
	q := &DiskQueue{
		tcpServer: NewTCPServer(opts.TCPAddress),
		dl:        dirlock.New(opts.DataPath),
	}
	q.opts.Store(opts)
	if err := q.dl.Lock(); err != nil {
		log.Fatal(err)
	}
	log.Infof("ID=%d\tAddress=%s\tDataPath=%s", opts.ID, opts.TCPAddress, opts.DataPath)
	return q
}

func (q *DiskQueue) getOpts() *Options {
	return q.opts.Load().(*Options)
}

func (q *DiskQueue) Go() {
	q.wg.Add(1)
	go func() {
		ctx := context.WithValue(context.Background(), "diskqueue", q)
		q.tcpServer.ListenAndServe(ctx)
		q.wg.Done()
	}()
	signalChan := make(chan os.Signal)
	signal.Notify(signalChan)
	<-signalChan
	q.Exit()
}

func (q *DiskQueue) GetTopic(topicName string) *Topic {
	if t, ok := q.topicMap.Load(topicName); ok {
		return t.(*Topic)
	}

	q.Lock()
	if t, ok := q.topicMap.Load(topicName); ok {
		q.Unlock()
		return t.(*Topic)
	}
	topic, _ := NewTopic(q.getOpts().DataPath, topicName)
	q.topicMap.Store(topicName, topic)
	q.Unlock()

	log.Debugf("CreateTopic\tTopic=%s\n", topicName)
	return topic
}

func (q *DiskQueue) Exit() {
	q.tcpServer.Close()
	log.Error("Wait")
	q.tcpServer.wg.Wait()
	log.Error("Exit")
	q.topicMap.Range(func(key, value interface{}) bool {
		topic := value.(*Topic)
		topic.Close()
		return true
	})
}
