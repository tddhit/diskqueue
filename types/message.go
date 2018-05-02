package types

import (
	"encoding/binary"
	"fmt"
	"io"
)

type Message struct {
	Id   uint64
	Data []byte
}

const (
	minValidMsgLength = 8
)

func DecodeMessage(b []byte) (*Message, error) {
	var msg Message

	if len(b) < minValidMsgLength {
		return nil, fmt.Errorf("invalid message buffer size (%d)", len(b))
	}

	msg.Id = binary.BigEndian.Uint64(b[:8])
	msg.Data = b[8:]

	return &msg, nil
}

func (m *Message) WriteTo(w io.Writer) (int64, error) {
	var buf [8]byte
	var total int64

	binary.BigEndian.PutUint64(buf[:], m.Id)

	n, err := w.Write(buf[:])
	total += int64(n)
	if err != nil {
		return total, err
	}

	n, err = w.Write(m.Data)
	total += int64(n)
	if err != nil {
		return total, err
	}

	return total, nil
}
