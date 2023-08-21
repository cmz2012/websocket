package websocket

import (
	"bufio"
	"crypto/rand"
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
	IsServer     bool
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

func (up *Upgrader) generateKey() error {
	// 1. 16-byte value
	p := make([]byte, 16)

	// 2. Randomly selected
	if _, err := io.ReadFull(rand.Reader, p); err != nil {
		return err
	}

	// 3. Base64-encoded
	up.websocketKey = base64.StdEncoding.EncodeToString(p)
	return nil
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

func (up *Upgrader) Connect(buf *bufio.Writer) (err error) {
	fmt.Fprintf(buf, "GET %s HTTP/1.1\r\n", up.url.Path)
	fmt.Fprintf(buf, "Host: %s\r\n", up.url.Host)
	fmt.Fprintf(buf, "Upgrade: websocket\r\n")
	fmt.Fprintf(buf, "Connection: Upgrade\r\n")
	fmt.Fprintf(buf, "Sec-WebSocket-Key: %s\r\n", up.websocketKey)
	fmt.Fprintf(buf, "\r\n")
	return buf.Flush()
}

func (up *Upgrader) CheckShakeResp(rsp *http.Response) (err error) {
	if rsp.StatusCode != http.StatusSwitchingProtocols {
		return errors.New(fmt.Sprintf("status code: %d", rsp.StatusCode))
	}
	header := rsp.Header

	if header.Get("Upgrade") != "websocket" || header.Get("Connection") != "Upgrade" || header.Get("Sec-WebSocket-Accept") != up.accept {
		return errors.New("header error")
	}
	return
}

func (up *Upgrader) Upgrade(w http.ResponseWriter, r *http.Request) (cn *Conn, err error) {
	conn, buf, err := w.(http.Hijacker).Hijack()
	if err != nil {
		panic(err)
	}
	cn = &Conn{
		r: FrameReader{
			frame:       nil,
			buf:         buf.Reader,
			NeedMaskSet: up.IsServer,
		},
		w: FrameWriter{
			frame:       nil,
			buf:         buf.Writer,
			NeedMaskSet: !up.IsServer,
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
