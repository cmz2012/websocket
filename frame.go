package websocket

import (
	"bufio"
	"encoding/binary"
	"github.com/sirupsen/logrus"
	"github.com/valyala/bytebufferpool"
	"io"
	"math"
	"math/rand"
)

type Frame struct {
	Fin        bool
	Rsv        [3]bool
	OpCode     byte
	Mask       bool
	PayloadLen uint64
	MaskKey    [4]byte
	Data       *bytebufferpool.ByteBuffer
}

type FrameReader struct {
	frame       *Frame
	buf         *bufio.Reader
	NeedMaskSet bool
}

// ReadFrame replace the frame with the new coming frame
func (fr *FrameReader) ReadFrame() (err error) {
	frame := Frame{}

	b, err := fr.buf.ReadByte()
	// if EOF means no new frame
	if err == io.EOF {
		err = ErrNoNewFrame
	}
	if err != nil {
		return
	}
	frame.Fin = (b >> 7) > 0
	frame.OpCode = (b << 4) >> 4

	b, err = fr.buf.ReadByte()
	if err != nil {
		return
	}
	frame.Mask = (b >> 7) > 0
	if fr.NeedMaskSet && !frame.Mask {
		return ErrMaskNotSet
	}

	payLen := (b << 1) >> 1
	realLen := uint64(0)
	if payLen <= 125 {
		realLen = uint64(payLen)
	} else if payLen == 126 {
		bs := make([]byte, 2)
		_, err = fr.buf.Read(bs)
		if err != nil {
			return
		}
		realLen = uint64(binary.BigEndian.Uint16(bs))
	} else {
		bs := make([]byte, 8)
		_, err = fr.buf.Read(bs)
		if err != nil {
			return
		}
		realLen = binary.BigEndian.Uint64(bs)
	}
	frame.PayloadLen = realLen

	if frame.Mask {
		_, err = fr.buf.Read(frame.MaskKey[:])
		if err != nil {
			return
		}
	}

	fr.RefreshFrame(&frame)

	// decode byte read from buf
	err = fr.Decode()
	if err != nil {
		return
	}
	logrus.Infof("[ReadFrame]: data = %v", fr.frame.Data.String())

	return
}

func (fr *FrameReader) Decode() (err error) {
	bf := bytebufferpool.Get()
	index := uint64(0)
	// read PayloadLen byte
	for index < fr.frame.PayloadLen {
		b, err := fr.buf.ReadByte()
		if err != nil {
			return ErrInComplete
		}

		if fr.frame.Mask {
			bf.WriteByte(b ^ fr.frame.MaskKey[index%4])
		} else {
			bf.WriteByte(b)
		}
		index++
	}
	fr.frame.Data = bf
	return
}

func (fr *FrameReader) Read(p []byte) (n int, err error) {
	if fr.frame == nil || fr.frame.Data == nil {
		return 0, io.EOF
	}
	n = copy(p, fr.frame.Data.B)
	fr.frame.Data.B = fr.frame.Data.B[n:]
	if n == 0 {
		if len(p) == 0 {
			return 0, nil
		}
		return 0, io.EOF
	}
	return
}

func (fr *FrameReader) RefreshFrame(f *Frame) {
	// put the old frame's data into buffer pool
	if fr.frame != nil && fr.frame.Data != nil {
		fr.frame.Data.Reset()
		bytebufferpool.Put(fr.frame.Data)
	}
	fr.frame = f
}

type FrameWriter struct {
	frame        *Frame
	buf          *bufio.Writer
	NeedMaskSet  bool
	MaxFrameSize int
}

func (fw *FrameWriter) WriteFrame() (err error) {
	logrus.Infof("[WriteFrame]: data = %v", fw.frame.Data.String())

	b := byte(1) << 7
	if !fw.frame.Fin {
		b = 0
	}
	// fin, opcode
	err = fw.buf.WriteByte(fw.frame.OpCode + b)
	if err != nil {
		return
	}

	// payload len
	b = byte(1) << 7
	if !fw.frame.Mask {
		b = 0
	}
	l := make([]byte, 0)
	if fw.frame.PayloadLen <= 125 {
		b += byte(fw.frame.PayloadLen)
		l = append(l, b)
	} else if fw.frame.PayloadLen <= math.MaxUint16 {
		b += 126
		t := make([]byte, 2)
		binary.BigEndian.PutUint16(t, uint16(fw.frame.PayloadLen))
		l = append(l, b, t[0], t[1])
	} else {
		b += 127
		l = append(l, b)
		t := make([]byte, 8)
		binary.BigEndian.PutUint64(t, fw.frame.PayloadLen)
		l = append(l, t...)
	}
	_, err = fw.buf.Write(l)
	if err != nil {
		return
	}

	// mask
	if fw.frame.Mask {
		_, err = fw.buf.Write(fw.frame.MaskKey[:])
		if err != nil {
			return
		}
	}

	// data
	_, err = fw.frame.Data.WriteTo(fw.buf)
	if err != nil {
		return
	}
	return fw.buf.Flush()
}

func (fw *FrameWriter) RefreshFrame(f *Frame) {
	// put the old frame's data into buffer pool
	if fw.frame != nil && fw.frame.Data != nil {
		fw.frame.Data.Reset()
		bytebufferpool.Put(fw.frame.Data)
	}
	fw.frame = f
}

func (fw *FrameWriter) NewFrame(status int, payloadType byte, data []byte) {
	frame := &Frame{
		Fin:        status == FragmentEnd,
		Rsv:        [3]bool{},
		OpCode:     payloadType,
		Mask:       fw.NeedMaskSet,
		PayloadLen: uint64(len(data)),
		MaskKey:    [4]byte{},
		Data:       nil,
	}
	if status == FragmentMiddle {
		frame.OpCode = PayloadTypeContinue
	}

	if frame.Mask {
		// generate random mask key
		binary.BigEndian.PutUint32(frame.MaskKey[:4], rand.Uint32())
	}
	bf := bytebufferpool.Get()
	for i := 0; i < len(data); i++ {
		if frame.Mask {
			bf.WriteByte(data[i] ^ frame.MaskKey[i%4])
		} else {
			bf.WriteByte(data[i])
		}
	}
	frame.Data = bf
	fw.RefreshFrame(frame)
}
