package main

import (
	"fmt"
	"github.com/cmz2012/websocket"
)

func main() {
	c, err := websocket.Dial("ws://localhost:12345/echo")
	if err != nil {
		return
	}
	err = c.WriteControl(websocket.PayloadTypePing, []byte("first ping test"))
	fmt.Println(err)
	c.Close()
}
