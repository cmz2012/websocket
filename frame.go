package websocket

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"io"
	"math"
)

type Frame struct {
	Fin        bool
	Rsv        [3]bool
	OpCode     byte
	Mask       bool
	PayloadLen uint64
	MaskKey    [4]byte
	Data       *bytes.Buffer
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

	fr.frame = &frame

	// decode byte read from buf
	err = fr.Decode()
	if err != nil {
		return
	}

	return
}

func (fr *FrameReader) Decode() (err error) {
	bf := bytes.NewBuffer(nil)
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

type FrameWriter struct {
	frame        *Frame
	buf          *bufio.Writer
	NeedMaskSet  bool
	MaxFrameSize int
}

func (fw *FrameWriter) WriteFrame() (err error) {
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
	_, err = fw.buf.ReadFrom(fw.frame.Data)
	if err != nil {
		return
	}
	return fw.buf.Flush()
}
