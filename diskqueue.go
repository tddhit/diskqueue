package diskqueue

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/tddhit/tools/dirlock"
	"github.com/tddhit/tools/log"
)

type DiskQueue struct {
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
}

func (q *DiskQueue) GetTopic(topicName string) *Topic {
	if t, ok := q.topicMap.Load(topicName); ok {
		return t.(*Topic)
	}
	topic, _ := NewTopic(q.getOpts().DataPath, topicName)
	q.topicMap.Store(topicName, topic)
	log.Debugf("CreateTopic\tTopic=%s\n", topicName)
	return topic
}

func (q *DiskQueue) Wait() {
	q.wg.Wait()
}

func (q *DiskQueue) Exit() {
}
