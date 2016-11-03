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
	"time"
	//"io/ioutil"
	"fmt"
	//"encoding/base64"
	//"os"
	"encoding/json"
	//"errors"
)

var (
	address = flag.String("addr", ":8001", "http service address")

	upGrader = websocket.Upgrader{} // use default options

	sendVRequestContent = `GET /screenshot.jpg?vlfnnn14670333662470 HTTP/1.1
Accept: image/webp,image/*,*/*;q=0.8
Accept-Encoding: gzip, deflate, sdch
Accept-Language: zh-CN,zh;q=0.8,en;q=0.6
Cache-Control: max-age=259200
Connection: keep-alive

`
	sendHRequestContent = `GET /screenshot.jpg?hlfnnn14670333662470 HTTP/1.1
Accept: image/webp,image/*,*/*;q=0.8
Accept-Encoding: gzip, deflate, sdch
Accept-Language: zh-CN,zh;q=0.8,en;q=0.6
Cache-Control: max-age=259200
Connection: keep-alive

`
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 5 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10
)

type ClientParam struct {
	DeviceType string `json:"type"`
	Token      string `json:"token"`
}

//func judge_auth(token string, deviceName string) error {
//	//os.Setenv("HTTP_PROXY", "http://proxy.tencent.com:8080")
//	client := &http.Client{}
//
//	req, err := http.NewRequest("GET", "http://yunphone.shinegame.cn/api/1.1/device/user", nil)
//	if err != nil {
//		errors.New("http error")
//	}
//
//	base64Token := base64.StdEncoding.EncodeToString([]byte(token + ":"))
//
//	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", base64Token))
//	resp, err := client.Do(req)
//
//	defer resp.Body.Close()
//
//	body, err := ioutil.ReadAll(resp.Body)
//	if err != nil {
//		errors.New("http error")
//	}
//
//	var f interface{}
//	err = json.Unmarshal(body, &f)
//
//	data := f.(map[string]interface{})
//	content := data["content"]
//	devices := content.([]interface{})
//	for _, device := range devices {
//		device_map := device.(map[string]interface{})
//		device_name := device_map["device_name"].(string)
//		if device_name == deviceName {
//			return nil
//		}
//	}
//
//	return errors.New("no auth")
//}

type ClientConn struct {
	alias string
	ws    *websocket.Conn
	stop  chan int
	send  chan []byte
}

// readPump pumps messages from the websocket connection to the hub.
func (c *ClientConn) readPump() {
	defer func() {
		c.ws.Close()
		c.stop <- 1
	}()
	c.ws.SetReadDeadline(time.Now().Add(pongWait))
	c.ws.SetPongHandler(func(string) error {
		c.ws.SetReadDeadline(time.Now().Add(pongWait)); return nil
	})
	for {
		_, _, err := c.ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				log.Printf("error: %v", err)
			}
			log.Printf("%v c.ws.ReadMessage", err)
			break
		}
	}
}

// write writes a message with the given message type and payload.
func (c *ClientConn) write(mt int, payload []byte) error {
	c.ws.SetWriteDeadline(time.Now().Add(writeWait))
	return c.ws.WriteMessage(mt, payload)
}

// writePump pumps messages from the hub to the websocket connection.
func (c *ClientConn) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.ws.Close()
		c.stop <- 1
	}()
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				// The hub closed the channel.
				log.Printf("%v c.send not ok", ok)
				c.write(websocket.CloseMessage, []byte{})
				return
			}

			c.ws.SetWriteDeadline(time.Now().Add(writeWait))
			w, err := c.ws.NextWriter(websocket.BinaryMessage)
			if err != nil {
				log.Printf("%v NextWriter", err)
				return
			}

			w.Write(message)

			if err := w.Close(); err != nil {
				log.Printf("%v w.Close", err)
				return
			}
		case <-ticker.C:
			if err := c.write(websocket.PingMessage, []byte{}); err != nil {
				log.Printf("%v write ping message", err)
				return
			}
			log.Printf("%v tick", c.alias)
			if (c.alias != "") {
				// 判断地址是否还存在，如果不存在则应该停止WS
				device_map, err := trans_phone_address(c.alias)
				if _, ok := phones[device_map]; err != nil || !ok {
					log.Printf("phone map not exist, clost ws %v", c.alias)
					return
				}
			}
		}
	}
}

func get_screen(w http.ResponseWriter, req *http.Request) {
	req.Header["Origin"] = nil
	conn, err := upGrader.Upgrade(w, req, nil)
	if err != nil {
		log.Print("upgrade:", err)
		conn.Close()
		return
	}
	log.Println("receive client conn from address", conn.RemoteAddr().String())
	conn.WriteMessage(websocket.TextMessage, []byte(`{"action":"init","width":960,"height":540}`))

	device_type := "v"
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	var clientParam ClientParam
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				log.Printf("error: %v", err)
			}
			break
		}
		err = json.Unmarshal(message, &clientParam)
		device_type = string(clientParam.DeviceType)
		break
	}

	uri := req.RequestURI
	log.Println("URI:", uri)
	infos := strings.Split(uri, "/")
	if (len(infos) <= 1) {
		log.Println("wrong url")
		conn.Close()
		return
	}
	device_name := infos[1]
	alias := ""

	//err = judge_auth(clientParam.token,device_name)
	//if err != nil{
	//	log.Println(device_name + " wrong auth")
	//	conn.Close()
	//	return
	//}

	if _, ok := phones[device_name]; !ok {
		device_map, err := trans_phone_address(device_name)
		if _, ok := phones[device_map]; err == nil && ok {
			alias = device_name
			device_name = device_map
		} else {
			log.Println(device_name + " not exist")
			conn.Close()
			return
		}
	}

	ws_state := get_phone_ws_state_in_redis(device_name)
	if ws_state == 1{
		phones[device_name].log_to_file(device_name + " is in use")
		conn.WriteMessage(websocket.TextMessage, []byte("device is in use"))
		conn.Close()
		return
	}

	phones[device_name].log_to_file(fmt.Sprintf("param : %v", clientParam))
	//log.Printf("param : %v\n", clientParam)

	clientConn := &ClientConn{send: make(chan []byte, 4096), ws: conn, stop: make(chan int), alias: alias}

	go clientConn.writePump()
	go clientConn.readPump()

	set_phone_ws_state_in_redis(device_name, 1)

	for {
		select {
		case stop := <-clientConn.stop:
			if stop == 1 {
				phones[device_name].log_to_file("client close, stop fetch data")
				//log.Println("client close, stop fetch data")
				set_phone_ws_state_in_redis(device_name, 0)
				return
			}
		default:
			var phone_conn net.TCPConn
			retry := 0
			for {
				phone_conn, err = phones[device_name].get_conn()
				if (net.TCPConn{}) == phone_conn || err != nil {
					phones[device_name].log_to_file("no phone conn error:", err)
					//log.Println("no phone conn error:", err)
					if (retry >= 3) {
						conn.WriteMessage(websocket.TextMessage, []byte("no phone conn error"))
						conn.Close()
						set_phone_ws_state_in_redis(device_name, 0)
						return
					}
					retry++
					time.Sleep(time.Millisecond * 50)
					continue
				}
				if device_type == "h" {
					_, err = phone_conn.Write([]byte(sendHRequestContent))
					if err != nil {
						//log.Println("send error", err)
						phones[device_name].log_to_file("send error", err)
					} else {
						break
					}
				} else {
					_, err = phone_conn.Write([]byte(sendVRequestContent))
					if err != nil {
						//log.Println("send error", err)
						phones[device_name].log_to_file("send error", err)
					} else {
						break
					}
				}
				phone_conn.Close()
			}

			data_len := 0
			var data [] byte
			for {
				var buf = make([]byte, 4096)
				n, err := phone_conn.Read(buf)

				if err == io.EOF {
					break
				}

				if err != nil {
					//log.Println("conn read error:", err)
					phones[device_name].log_to_file("conn read error:", err)
					//conn.WriteMessage(websocket.TextMessage, []byte("no data error"))
					set_phone_ws_state_in_redis(device_name, 0)
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
			clientConn.send <- data
		//log.Println(uri, "send", len(data))
			phones[device_name].log_to_file(uri, "send", len(data))
			phone_conn.Close()
			time.Sleep(time.Millisecond * 50)
		}

	}
}

func start_ws() {
	http.HandleFunc("/", get_screen)
	log.Println("listen web socket 8001 success")
	http.ListenAndServe(*address, nil)
}
