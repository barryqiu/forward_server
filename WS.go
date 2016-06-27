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

var sendRequestContent = `GET /screenshot.jpg?vlfnnn14670333662470 HTTP/1.1
Host: 101.201.37.72:8000
User-Agent: Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/51.0.2704.84 Safari/537.36
Accept: image/webp,image/*,*/*;q=0.8
Referer: http://101.201.37.72:8000/IYIACJY5CN/phone.html
Accept-Encoding: gzip, deflate, sdch
Accept-Language: zh-CN,zh;q=0.8,en;q=0.6
Cookie: leftWidgetList=%5B%5D; time=1467033365808160; lang=zh
Cache-Control: max-age=259200
Connection: keep-alive

`

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
			log.Printf("%v", phones[device_name])
			phone_conn, err = phones[device_name].get_conn()
			if (net.TCPConn{}) == phone_conn || err != nil {
				log.Println("no phone conn error:", err)
				conn.WriteMessage(websocket.TextMessage, []byte("no phone conn error"))
				conn.Close()
				return
			}

			_, err = phone_conn.Write([]byte(sendRequestContent))
			if err != nil {
				log.Println("send error", err)
			} else {
				break
			}
			phone_conn.Close()
		}

		data_len := 0
		for {
			log.Println("start receive data")
			var buf = make([]byte, 4096)
			n, err := phone_conn.Read(buf)
			log.Println(uri, "receive", n)

			if n == 0 || err == io.EOF {
				break
			}

			if err != nil {
				log.Println("conn read error:", err)
				conn.WriteMessage(websocket.TextMessage, []byte("no data error"))
				return
			}
			conn.WriteMessage(websocket.BinaryMessage, buf[:n])
			data_len += n

		}
		log.Println(uri, "receive", data_len)
		phone_conn.Close()
		time.Sleep(time.Millisecond * 50)
	}
}

func start_ws() {
	http.HandleFunc("/", get_screen)
	log.Println("listen web socket 8001 success")
	http.ListenAndServe(*address, nil)
}