package main

import (
	"fmt"
	"github.com/cmz2012/websocket"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
)

func HandConn(rwc io.ReadWriteCloser) {
	defer rwc.Close()
	msg := make([]byte, 10)
	n, err := rwc.Read(msg)
	fmt.Printf("ReadString: %v, %v, %v\n", string(msg), err, n)
	n, err = rwc.Write([]byte("hello, client!"))
	fmt.Printf("WriteString: %v, %v", n, err)
}

func socketHandler(w http.ResponseWriter, r *http.Request) {
	up := &websocket.Upgrader{}
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
	err := http.ListenAndServe(":12345", nil)
	if err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}
