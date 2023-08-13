package websocket

import "net"

type Conn struct {
	r  FrameReader
	w  FrameWriter
	nc net.Conn
}

func (c Conn) Read(p []byte) (n int, err error) {
	return c.r.Read(p)
}

func (c Conn) Write(p []byte) (n int, err error) {
	return c.w.Write(p)
}

func (c Conn) Close() error {
	return c.nc.Close()
}
