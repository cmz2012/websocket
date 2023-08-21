package websocket

import (
	"bufio"
	"github.com/sirupsen/logrus"
	"net"
	"net/http"
	"net/url"
)

// Dial 建议tcp连接，发送GET upgrade请求
func Dial(rawUrl string) (c *Conn, err error) {
	_url, err := url.Parse(rawUrl)
	if err != nil {
		return
	}
	logrus.Infof("[Dial: url = %v", *_url)

	upgrader := &Upgrader{
		websocketKey: "",
		accept:       "",
		url:          _url,
		IsServer:     false,
	}
	err = upgrader.generateKey()
	if err != nil {
		return
	}
	accept, err := upgrader.getKeyAccept([]byte(upgrader.websocketKey))
	if err != nil {
		return
	}
	upgrader.accept = string(accept)

	// tcp connection
	conn, err := net.Dial("tcp", _url.Host)
	if err != nil {
		logrus.Errorf("[Dial]: host = %v, err = %v", _url.Host, err)
		return
	}
	logrus.Infof("[Dial]: tcp connected remote = %v", conn.RemoteAddr().String())
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	// http upgrade
	err = upgrader.Connect(writer)
	if err != nil {
		logrus.Errorf("[Dial]: websocket upgrade http fails: %v", err)
		return
	}

	rsp, err := http.ReadResponse(reader, nil)
	if err != nil {
		logrus.Errorf("[Dial]: websocket upgrade response: %v", err)
		return
	}

	err = upgrader.CheckShakeResp(rsp)
	if err != nil {
		logrus.Errorf("[Dial]: %v", err)
		return
	}

	c = &Conn{
		r: FrameReader{
			frame:       nil,
			buf:         reader,
			NeedMaskSet: false,
		},
		w: FrameWriter{
			frame:        nil,
			buf:          writer,
			NeedMaskSet:  true,
			MaxFrameSize: 0,
		},
		nc:          conn,
		pingHandle:  nil,
		pongHandle:  nil,
		closeHandle: nil,
	}
	return
}
