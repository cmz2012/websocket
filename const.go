package websocket

import "errors"

const (
	websocketGUID    = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
	DefaultFrameSize = 1024
)

var (
	ErrUnexpectedFrame = errors.New("ErrUnexpectedFrame")
	ErrInComplete      = errors.New("ErrInComplete")
	ErrMaskNotSet      = errors.New("ErrMaskNotSet")
	ErrNoNewFrame      = errors.New("ErrNoNewFrame")
)

const (
	PayloadTypeContinue byte = 0x0
	PayloadTypeText     byte = 0x1
	PayloadTypeBinary   byte = 0x2
	PayloadTypePing     byte = 0x9
	PayloadTypePong     byte = 0xa
)
