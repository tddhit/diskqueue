package client

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

var byteSpace = []byte(" ")
var byteNewLine = []byte("\n")

type Command struct {
	Name   []byte
	Params [][]byte
	Body   []byte
}

func (c *Command) String() string {
	if len(c.Params) > 0 {
		return fmt.Sprintf("%s %s", c.Name, string(bytes.Join(c.Params, byteSpace)))
	}
	return string(c.Name)
}

func (c *Command) WriteTo(w io.Writer) (int64, error) {
	var total int64
	var buf [4]byte

	n, err := w.Write(c.Name)
	total += int64(n)
	if err != nil {
		return total, err
	}

	for _, param := range c.Params {
		n, err := w.Write(byteSpace)
		total += int64(n)
		if err != nil {
			return total, err
		}
		n, err = w.Write(param)
		total += int64(n)
		if err != nil {
			return total, err
		}
	}

	n, err = w.Write(byteNewLine)
	total += int64(n)
	if err != nil {
		return total, err
	}

	if c.Body != nil {
		bufs := buf[:]
		binary.BigEndian.PutUint32(bufs, uint32(len(c.Body)))
		n, err := w.Write(bufs)
		total += int64(n)
		if err != nil {
			return total, err
		}
		n, err = w.Write(c.Body)
		total += int64(n)
		if err != nil {
			return total, err
		}
	}

	return total, nil
}

func Publish(topic string, body []byte) *Command {
	var params = [][]byte{[]byte(topic)}
	return &Command{[]byte("PUB"), params, body}
}

func Subscribe(topic, channel, msgid string) *Command {
	var params = [][]byte{[]byte(topic), []byte(channel), []byte(msgid)}
	return &Command{[]byte("SUB"), params, nil}
}

func StartClose() *Command {
	return &Command{[]byte("CLS"), nil, nil}
}

func Nop() *Command {
	return &Command{[]byte("NOP"), nil, nil}
}

func Pull(topic, channel string) *Command {
	var params = [][]byte{[]byte(topic), []byte(channel)}
	return &Command{[]byte("PULL"), params, nil}
}
