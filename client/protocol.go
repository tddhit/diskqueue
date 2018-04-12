package client

import (
	"encoding/binary"
	"errors"
	"io"
)

const (
	FrameTypeResponse int32 = 0
	FrameTypeError    int32 = 1
	FrameTypeMessage  int32 = 2
)

// ReadResponse is a client-side utility function to read from the supplied Reader
// according to the NSQ protocol spec:
//
//    [x][x][x][x][x][x][x][x]...
//    |  (int32) || (binary)
//    |  4-byte  || N-byte
//    ------------------------...
//        size       data
func ReadResponse(r io.Reader) ([]byte, error) {
	var msgSize int32
	err := binary.Read(r, binary.BigEndian, &msgSize)
	if err != nil {
		return nil, err
	}
	buf := make([]byte, msgSize)
	_, err = io.ReadFull(r, buf)
	if err != nil {
		return nil, err
	}
	return buf, nil
}

// UnpackResponse is a client-side utility function that unpacks serialized data
// according to NSQ protocol spec:
//
//    [x][x][x][x][x][x][x][x]...
//    |  (int32) || (binary)
//    |  4-byte  || N-byte
//    ------------------------...
//      frame ID     data
//
// Returns a triplicate of: frame type, data ([]byte), error
func UnpackResponse(response []byte) (int32, []byte, error) {
	if len(response) < 4 {
		return -1, nil, errors.New("length of response is too small")
	}
	return int32(binary.BigEndian.Uint32(response)), response[4:], nil
}

func ReadUnpackedResponse(r io.Reader) (int32, []byte, error) {
	resp, err := ReadResponse(r)
	if err != nil {
		return -1, nil, err
	}
	return UnpackResponse(resp)
}
