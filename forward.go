package main

import (
	"bufio"
	"strings"
	"net/http"
	"log"
	"net"
	"os"
	"fmt"
	"time"
	"path/filepath"
	"github.com/garyburd/redigo/redis"
)

var phones map[string]*Phone
var db_file_name string = "phones.txt"

func getRedisConn() (redis.Conn,error) {
	return  redis.DialURL("redis://:doodod123@localhost:6542/0")
}

func main() {

	string_date := current_date_string()
	os.MkdirAll("log" + string(filepath.Separator) + string_date, 06660)
	log_file_name := "log" + string(filepath.Separator) + "info.log"
	f, err := os.OpenFile(log_file_name, os.O_RDWR | os.O_CREATE | os.O_APPEND, 0666)
	if err != nil {
		return
	}

	defer func() {
		f.Close()
	}()

	log.SetOutput(f)

	phones = make(map[string]*Phone)
	go start_phones()

	go start_ws()

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
	}
}

func getRequestInfo(str string) (http.Request, error) {
	reader := bufio.NewReader(strings.NewReader(str))
	req, err := http.ReadRequest(reader)
	if err != nil {
		log.Println("parse http request error:", err)
		return http.Request{}, err
	}
	return *req, nil
}

func current_date_string() string {
	t := time.Now()
	return fmt.Sprintf("%d%02d%02d", t.Year(), t.Month(), t.Day())
}
func current_time_string() string {
	t := time.Now()
	return fmt.Sprintf("%d-%02d-%02d %02d:%02d:%02d", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
}