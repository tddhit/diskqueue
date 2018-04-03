package diskqueue

import (
	"strconv"
	"testing"
	"time"

	"github.com/tddhit/tools/log"
)

var q *DiskQueue

func startWrite() {
	for i := 0; i < 1000; i++ {
		str := "hello" + strconv.Itoa(i)
		err := q.Put([]byte(str))
		if err != nil {
			log.Fatal(err)
		}
		log.Debug("write:", i)
	}
}

func startRead(i int) {
	ch, err := q.StartRead(0)
	if err != nil {
		log.Fatal(err)
	}
	for msg := range ch {
		log.Infof("goroutine(%d):%d-%s\n", i, msg.Id, string(msg.Data))
	}
}

func TestDiskQueue(t *testing.T) {
	log.Init("", log.INFO)
	var err error
	q, err = NewDiskQueue("test_topic")
	if err != nil {
		log.Fatal(err)
	}
	go startWrite()
	for i := 0; i < 1; i++ {
		go startRead(i)
	}
	time.Sleep(10 * time.Second)
}
