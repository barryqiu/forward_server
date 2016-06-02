package main

import (
	"bufio"
	"strings"
	"net/http"
	"log"
	"net"
	"os"
)

var phones map[string] *Phone
var db_file_name string = "phones.txt"

func main() {

	f, err := os.OpenFile("testlogfile", os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
	if err != nil {
		return
	}
	defer f.Close()

	log.SetOutput(f)

	phones = make(map[string]*Phone)
	test()
	go start_phones()

	add, err := net.ResolveTCPAddr("tcp", ":8000")
	if err != nil {
		log.Println("error listen:", err)
		return
	}
	listen, err := net.ListenTCP("tcp", add)
	if err != nil {
		log.Println("error listen:", err)
		return
	}
	defer listen.Close()
	log.Println("client listen 8000 ok")

	for {
		conn, err := listen.AcceptTCP()
		if err != nil {
			log.Println("client accept error:", err)
		}
		go processClientReq(*conn)
		log.Printf("accept a new client connection\n")
	}
}

func getRequestInfo(str string) (http.Request, error) {
	reader := bufio.NewReader(strings.NewReader(str))
	req, err := http.ReadRequest(reader)
	if err != nil {
		log.Println("parse http request error:", err)
		return *req, err
	}
	return *req, nil
}

func test() {
	return
}
