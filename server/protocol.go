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

	ierrors "github.com/tddhit/diskqueue/errors"
	"github.com/tddhit/diskqueue/types"
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

type Command struct {
	Name   []byte
	Params [][]byte
	Body   []byte
}

type protocol struct {
}

func (p *protocol) IOLoop(ctx context.Context, conn net.Conn) error {
	var err error
	var line []byte

	dq := ctx.Value("diskqueue").(*DiskQueue)
	clientID := atomic.AddInt64(&dq.clientID, 1)
	client := newClient(clientID, conn)

	messagePumpStartedChan := make(chan bool)
	cmdChan := make(chan struct{})
	go p.heartbeatLoop(client)
	go p.messagePump(ctx, client, messagePumpStartedChan, cmdChan)
	<-messagePumpStartedChan

	for {
		if client.HeartbeatInterval > 0 {
			client.SetReadDeadline(time.Now().Add(client.HeartbeatInterval * 10))
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
		params := bytes.Split(line, separatorBytes)
		log.Infof("Command\tClient=%s\tParams=%s\n", client, params)
		if bytes.Equal(params[0], []byte("PULL")) {
			cmdChan <- struct{}{}
		} else {
			frameType, response, err := p.Exec(ctx, client, params)
			if err != nil {
				if err == ierrors.ErrAlreadyClose {
					break
				}
				sendErr := p.Send(client, frameTypeError, []byte(err.Error()))
				if sendErr != nil {
					log.Errorf("[%s] - %s%s", client, sendErr)
					break
				}
				continue
			}
			log.Debugf("Response\tType=%d\tClient=%s\tBody=%s\n", frameType, client, string(response))
			if response != nil {
				err = p.Send(client, frameType, response)
				if err != nil {
					err = fmt.Errorf("failed to send response - %s", err)
					break
				}
			}
		}
	}
	log.Infof("ExitIOLoop\tClient=%s\n", client)
	conn.Close()
	close(client.ExitChan) //通知messagePump退出
	if client.Channel != nil {
		client.Channel.RemoveClient()
	}
	return err
}

func (p *protocol) Send(client *client, frameType int32, data []byte) error {
	client.writeLock.Lock()
	if client.HeartbeatInterval > 0 {
		client.SetWriteDeadline(time.Now().Add(client.HeartbeatInterval))
	}
	// 写socket为什么需要加锁？
	_, err := SendFramedResponse(client.Writer, frameType, data)
	if err != nil {
		client.writeLock.Unlock()
		return err
	}
	//if frameType != frameTypeMessage {
	err = client.Flush()
	//}
	client.writeLock.Unlock()
	return err
}

func (p *protocol) parsePUB(client *client, params [][]byte) (*Command, error) {
	var err error

	if len(params) < 2 {
		log.Error("InvalidParams")
		return nil, ierrors.ErrInvalidParams
	}
	bodyLen, err := readLen(client.Reader, client.lenSlice)
	if err != nil || bodyLen <= 0 {
		return nil, ierrors.ErrBadMessage
	}
	messageBody := make([]byte, bodyLen)
	if _, err = io.ReadFull(client.Reader, messageBody); err != nil {
		return nil, ierrors.ErrBadMessage
	}
	log.Info("parsePUB", string(params[0]))
	return &Command{params[0], params, messageBody}, nil
}

func (p *protocol) Exec(
	ctx context.Context,
	client *client,
	params [][]byte) (int32, []byte, error) {

	switch {
	case bytes.Equal(params[0], []byte("PUB")):
		return p.PUB(ctx, client, params)
	case bytes.Equal(params[0], []byte("SUB")):
		return p.SUB(ctx, client, params)
	case bytes.Equal(params[0], []byte("NOP")):
		return p.NOP(ctx, client, params)
	case bytes.Equal(params[0], []byte("CLS")):
		return p.CLS(ctx, client, params)
	}
	return frameTypeError, nil, ierrors.ErrInvalidParams
}

func (p *protocol) heartbeatLoop(client *client) {
	heartbeatTicker := time.NewTicker(client.HeartbeatInterval)
	for {
		select {
		case <-heartbeatTicker.C:
			err := p.Send(client, frameTypeResponse, heartbeatBytes)
			if err != nil {
				goto exit
			}
		case <-client.ExitChan:
			goto exit
		}
	}
exit:
	heartbeatTicker.Stop()
}

func (p *protocol) SendMessage(client *client, msg *types.Message) error {
	var buf bytes.Buffer
	_, err := msg.WriteTo(&buf)
	if err != nil {
		return err
	}
	err = p.Send(client, frameTypeMessage, buf.Bytes())
	if err != nil {
		return err
	}
	log.Info("SendMessage:", msg.Id, string(msg.Data))
	return nil
}

func (p *protocol) messagePump(
	ctx context.Context,
	client *client,
	startedChan chan bool,
	cmdChan chan struct{}) {

	var err error
	close(startedChan)

	for {
		select {
		case <-cmdChan:
			var err error
			var msg *types.Message

			if atomic.LoadInt32(&client.State) != stateSubscribed {
				err = ierrors.ErrInvalidState
			} else {
				msg = client.Channel.GetMessage()
				if msg == nil {
					err = ierrors.ErrAlreadyClose
				}
			}
			if err != nil {
				sendErr := p.Send(client, frameTypeError, []byte(err.Error()))
				if sendErr != nil {
					log.Errorf("[%s] - %s", client, sendErr)
					goto exit
				}
				continue
			}
			if msg != nil {
				err = p.SendMessage(client, msg)
				if err != nil {
					err = fmt.Errorf("failed to send response - %s", err)
					goto exit
				}
			}
		case <-client.ExitChan:
			goto exit
		}
	}
exit:
	log.Infof("ExitMessagePump\tClient=%s\n", client)
	if err != nil {
		log.Errorf("PROTOCOL(V2): [%s] messagePump error - %s", client, err)
	}
}

func (p *protocol) SUB(
	ctx context.Context,
	client *client,
	params [][]byte) (int32, []byte, error) {

	if atomic.LoadInt32(&client.State) != stateInit {
		return frameTypeError, nil, errors.New("cannot SUB in current state")
	}
	if client.HeartbeatInterval <= 0 {
		return frameTypeError, nil, errors.New("cannot SUB with heartbeats disabled")
	}
	if len(params) < 3 {
		return frameTypeError, nil, errors.New("SUB insufficient number of parameters")
	}
	topicName := string(params[1])
	channelName := string(params[2])
	msgid := string(params[3])
	topic := ctx.Value("diskqueue").(*DiskQueue).GetTopic(topicName)
	channel, err := topic.GetChannel(channelName, msgid)
	if err != nil {
		return frameTypeError, nil, errors.New("not found msgid")
	}
	err = channel.AddClient(client)
	if err != nil {
		return frameTypeError, nil, errors.New("already subscribe")
	}
	atomic.StoreInt32(&client.State, stateSubscribed)
	client.Channel = channel
	return frameTypeResponse, []byte("SUB_OK-" + msgid), nil
}

func (p *protocol) CLS(ctx context.Context, client *client, params [][]byte) (int32, []byte, error) {
	if atomic.LoadInt32(&client.State) != stateSubscribed {
		return frameTypeError, nil, ierrors.ErrInvalidState
	}
	client.StartClose()
	return frameTypeResponse, []byte("CLOSE_WAIT"), nil
}

func (p *protocol) NOP(ctx context.Context, client *client, params [][]byte) (int32, []byte, error) {
	return frameTypeResponse, nil, nil
}

func (p *protocol) PUB(ctx context.Context, client *client, params [][]byte) (int32, []byte, error) {
	var err error

	if len(params) < 2 {
		log.Error("InvalidParams")
		return frameTypeError, nil, ierrors.ErrInvalidParams
	}
	topicName := string(params[1])
	bodyLen, err := readLen(client.Reader, client.lenSlice)
	if err != nil || bodyLen <= 0 {
		return frameTypeError, nil, ierrors.ErrBadMessage
	}
	messageBody := make([]byte, bodyLen)
	if _, err = io.ReadFull(client.Reader, messageBody); err != nil {
		return frameTypeError, nil, ierrors.ErrBadMessage
	}
	topic := ctx.Value("diskqueue").(*DiskQueue).GetTopic(topicName)
	if err = topic.PutMessage(messageBody); err != nil {
		return frameTypeError, nil, err
	}
	return frameTypeResponse, okBytes, nil
}

func readLen(r io.Reader, tmp []byte) (int32, error) {
	_, err := io.ReadFull(r, tmp)
	if err != nil {
		return 0, err
	}
	return int32(binary.BigEndian.Uint32(tmp)), nil
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

func (p *protocol) Close() {
}
