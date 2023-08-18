package websocket

import (
	"encoding/json"
	"errors"
)

func (c *Conn) WriteJson(v any) (err error) {
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	_, err = c.WriteMessage(PayloadTypeBinary, b)
	return
}

func (c *Conn) ReadJson(v any) (err error) {
	msg, payloadType, err := c.ReadMessage()
	if err != nil {
		return
	}
	if payloadType != PayloadTypeBinary {
		return errors.New("must be PayloadTypeBinary")
	}
	return json.Unmarshal(msg, v)
}
