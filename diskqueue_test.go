package diskqueue

import (
	"strconv"
	"testing"

	"github.com/tddhit/tools/log"
)

func TestDiskQueue(t *testing.T) {
	q := NewDiskQueue("test_topic")
	for i := 0; i < 100; i++ {
		str := strconv.Itoa(i)
		err := q.Put([]byte(str))
		if err != nil {
			log.Error(err)
			continue
		}
	}
	for i := 0; i < 100; i++ {
		data, err := q.Pop(uint64(i))
		if err != nil {
			log.Error(err)
			continue
		}
		log.Debug(string(data))
	}
}
