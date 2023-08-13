package websocket

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"net/url"
)

type Upgrader struct {
	websocketKey string
	accept       string
	url          *url.URL
}

func (up *Upgrader) getKeyAccept(nonce []byte) (expected []byte, err error) {
	h := sha1.New()
	if _, err = h.Write(nonce); err != nil {
		return
	}
	if _, err = h.Write([]byte(websocketGUID)); err != nil {
		return
	}
	expected = make([]byte, 28)
	base64.StdEncoding.Encode(expected, h.Sum(nil))
	return
}

func (up *Upgrader) writeShakeSuccessResponse(buf *bufio.ReadWriter) {
	str := fmt.Sprintf("HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: %s\r\n\r\n", up.accept)
	buf.WriteString(str)
	buf.Flush()
	logrus.Infof("[Upgrade]: writeShakeSuccessResponse")
}

func (up *Upgrader) writeShakeBadResponse(buf *bufio.ReadWriter) {
	fmt.Fprintf(buf, "HTTP/1.1 %03d %s\r\n", http.StatusBadRequest, http.StatusText(http.StatusBadRequest))
	fmt.Fprintf(buf, "Sec-WebSocket-Version: %d\r\n", 13)
	buf.WriteString("\r\n")
	buf.WriteString("websocket handshake error")
	buf.Flush()
	logrus.Errorf("[Upgrade]: writeShakeBadResponse")
}

func (up *Upgrader) Upgrade(w http.ResponseWriter, r *http.Request) (rwc io.ReadWriteCloser, err error) {
	conn, buf, err := w.(http.Hijacker).Hijack()
	if err != nil {
		panic(err)
	}
	rwc = Conn{
		r: FrameReader{
			frame:       nil,
			buf:         buf.Reader,
			NeedMaskSet: true,
		},
		w: FrameWriter{
			frame:       nil,
			buf:         buf.Writer,
			NeedMaskSet: false,
		},
		nc: conn,
	}
	// 一次握手
	method := r.Method
	upgrade := r.Header.Get("Upgrade")
	connection := r.Header.Get("Connection")
	key := r.Header.Get("Sec-WebSocket-Key")
	up.websocketKey = key
	up.url = r.URL
	logrus.Infof("[Upgrade]: url = %v", *(up.url))
	if method != "GET" || upgrade != "websocket" || connection != "Upgrade" {
		up.writeShakeBadResponse(buf)
		err = errors.New("shake hand error")
		conn.Close()
		return
	}
	accept, err := up.getKeyAccept([]byte(key))
	if err != nil {
		logrus.Errorf("[Upgrade] getKeyAccept: %v", err)
		up.writeShakeBadResponse(buf)
		conn.Close()
		return
	}
	up.accept = string(accept)

	// 写握手的response
	up.writeShakeSuccessResponse(buf)
	return
}
