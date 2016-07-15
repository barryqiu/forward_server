package main

import (
	"log"
	"net"
	"bytes"
	"strings"
	"io"
	"os"
)
// 策略:
// 使用一个全局的slice数组存储所有的Phone
// 接受客户端请求的部分使用GO语言的HTTP来处理,在这部分利用全局变量访问云端设备的链接
// 接受云端设备请求的部分使用Go的socket编程处理,同样使用全局变量来操作云端设备对象,实现添加链接等操作

// 测试链接是否可用：/{device_name}/testconn
var headerHtml = `HTTP/1.1 200 OK
Cache-Control: no-store, no-cache, must-revalidate
Cache-Control: post-check=0, pre-check=0
Pragma: no-cache
Connection: close

`

func renderHtmlFileAndClose(conn net.TCPConn, tpl string) {
	defer conn.Close()

	//file to read
	file, err := os.Open(strings.TrimSpace(tpl)) // For read access.
	if err != nil {

		log.Fatal(err)
	}

	defer file.Close()
	buf := make([]byte, 4096)
	conn.Write([]byte(headerHtml))
	for {
		n, _ := file.Read(buf)
		if 0 == n {
			break
		}
		conn.Write(buf[:n])
	}
	return
}

func renderHtmlString(conn net.TCPConn, content string) {
	conn.Write([]byte(headerHtml))
	conn.Write([]byte(content))
	return
}

func processTestConn(device_name string, conn net.TCPConn) {
	defer conn.Close()
	for i := 0; i < 3; i++ {
		var phone_conn net.TCPConn
		for {
			phone_conn, err := phones[device_name].get_conn()
			if (net.TCPConn{}) == phone_conn || err != nil {
				log.Println(device_name, "test conn no phone conn error:", err)
				renderHtmlFileAndClose(conn, "net_error.html")
				return
			}

			data := []byte("GET /test HTTP/1.1\r\nHOST: anything\r\n\r\n")
			_, err = phone_conn.Write(data)
			if err != nil {
				log.Println("send error", err)
			} else {
				break
			}
			phone_conn.Close()
		}

		var buf = make([]byte, 4096)
		n, err := phone_conn.Read(buf)

		if n == 0 || err == io.EOF {
			phone_conn.Close()
			continue
		}

		if err != nil {
			log.Println(device_name, "test conn read error:", err)
			renderHtmlFileAndClose(conn, "net_error.html")
			phone_conn.Close()
			return
		}

		if string(buf[:n]) == "Webkey" {
			renderHtmlString(conn, "Phone is OK")
			log.Println(device_name, "test conn OK")
			phone_conn.Close()
			break
		} else {
			renderHtmlString(conn, "Phone is off line")
			log.Println(device_name, "test conn off line")
			phone_conn.Close()
		}
	}
}

func processClientReq(conn net.TCPConn) {

	if (net.TCPConn{}) == conn {
		return
	}
	var content []byte
	var header []byte
	var body []byte
	for {
		var buf = make([]byte, 4096)
		n, err := conn.Read(buf)
		if err != nil {
			log.Println("cleint conn read error:", err)
			renderHtmlFileAndClose(conn, "net_error.html")
			return
		}
		content = append(content, buf[:n]...)
		header_end := bytes.Index(content, []byte("\r\n\r\n"))
		if (header_end != -1) {
			header = content[:header_end + 4]
			body = content[header_end + 4:]
			break
		}
	}

	req, err := getRequestInfo(string(header))
	if err != nil {
		log.Println("client conn read error:", err)
		renderHtmlFileAndClose(conn, "net_error.html")
		return
	}
	for len(body) < int(req.ContentLength) {
		var buf = make([]byte, 4096)
		n, err := conn.Read(buf)
		if err != nil {
			log.Println("conn read error:", err)
			renderHtmlFileAndClose(conn, "net_error.html")
			return
		}
		body = append(body, buf[:n]...)

	}
	uri := req.RequestURI
	log.Println("URI:", uri)
	infos := strings.Split(uri, "/")
	if (len(infos) <= 1) {
		renderHtmlFileAndClose(conn, "net_error.html")
		log.Println("wrong url")
		return
	}
	device_name := infos[1]

	if _, ok := phones[device_name]; !ok {
		renderHtmlFileAndClose(conn, "404.html")
		log.Println(device_name + " not exist")
		return
	}

	if len(infos) > 2 && strings.HasPrefix(infos[2], "phone.html") {
		renderHtmlFileAndClose(conn, "phone.html")
		log.Println(device_name + " phone.html")
		return
	}

	if (strings.Contains(uri, "/testconn")) {
		processTestConn(device_name, conn)
		log.Println(device_name + " testconn")
		return
	}


	// if uri like /device_name
	if len(infos) == 2 {
		header = bytes.Replace(header, [] byte(device_name), []byte(device_name + "/"), 1)
	}

	var phone_conn net.TCPConn
	for {
		phone_conn, err = phones[device_name].get_conn()
		if (net.TCPConn{}) == phone_conn || err != nil {
			log.Println("no phone conn error:", err)
			renderHtmlFileAndClose(conn, "net_error.html")
			return
		}

		new_header := bytes.Replace(header, [] byte("/" + device_name), []byte(""), 1)
		data := append(new_header, body...)
		_, err = phone_conn.Write(data)
		//log.Println("new request data", string(data))
		if err != nil {
			log.Println("send error", err)
		} else {
			break
		}
		phone_conn.Close()
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
			renderHtmlFileAndClose(conn, "net_error.html")
			return
		}

		data_len += n

		conn.Write(buf[:n])
	}
	conn.Close()
	log.Println(uri, "receive", data_len)
	phone_conn.Close()
}