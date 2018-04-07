package diskqueue

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync/atomic"
	"time"

	"github.com/tddhit/tools/log"
)

const (
	frameTypeResponse int32 = 0
	frameTypeError    int32 = 1
	frameTypeMessage  int32 = 2
)

var (
	separatorBytes = []byte(" ")
	heartbeatBytes = []byte("_heartbeat_")
	okBytes        = []byte("OK")
)

var (
	ErrInvalidParams = errors.New("invalid params")
	ErrBadMessage    = errors.New("bad message")
	ErrInvalidState  = errors.New("invalid state")
)

type protocol struct {
}

func (p *protocol) IOLoop(ctx context.Context, conn net.Conn) error {
	var err error
	var line []byte
	var zeroTime time.Time

	//clientID := atomic.AddInt64(&p.ctx.nsqd.clientIDSequence, 1)
	var clientID int64 = 1
	client := newClientV2(clientID, conn)

	messagePumpStartedChan := make(chan bool)
	go p.messagePump(client, messagePumpStartedChan)
	<-messagePumpStartedChan

	for {
		if client.HeartbeatInterval > 0 {
			client.SetReadDeadline(time.Now().Add(client.HeartbeatInterval * 2))
		} else {
			client.SetReadDeadline(zeroTime)
		}
		line, err = client.Reader.ReadSlice('\n')
		if err != nil {
			log.Errorf("ReadError\tClient=%s\tErr=%s\n", client, err)
			if err == io.EOF {
				err = nil
			} else {
				err = fmt.Errorf("failed to read command - %s", err)
			}
			break
		}
		line = line[:len(line)-1]
		if len(line) > 0 && line[len(line)-1] == '\r' {
			line = line[:len(line)-1]
		}
		params := bytes.Split(line, separatorBytes)
		log.Debugf("Command\tClient=%s\tParams=%s\n", client, params)
		response, err := p.Exec(ctx, client, params)
		if err != nil {
			sendErr := p.Send(client, frameTypeError, []byte(err.Error()))
			if sendErr != nil {
				log.Errorf("[%s] - %s%s", client, sendErr)
				break
			}
			continue
		}
		log.Debugf("Response\tClient=%s\tBody=%s\n", client, string(response))
		if response != nil {
			err = p.Send(client, frameTypeResponse, response)
			if err != nil {
				err = fmt.Errorf("failed to send response - %s", err)
				break
			}
		}
	}
	log.Infof("ExitIOLoop\tClient=%s\n", client)
	conn.Close()
	close(client.ExitChan) //通知messagePump退出
	if client.Channel != nil {
		client.Channel.RemoveClient(client.ID)
	}
	return err
}

func (p *protocol) Send(client *clientV2, frameType int32, data []byte) error {
	client.writeLock.Lock()
	var zeroTime time.Time
	if client.HeartbeatInterval > 0 {
		client.SetWriteDeadline(time.Now().Add(client.HeartbeatInterval))
	} else {
		client.SetWriteDeadline(zeroTime)
	}
	// 写socket为什么需要加锁？
	_, err := SendFramedResponse(client.Writer, frameType, data)
	if err != nil {
		client.writeLock.Unlock()
		return err
	}
	if frameType != frameTypeMessage {
		err = client.Flush()
	}
	client.writeLock.Unlock()
	return err
}

func (p *protocol) Exec(ctx context.Context, client *clientV2, params [][]byte) ([]byte, error) {
	switch {
	case bytes.Equal(params[0], []byte("PUB")):
		return p.PUB(ctx, client, params)
	case bytes.Equal(params[0], []byte("SUB")):
		return p.SUB(ctx, client, params)
	case bytes.Equal(params[0], []byte("POLL")):
		return p.POLL(ctx, client, params)
	case bytes.Equal(params[0], []byte("NOP")):
		return p.NOP(ctx, client, params)
	case bytes.Equal(params[0], []byte("CLS")):
		return p.CLS(ctx, client, params)
	}
	return nil, errors.New("invalid params")
}

func (p *protocol) messagePump(client *clientV2, startedChan chan bool) {
	var err error

	heartbeatTicker := time.NewTicker(client.HeartbeatInterval)
	heartbeatChan := heartbeatTicker.C

	close(startedChan)
	for {
		select {
		case <-heartbeatChan:
			err = p.Send(client, frameTypeResponse, heartbeatBytes)
			if err != nil {
				goto exit
			}
		case <-client.ExitChan:
			goto exit
		}
	}
exit:
	log.Infof("ExitMessagePump\tClient=%s\n", client)
	heartbeatTicker.Stop()
	if err != nil {
		log.Errorf("PROTOCOL(V2): [%s] messagePump error - %s", client, err)
	}
}

func (p *protocol) SUB(ctx context.Context, client *clientV2, params [][]byte) ([]byte, error) {
	if atomic.LoadInt32(&client.State) != stateInit {
		return nil, errors.New("cannot SUB in current state")
	}
	if client.HeartbeatInterval <= 0 {
		return nil, errors.New("cannot SUB with heartbeats disabled")
	}
	if len(params) < 3 {
		return nil, errors.New("SUB insufficient number of parameters")
	}
	topicName := string(params[1])
	channelName := string(params[2])
	topic := ctx.Value("diskqueue").(*DiskQueue).GetTopic(topicName)
	channel := topic.GetChannel(channelName)
	channel.AddClient(client.ID, client)
	atomic.StoreInt32(&client.State, stateSubscribed)
	client.Channel = channel
	client.SubEventChan <- channel
	return okBytes, nil
}

func (p *protocol) CLS(ctx context.Context, client *clientV2, params [][]byte) ([]byte, error) {
	if atomic.LoadInt32(&client.State) != stateSubscribed {
		return nil, ErrInvalidState
	}
	client.StartClose()
	return []byte("CLOSE_WAIT"), nil
}

func (p *protocol) NOP(ctx context.Context, client *clientV2, params [][]byte) ([]byte, error) {
	return nil, nil
}

func (p *protocol) POLL(ctx context.Context, client *clientV2, params [][]byte) ([]byte, error) {
	if atomic.LoadInt32(&client.State) != stateInit {
		return nil, ErrInvalidState
	}
	if len(params) < 3 {
		return nil, ErrInvalidParams
	}
	topicName := string(params[1])
	channelName := string(params[2])
	topic := ctx.Value("diskqueue").(*DiskQueue).GetTopic(topicName)
	channel := topic.GetChannel(channelName)
	msg := channel.Get()
	return msg.Data, nil
}

func (p *protocol) PUB(ctx context.Context, client *clientV2, params [][]byte) ([]byte, error) {
	var err error

	if len(params) < 2 {
		return nil, ErrInvalidParams
	}
	topicName := string(params[1])
	bodyLen, err := readLen(client.Reader, client.lenSlice)
	if err != nil || bodyLen <= 0 {
		return nil, ErrBadMessage
	}
	messageBody := make([]byte, bodyLen)
	if _, err = io.ReadFull(client.Reader, messageBody); err != nil {
		return nil, ErrBadMessage
	}
	topic := ctx.Value("diskqueue").(*DiskQueue).GetTopic(topicName)
	if err = topic.Put(messageBody); err != nil {
		return nil, err
	}
	return okBytes, nil
}

func readLen(r io.Reader, tmp []byte) (int32, error) {
	_, err := io.ReadFull(r, tmp)
	if err != nil {
		return 0, err
	}
	return int32(binary.BigEndian.Uint32(tmp)), nil
}

func SendResponse(w io.Writer, data []byte) (int, error) {
	err := binary.Write(w, binary.BigEndian, int32(len(data)))
	if err != nil {
		return 0, err
	}

	n, err := w.Write(data)
	if err != nil {
		return 0, err
	}

	return (n + 4), nil
}

func SendFramedResponse(w io.Writer, frameType int32, data []byte) (int, error) {
	beBuf := make([]byte, 4)
	size := uint32(len(data)) + 4

	binary.BigEndian.PutUint32(beBuf, size)
	n, err := w.Write(beBuf)
	if err != nil {
		return n, err
	}

	binary.BigEndian.PutUint32(beBuf, uint32(frameType))
	n, err = w.Write(beBuf)
	if err != nil {
		return n + 4, err
	}

	n, err = w.Write(data)
	return n + 8, err
}
