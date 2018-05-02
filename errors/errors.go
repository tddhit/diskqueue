package errors

import "errors"

var (
	ErrAlreadyClose  = errors.New("topic already close")
	ErrInvalidParams = errors.New("invalid params")
	ErrBadMessage    = errors.New("bad message")
	ErrInvalidState  = errors.New("invalid state")
)
