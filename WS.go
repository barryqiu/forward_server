package main

import (
	"flag"
	"github.com/gorilla/websocket"
	"net/http"
	"log"
	"strings"
	"net"
	"io"
	//"time"
	"bytes"
)

var address = flag.String("addr", ":8001", "http service address")

var upGrader = websocket.Upgrader{} // use default options

var sendRequestContent = `GET /screenshot.jpg?vlfnnn14670333662470 HTTP/1.1
Accept: image/webp,image/*,*/*;q=0.8
Accept-Encoding: gzip, deflate, sdch
Accept-Language: zh-CN,zh;q=0.8,en;q=0.6
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
	conn.WriteMessage(websocket.TextMessage, []byte(`{"action":"init","width":960,"height":540}`))
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

			_, err = phone_conn.Write([]byte(sendRequestContent))
			if err != nil {
				log.Println("send error", err)
			} else {
				break
			}
			phone_conn.Close()
		}

		data_len := 0
		var data [] byte
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
			start_index := 0
			header_index := bytes.Index(buf[:n], []byte("\r\n\r\n"))
			if header_index > 0 {
				start_index = header_index + 4
			}
			data = append(data, buf[start_index:n]...)
			//conn.WriteMessage(websocket.BinaryMessage, buf[start_index:n])
			data_len += n

		}
		conn.WriteMessage(websocket.BinaryMessage, data)
		//log.Println(uri, "send", len(data))
		phone_conn.Close()
		//time.Sleep(time.Millisecond * 20)
	}
}

func start_ws() {
	http.HandleFunc("/", get_screen)
	log.Println("listen web socket 8001 success")
	http.ListenAndServe(*address, nil)
}