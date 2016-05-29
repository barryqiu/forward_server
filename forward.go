package main

import (
	"fmt"
	"bufio"
	"strings"
	"net/http"
	"log"
)

var HOST = ""
var PORT = 8000

func main() {
	slice := []byte{'a'}
	slice = slice[1:]
	fmt.Printf("%d", len(slice))

	fmt.Println(HOST, PORT)

	//start_phones()
	logEntry := "Content-Encoding: gzip\r\nLast-Modified: Tue, 20 Aug 2013 15:45:41 GMT\r\nServer: nginx/0.8.54\r\nAge: 18884\r\nVary: Accept-Encoding\r\nContent-Type: text/html\r\nCache-Control: max-age=864000, public\r\nX-UA-Compatible: IE=Edge,chrome=1\r\nTiming-Allow-Origin: *\r\nContent-Length: 14888\r\nExpires: Mon, 31 Mar 2014 06:45:15 GMT\r\nyp-token: 1233\r\n"

	reader := bufio.NewReader(strings.NewReader("GET / HTTP/1.1\r\n" + logEntry + "\r\n"))

	req, err := http.ReadRequest(reader)
	if err != nil {
		log.Fatal(err)
	}

	//log.Println(req.Header)
	fmt.Println(req.Header.Get("yp-token"))
	fmt.Println(req.RequestURI)
}

func getRequestInfo(str string) (http.Request, error) {
	reader := bufio.NewReader(strings.NewReader(str))
	req, err := http.ReadRequest(reader)
	if err != nil {
		log.Println("parse http request error:", err)
		return nil, err
	}
	return req, nil
}

