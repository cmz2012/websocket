package websocket

import (
	"bytes"
	"encoding/binary"
	"errors"
	"github.com/sirupsen/logrus"
	"io"
	"math/rand"
	"net"
)

type Conn struct {
	r  FrameReader
	w  FrameWriter
	nc net.Conn

	pingHandle  func(msg []byte) error
	pongHandle  func(msg []byte) error
	closeHandle func(msg []byte) error
	frame       chan *Frame
}

// 从buf reader读取字节流生成frame，然后从frame的buffer读取
func (c Conn) Read(p []byte) (n int, err error) {
	for {
		if c.r.frame != nil && c.r.frame.Data != nil {
			// read from buffer first
			var cnt int
			cnt, err = c.r.frame.Data.Read(p[n:])
			n += cnt

			if n == len(p) || (err != nil && err != io.EOF) {
				return
			}
		}

		// read the next frame from buf
		err = c.r.ReadFrame()

		if err == ErrNoNewFrame {
			if n > 0 {
				return n, nil
			}
			return 0, io.EOF
		} else if err != nil {
			return n, nil
		}

		if isControlFrame(c.r.frame.OpCode) {
			c.handleControl()
			c.r.clear()
		}
	}
}

func (c Conn) handleControl() {
	switch c.r.frame.OpCode {
	case PayloadTypePing:
		logrus.Infof("[handleControl]: receive ping frame msg = %v", string(c.r.frame.Data.Bytes()))
		if c.pingHandle != nil {
			c.pingHandle(c.r.frame.Data.Bytes())
		} else {
			// default ping handler
			c.w.frame = &Frame{
				Fin:        true,
				Rsv:        [3]bool{},
				OpCode:     PayloadTypePong,
				Mask:       false,
				PayloadLen: uint64(c.r.frame.Data.Len()),
				MaskKey:    [4]byte{},
				Data:       c.r.frame.Data,
			}
			pongErr := c.w.WriteFrame()
			if pongErr != nil {
				logrus.Errorf("[handleControl]: write pong frame err = %v", pongErr)
			}
		}
	case PayloadTypePong:
		logrus.Infof("[handleControl]: receive pong frame msg = %v", string(c.r.frame.Data.Bytes()))
		if c.pongHandle != nil {
			c.pongHandle(c.r.frame.Data.Bytes())
		} else {
			// default pong handler
			logrus.Infof("[handleControl]: default pong handler is nil")
		}
	case PayloadTypeClose:
		code, msg := DecodeCloseMessage(c.r.frame.Data.Bytes())
		logrus.Infof("[handleControl]: receive close frame code = %v, msg = %v", code, msg)
		if c.closeHandle != nil {
			c.closeHandle(c.r.frame.Data.Bytes())
		} else {
			msg := "server receive close frame and send close frame"
			c.w.frame = &Frame{
				Fin:        true,
				Rsv:        [3]bool{},
				OpCode:     PayloadTypeClose,
				Mask:       false,
				PayloadLen: uint64(len(msg)),
				MaskKey:    [4]byte{},
				Data:       bytes.NewBuffer(FormatCloseMessage(CloseNormalClosure, msg)),
			}
			closeErr := c.w.WriteFrame()
			if closeErr != nil {
				logrus.Errorf("[handleControl]: write close frame err = %v", closeErr)
			}
		}
		c.Close()
	default:
		panic("unsupported payload type")
	}
}

func (c Conn) Write(p []byte) (n int, err error) {
	size := c.w.MaxFrameSize
	if size <= 0 {
		size = DefaultFrameSize
	}
	pSize := len(p)
	for n < pSize {
		// generate new frame, compute the frame size
		if pSize-n < size {
			size = pSize - n
		}
		frame := &Frame{
			Fin:        (n + size) >= pSize,
			Rsv:        [3]bool{},
			OpCode:     PayloadTypeText,
			Mask:       c.w.NeedMaskSet,
			PayloadLen: uint64(size),
			MaskKey:    [4]byte{},
			Data:       nil,
		}
		if frame.Mask {
			// generate random mask key
			binary.BigEndian.PutUint32(frame.MaskKey[:4], rand.Uint32())
		}
		bf := bytes.NewBuffer(nil)
		for i := 0; i < size; i++ {
			if frame.Mask {
				bf.WriteByte(p[n+i] ^ frame.MaskKey[i%4])
			} else {
				bf.WriteByte(p[n+i])
			}
		}
		frame.Data = bf
		c.w.frame = frame

		err = c.w.WriteFrame()
		if err != nil {
			return
		}
		n += size
	}
	return n, nil
}

func (c Conn) WriteControl(payloadType byte, msg string) (err error) {
	if !isControlFrame(payloadType) {
		return errors.New("not control frame")
	}
	c.w.frame = &Frame{
		Fin:        true,
		Rsv:        [3]bool{},
		OpCode:     payloadType,
		Mask:       false,
		PayloadLen: uint64(len(msg)),
		MaskKey:    [4]byte{},
		Data:       nil,
	}
	if payloadType == PayloadTypeClose {
		c.w.frame.Data = bytes.NewBuffer(FormatCloseMessage(CloseNormalClosure, msg))
	} else {
		c.w.frame.Data = bytes.NewBuffer([]byte(msg))
	}
	err = c.w.WriteFrame()
	return
}

func (c Conn) Close() error {
	return c.nc.Close()
}

func (c Conn) SetPingHandle(f func(msg []byte) error) {
	c.pingHandle = f
}

func (c Conn) SetPongHandle(f func(msg []byte) error) {
	c.pongHandle = f
}

func (c Conn) SetCloseHandle(f func(msg []byte) error) {
	c.closeHandle = f
}

func FormatCloseMessage(closeCode int, text string) []byte {
	if closeCode == CloseNoStatusReceived {
		// Return empty message because it's illegal to send
		// CloseNoStatusReceived. Return non-nil value in case application
		// checks for nil.
		return []byte{}
	}
	buf := make([]byte, 2+len(text))
	binary.BigEndian.PutUint16(buf, uint16(closeCode))
	copy(buf[2:], text)
	return buf
}

func DecodeCloseMessage(payload []byte) (code int, msg string) {
	code = CloseNoStatusReceived
	msg = ""
	if len(payload) >= 2 {
		code = int(binary.BigEndian.Uint16(payload))
		msg = string(payload[2:])
	}
	return
}
