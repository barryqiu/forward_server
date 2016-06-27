package main

import (
	"flag"
	"github.com/gorilla/websocket"
	"net/http"
	"log"
	"strings"
	"net"
	"io"
	"time"
)

var address = flag.String("addr", ":8001", "http service address")

var upGrader = websocket.Upgrader{} // use default options

var sendRequestContent= `GET /screenshot.jpg HTTP/1.1\r\nConnection: keep-alive\r\nAccept: */*\r\nAccept-Encoding: gzip, deflate, sdch\r\nAccept-Language: zh-CN,zh;q=0.8,en;q=0.6\r\n\r\n`

func get_screen(w http.ResponseWriter, req *http.Request) {
	req.Header["Origin"] = nil
	conn, err := upGrader.Upgrade(w, req, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer conn.Close()
	uri := req.RequestURI
	log.Println("URI:", uri)
	infos := strings.Split(uri, "/")
	if (len(infos) <= 1) {
		log.Println("wrong url")
		return
	}
	device_name := infos[1]

	if _, ok := phones[device_name]; !ok {
		log.Println(device_name + " not exist")
		return
	}

	for {
		var phone_conn net.TCPConn
		for {
			phone_conn, err = phones[device_name].get_conn()
			if (net.TCPConn{}) == phone_conn || err != nil {
				log.Println("no phone conn error:", err)
				conn.WriteMessage(websocket.TextMessage, []byte("no phone conn error"))
				conn.Close()
				return
			}
			phone_conn.Write([]byte(sendRequestContent))
		}

		data_len := 0
		for {
			var buf = make([]byte, 4096)
			n, err := phone_conn.Read(buf)

			if n == 0 || err == io.EOF {
				break
			}

			if err != nil {
				log.Println("conn read error:", err)
				conn.WriteMessage(websocket.TextMessage, []byte("no data error"))
				return
			}
			conn.WriteMessage(websocket.BinaryMessage,buf[:n])
			data_len += n
		}
		log.Println(uri, "receive", data_len)
		phone_conn.Close()
	}
	time.Sleep(time.Millisecond * 10)
}

func start_ws() {
	http.HandleFunc("/", get_screen)
	log.Println("listen web socket 8001 success")
	http.ListenAndServe(*address, nil)
}