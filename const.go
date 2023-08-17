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
	FragmentBegin  int = 0
	FragmentMiddle int = 1
	FragmentEnd    int = 2
)

const (
	PayloadTypeContinue byte = 0x0
	PayloadTypeText     byte = 0x1
	PayloadTypeBinary   byte = 0x2
	PayloadTypeClose    byte = 0x8
	PayloadTypePing     byte = 0x9
	PayloadTypePong     byte = 0xa
)

const (
	CloseNormalClosure           = 1000
	CloseGoingAway               = 1001
	CloseProtocolError           = 1002
	CloseUnsupportedData         = 1003
	CloseNoStatusReceived        = 1005
	CloseAbnormalClosure         = 1006
	CloseInvalidFramePayloadData = 1007
	ClosePolicyViolation         = 1008
	CloseMessageTooBig           = 1009
	CloseMandatoryExtension      = 1010
	CloseInternalServerErr       = 1011
	CloseServiceRestart          = 1012
	CloseTryAgainLater           = 1013
	CloseTLSHandshake            = 1015
)
