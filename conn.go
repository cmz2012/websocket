package websocket

import (
	"bytes"
	"encoding/binary"
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
			switch c.r.frame.OpCode {
			case PayloadTypePing:
				logrus.Infof("[ReadFrame]: receive ping frame msg = %v", string(c.r.frame.Data.Bytes()))
				if c.pingHandle != nil {
					c.pingHandle(c.r.frame.Data.Bytes())
				} else {
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
						logrus.Errorf("[ReadFrame]: write pong frame err = %v", pongErr)
					}
				}
			case PayloadTypePong:
				logrus.Infof("[ReadFrame]: receive pong frame msg = %v", string(c.r.frame.Data.Bytes()))
				if c.pongHandle != nil {
					c.pongHandle(c.r.frame.Data.Bytes())
				} else {

				}
			case PayloadTypeClose:
				logrus.Infof("[ReadFrame]: receive close frame msg = %v", string(c.r.frame.Data.Bytes()))
				if c.closeHandle != nil {
					c.closeHandle(c.r.frame.Data.Bytes())
				} else {
					msg := []byte("server receive close frame and send close frame")
					c.w.frame = &Frame{
						Fin:        true,
						Rsv:        [3]bool{},
						OpCode:     PayloadTypeClose,
						Mask:       false,
						PayloadLen: uint64(len(msg)),
						MaskKey:    [4]byte{},
						Data:       bytes.NewBuffer(msg),
					}
					closeErr := c.w.WriteFrame()
					if closeErr != nil {
						logrus.Errorf("[ReadFrame]: write close frame err = %v", closeErr)
					}
					c.Close()
					return n, nil
				}
			default:
				panic("unsupported payload type")
			}
		}
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
