package main

import (
	"fmt"
	"github.com/cmz2012/websocket"
	"github.com/sirupsen/logrus"
	"net/http"
	"time"
)

func HandConn(rwc *websocket.Conn) {
	defer rwc.Close()
	go func() {
		cnt := 1
		for {
			time.Sleep(5 * time.Second)
			s := fmt.Sprintf("ping number: %v", cnt)
			rwc.WriteControl(websocket.PayloadTypePing, []byte(s))
			cnt++
		}
	}()
	msg, t, err := rwc.ReadMessage()
	fmt.Println(string(msg), t, err)
}

func socketHandler(w http.ResponseWriter, r *http.Request) {
	up := &websocket.Upgrader{IsServer: true}
	rwc, err := up.Upgrade(w, r)
	if err != nil {
		logrus.Infoln(err)
		return
	}
	// handler
	HandConn(rwc)
}

func main() {
	http.HandleFunc("/echo", socketHandler)
	//err := http.ListenAndServe(":12345", nil)
	err := http.ListenAndServeTLS(":12345", "server.crt", "server.key", nil)
	if err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}
