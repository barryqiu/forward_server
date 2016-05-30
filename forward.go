package main

import (
	"bufio"
	"strings"
	"net/http"
	"log"
	"fmt"
)

var phones map[string]Phone
var db_file_name string = "phones.txt"

func main() {
	phones = make(map[string]Phone)
	test()
	start_phones()
	http.HandleFunc("/", processClientReq) //设置访问的路由
	err := http.ListenAndServe(":8000", nil) //设置监听的端口
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
	fmt.Println("start listen 110")
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
