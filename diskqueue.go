package diskqueue

import ()

type DiskQueue struct {
	tcpServer *TCPServer
	topicMap  sync.Map

	opts atomic.Value
	dl   *dirlock.DirLock
	wg   sync.WaitGroup
}

func New(opts *Options) *DiskQueue {
	q := &DiskQueue{
		tcpServer: NewTCPServer(opts.TCPAddress),
		dl:        dirlock.New(opts.DataPath),
	}
	q.opts.Store(opts)
	if err := q.dl.Lock(); err != nil {
		log.Fatalf("%s in use", opts.DataPath)
	}
	log.Infof("ID: %d", opts.ID)
	return q
}

func (q *DiskQueue) getOpts() *Options {
	return q.opts.Load().(*Options)
}

func (q *DiskQueue) Go() {
	q.wg.Add(1)
	go func() {
		q.tcpServer.ListenAndServe()
		q.wg.Done()
	}()
}

func (q *DiskQueue) GetTopic(topicName string) *Topic {
	if t, ok := q.topicMap.Load(topicName); ok {
		return t.(*Topic)
	}
	topic := NewTopic(q.getOpts().DataPath, topicName)
	n.topicMap.Store(topicName, topic)
	log.Debugf("CreateTopic\tTopic=%s\n", topicName)
	return topic
}
