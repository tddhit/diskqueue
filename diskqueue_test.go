package diskqueue

import (
	"sync"
	"testing"
	"time"

	"github.com/tddhit/tools/log"
)

var q *DiskQueue

func startWrite() {
	//for i := 0; i < 100000; i++ {
	//	str := "hello" + strconv.Itoa(i)
	//	err := q.Put([]byte(str))
	//	if err != nil {
	//		log.Fatal(err)
	//	}
	//	log.Debug("write:", i)
	//}
	log.Debug("End Write!")
	time.Sleep(2000 * time.Second)
	q.Close()
}

func startRead(i int) {
	ch, err := q.StartRead(0)
	if err != nil {
		log.Fatal(err)
	}
	for msg := range ch {
		//log.Debugf("%d-%s\n", msg.Id, string(msg.Data))
		log.Infof("gid(%d):%d-%s", i, msg.Id, string(msg.Data))
	}
	log.Debug("End Read!")
}

func TestDiskQueue(t *testing.T) {
	log.Init("", log.INFO)
	var wg sync.WaitGroup
	var err error
	q, err = NewDiskQueue("data", "topic")
	if err != nil {
		log.Fatal(err)
	}
	wg.Add(1)
	go func() {
		startWrite()
		wg.Done()
	}()
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func(i int) {
			startRead(i)
			wg.Done()
		}(i)
	}
	wg.Wait()
}
