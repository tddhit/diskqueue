package core

import (
	"net/http"
	_ "net/http/pprof"
	"strconv"
	"sync"
	"testing"

	"github.com/tddhit/tools/log"
)

var topic *Topic

func startWrite(i int) {
	var j = 0
	for j = i * 10000; j < i*10000+10000; j++ {
		d := "hello" + strconv.Itoa(j)
		err := topic.PutMessage([]byte(d))
		if err != nil {
			log.Fatal(err)
		}
	}
	log.Info(j)
}

func startRead(i int) {
	name := "channel" + strconv.Itoa(i)
	channel, err := topic.GetChannel(name, "0")
	if err != nil {
		log.Fatal(err)
	}
	count := 0
	for {
		msg := channel.GetMessage()
		log.Info(msg.Id, string(msg.Data))
		count++
		if count == 10000 {
			break
		}
	}
}

func TestTopic(t *testing.T) {
	log.Init("topic.log", log.INFO)
	var (
		wg  sync.WaitGroup
		err error
	)
	topic, err = NewTopic("./data/", "test")
	if err != nil {
		log.Fatal(err)
	}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			startWrite(i)
			wg.Done()
		}(i)
	}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			startRead(i)
			wg.Done()
		}(i)
	}
	go func() {
		http.ListenAndServe(":6063", nil)
	}()
	wg.Wait()
	topic.Close()
}
