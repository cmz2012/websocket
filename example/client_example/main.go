package main

import (
	"fmt"
	"github.com/cmz2012/websocket"
)

func client() {
	ws, err := websocket.Dial("wss://localhost:12345/echo")
	fmt.Println(err)
	_, _, _ = ws.ReadMessage()
}

func main() {
	client()
}
