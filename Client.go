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
var header = `HTTP/1.1 200 OK
Cache-Control: no-store, no-cache, must-revalidate
Cache-Control: post-check=0, pre-check=0
Pragma: no-cache
Connection: close

`

var errHTML string = `HTTP/1.1 200 OK\r\nCache-Control: no-store, no-cache, must-revalidate\r\nCache-Control: post-check=0, pre-check=0\r\nPragma: no-cache\r\nConnection: close\r\n\r\n<!DOCTYPE HTML><head><meta http-equiv="Content-Type" content="text/html; charset=utf-8" /><meta name="viewport" content="user-scalable=no, initial-scale=1.0, maximum-scale=1.0 minimal-ui"/><meta name="apple-mobile-web-app-capable" content="yes"/><meta name="apple-mobile-web-app-status-bar-style" content="black"><link rel="icon" type="image/png" href="http://cdnyunphone.shinegame.cn/webh5/assets/images/splash/android-chrome-192x192.png" sizes="192x192"><link rel="apple-touch-icon" sizes="196x196" href="http://cdnyunphone.shinegame.cn/webh5/assets/images/splash/apple-touch-icon-196x196.png"><link rel="apple-touch-icon" sizes="180x180" href="http://cdnyunphone.shinegame.cn/webh5/assets/images/splash/apple-touch-icon-180x180.png"><link rel="apple-touch-icon" sizes="152x152" href="http://cdnyunphone.shinegame.cn/webh5/assets/images/splash/apple-touch-icon-152x152.png"><link rel="apple-touch-icon" sizes="144x144" href="http://cdnyunphone.shinegame.cn/webh5/assets/images/splash/apple-touch-icon-144x144.png"><link rel="apple-touch-icon" sizes="120x120" href="http://cdnyunphone.shinegame.cn/webh5/assets/images/splash/apple-touch-icon-120x120.png"><link rel="apple-touch-icon" sizes="114x114" href="http://cdnyunphone.shinegame.cn/webh5/assets/images/splash/apple-touch-icon-114x114.png"><link rel="apple-touch-icon" sizes="76x76" href="http://cdnyunphone.shinegame.cn/webh5/assets/images/splash/apple-touch-icon-76x76.png"><link rel="apple-touch-icon" sizes="72x72" href="http://cdnyunphone.shinegame.cn/webh5/assets/images/splash/apple-touch-icon-72x72.png"><link rel="apple-touch-icon" sizes="60x60" href="http://cdnyunphone.shinegame.cn/webh5/assets/images/splash/apple-touch-icon-60x60.png"><link rel="apple-touch-icon" sizes="57x57" href="http://cdnyunphone.shinegame.cn/webh5/assets/images/splash/apple-touch-icon-57x57.png"><link rel="icon" type="image/png" href="http://cdnyunphone.shinegame.cn/webh5/assets/images/splash/favicon-96x96.png" sizes="96x96"><link rel="icon" type="image/png" href="http://cdnyunphone.shinegame.cn/webh5/assets/images/splash/favicon-32x32.png" sizes="32x32"><link rel="icon" type="image/png" href="http://cdnyunphone.shinegame.cn/webh5/assets/images/splash/favicon-16x16.png" sizes="16x16"><link rel="shortcut icon" href="http://cdnyunphone.shinegame.cn/webh5/assets/images/splash/favicon.ico" type="image/x-icon" /><title>出错了</title><link href="http://cdnyunphone.shinegame.cn/webh5/assets/styles/style.css"           rel="stylesheet" type="text/css"><link href="http://cdnyunphone.shinegame.cn/webh5/assets/styles/menus.css"           rel="stylesheet" type="text/css"><link href="http://cdnyunphone.shinegame.cn/webh5/assets/styles/framework.css"       rel="stylesheet" type="text/css"><link href="http://cdnyunphone.shinegame.cn/webh5/assets/styles/font-awesome.css"    rel="stylesheet" type="text/css"><link href="http://cdnyunphone.shinegame.cn/webh5/assets/styles/animate.css"         rel="stylesheet" type="text/css"><script type="text/javascript" src="http://cdnyunphone.shinegame.cn/webh5/assets/scripts/jquery.js"></script><script type="text/javascript" src="http://cdnyunphone.shinegame.cn/webh5/assets/scripts/jqueryui.js"></script><script type="text/javascript" src="http://cdnyunphone.shinegame.cn/webh5/assets/scripts/framework-plugins.js"></script><script type="text/javascript" src="http://cdnyunphone.shinegame.cn/webh5/assets/scripts/custom.js"></script></head><body class="dual-sidebar"><div id="preloader"><div id="status"></div></div><div id="header-fixed" class="header-light"><a class="header-logo" href="#"></a><div class="header-menu-overlay"></div><div class="header-menu header-menu-light"></div></div><div id="footer-fixed" class="footer-menu footer-light disabled"></div><div class="gallery-fix"></div><div class="all-elements"><div class="snap-drawers"><div class="snap-drawer snap-drawer-left sidebar-light-clean"><div class="sidebar-header"></div><div class="sidebar-logo"></div><div class="sidebar-divider no-bottom"></div><p class="sidebar-divider">Navigation</p><div class="sidebar-menu"></div></div><div id="content" class="snap-content"><div class="header-clear"></div><div class="error-page bg-3 cover-screen"><div class="error-content cover-center"><div class="unboxed-layout"><h3>当前网络不稳定</h3><h4>请重试</h4><div onclick="window.location.reload()" class="back-home"><i class="fa fa-refresh"></i></div><a href="http://yunphone.shinegame.cn/webh5/index.html" class="back-home"><i class="fa fa-home"></i></a></div></div><div class="overlay bg-black"></div></div></div></div><a href="#" class="back-to-top-badge"><i class="fa fa-caret-up"></i>Back to top</a></div></body>`

func sendPhoneHtml(conn net.TCPConn) {
	defer conn.Close()

	//file to read
	file, err := os.Open(strings.TrimSpace("phone.html")) // For read access.
	if err != nil {

		log.Fatal(err)
	}

	defer file.Close()
	buf := make([]byte, 4096)
	conn.Write([]byte(header))
	for {
		n, _ := file.Read(buf)
		if 0 == n {
			break
		}
		conn.Write(buf[:n])
	}
	return
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
			conn.Write([]byte(errHTML))
			conn.Close()
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
		conn.Write([]byte(errHTML))
		conn.Close()
		return
	}
	for len(body) < int(req.ContentLength) {
		var buf = make([]byte, 4096)
		n, err := conn.Read(buf)
		if err != nil {
			log.Println("conn read error:", err)
			conn.Write([]byte(errHTML))
			conn.Close()
			return
		}
		body = append(body, buf[:n]...)

	}
	uri := req.RequestURI
	log.Println("URI:", uri)
	infos := strings.Split(uri, "/")
	if (len(infos) <= 1) {
		conn.Write([]byte(errHTML))
		conn.Close()
		log.Println("wrong url")
		return
	}
	device_name := infos[1]

	if len(infos) > 2 && infos[2] == "phone.html" {
		sendPhoneHtml(conn)
		log.Println(device_name + " phone.html")
		return
	}

	if _, ok := phones[device_name]; !ok {
		conn.Write([]byte(errHTML))
		conn.Close()
		log.Println(device_name + " not exist")
		return
	}

	// if uri like /device_name
	if len(infos) == 2 {
		header = bytes.Replace(header, [] byte(device_name), []byte("/" + device_name), 1)
	}

	var phone_conn net.TCPConn
	var isTest = false
	for {
		phone_conn, err = phones[device_name].get_conn()
		if (net.TCPConn{}) == phone_conn || err != nil {
			log.Println("no phone conn error:", err)
			conn.Write([]byte(errHTML))
			conn.Close()
			return
		}

		new_header := bytes.Replace(header, [] byte("/" + device_name), []byte(""), 1)
		data := append(new_header, body...)
		if (strings.Contains(uri, "/testconn")) {
			isTest = true
			data = []byte("GET /test HTTP/1.1\r\nHOST: anything\r\n\r\n")
			log.Println(device_name, "test conn")
		}
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
			conn.Write([]byte(errHTML))
			conn.Close()
			return
		}

		if isTest == true {
			 if string(buf[:n]) == "Webkey"{
				 conn.Write(header)
				 conn.Write([]byte("Phone is OK"))
			 }else {
				 conn.Write(header)
				 conn.Write([]byte("Phone is off line"))
			 }
			conn.Close()
			break
		}

		conn.Write(buf[:n])
		data_len += n
	}
	conn.Close()
	log.Println(uri, "receive", data_len)
	phone_conn.Close()
}