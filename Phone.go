package main

import (
	"log"
	"net"
	"strings"
	"sync"
	"time"
	"os"
	"bufio"
	"io"
	"fmt"
)

type Phone struct {
	mu             sync.Mutex
	Conn_list      []net.TCPConn
	User_name      string
	Random         string
	Last_known     net.TCPAddr
	Redirect       net.TCPAddr
	Overhead       int
	Overheat_count int
}

func (phone Phone) append_conn(conn net.TCPConn, address net.TCPAddr) error {
	phone.mu.Lock()

	err := conn.SetKeepAlive(true)
	if err != nil {
		log.Println("set keep alive error:", err)
	}

	err = conn.SetKeepAlivePeriod(time.Second * 240)
	if err != nil {
		log.Println("set keep alive period error:", err)
	}

	phone.Conn_list = append(phone.Conn_list, conn)

	if len(phone.Conn_list) > phone.Overhead {
		conn0 := phone.Conn_list[0]
		err = conn0.Close()
		if err != nil {
			log.Println("close conn error:", err)
		}
		phone.Conn_list = phone.Conn_list[1:]

		if phone.Overheat_count > 100 {
			phone.Overheat_count = 0
			if phone.Overhead > 10 {
				phone.Overhead = 10
			} else {
				phone.Overhead = phone.Overhead + 1
			}
		}

	}

	if phone.Last_known.String() != address.String() {
		log.Println("not last knoen ip")
		log.Println("Last known IP:" + phone.Last_known.String() + "new:" + address.String())
		for len(phone.Conn_list) > 1 {
			conn0 := phone.Conn_list[0]
			err = conn0.Close()
			if err != nil {
				log.Println("close conn error:", err)
			}
			phone.Conn_list = phone.Conn_list[1:]
		}
		phone.Last_known = address
		phone.Redirect = net.TCPAddr{}
		phone.Overheat_count = 0
		phone.Overhead = 0
		go test_phone(address, phone)
	}

	if phone.Redirect.IP != nil {
		conn.Write([]byte("stop"))
		for len(phone.Conn_list) > 0 {
			conn0 := phone.Conn_list[0]
			err = conn0.Close()
			if err != nil {
				log.Println("close conn error:", err)
			}
			phone.Conn_list = phone.Conn_list[1:]
		}
	}

	phone.mu.Unlock()
	return nil
}

func (phone Phone) get_conn() (conn net.TCPConn, err error) {
	exp_time := time.Now().Add(time.Second * 1)
	phone.mu.Lock()
	for len(phone.Conn_list) == 0 {
		time.Sleep(time.Millisecond * 500)
		if exp_time.Before(time.Now()) {
			break
		}
	}

	if len(phone.Conn_list) == 0 {
		log.Println("Got signal , bu no connection " + phone.User_name)
		phone.mu.Unlock()
		return net.TCPConn{}, nil
	}

	conn0 := phone.Conn_list[0]
	if err != nil {
		log.Println("close conn error:", err)
	}
	phone.Conn_list = phone.Conn_list[1:]

	phone.mu.Unlock()
	return conn0, nil
}

func (phone Phone) close_all_conn() error {
	phone.mu.Lock()
	for len(phone.Conn_list) > 0 {
		conn0 := phone.Conn_list[0]
		err := conn0.Close()
		if err != nil {
			log.Println("close conn error:", err)
		}
		phone.Conn_list = phone.Conn_list[1:]
	}
	phone.mu.Unlock()
	return nil
}

func (phone Phone) init(user_name string, random string, conn net.TCPConn, address net.TCPAddr) error {
	phone.User_name = user_name
	phone.Random = random
	phone.Last_known = net.TCPAddr{}
	phone.Redirect = net.TCPAddr{}
	phone.Overhead = 0
	phone.Overheat_count = 0
	if conn == (net.TCPConn{}) {
		phone.append_conn(conn, address)
	}

	return nil
}

func (phone Phone) add_to_file() error {
	fl, err := os.OpenFile(db_file_name, os.O_CREATE | os.O_APPEND | os.O_RDWR, 0660)
	defer fl.Close()
	if (err != nil) {
		log.Println("open file error", err)
		return err
	}
	fl.WriteString(phone.User_name + " " + phone.Random + "\n")
	return nil
}

/**
read phone‘s info from file
 */
func read_phones_from_file() {
	fl, err := os.Open(db_file_name)
	if err != nil {
		log.Println("open file error", err)
		return
	}
	defer fl.Close()
	buff := bufio.NewReader(fl)
	for {
		line, err := buff.ReadString('\n')
		if err != nil || io.EOF == err {
			break
		}
		infos := strings.Split(" ", line)
		if len(infos) == 2 {
			phone := Phone{}
			phone.User_name = infos[0]
			phone.Random = infos[1]
			phones = append(phones, phone)
		}
	}
}

/**
start phone thread to listen request from yun phone
*/
func start_phones() {

	read_phones_from_file()

	add, err := net.ResolveTCPAddr("tcp", ":8110")
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
	log.Println("listen ok")

	var i int
	for {
		conn, err := listen.AcceptTCP()
		if err != nil {
			log.Println("accept error:", err)
		}
		go process_phone_conn(*conn)
		log.Printf("%d: accept a new connection\n", i)
	}
}

/**
test the phone conn
*/
func test_phone(address net.TCPAddr, phone Phone) {
	log.Println("test routine start")
	conn, err := net.DialTimeout("tcp", address.String(), 2 * time.Second)
	defer conn.Close()
	if err != nil {
		log.Println("dial  error:", err)
	}
	conn.Write([]byte("GET /test HTTP/1.1\r\nHOST: anything\r\n\r\n"))
	var data []byte
	n, err := conn.Read(data)
	if err != nil {
		log.Println("dial  error:", err)
	}
	log.Println("test routine receive data ", data)
	if n > 0 && string(data) == "Webkey" {
		phone.Redirect = address
	}
	log.Println("it can be redirected to ", address)
	// read or write on conn
}

func process_phone_conn(conn net.TCPConn) {
	if (net.TCPConn{}) == conn {
		return
	}
	var buf = make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		log.Println("conn read error:", err)
		return
	}
	first_line := string(buf[:n])

	fmt.Println(first_line)

	pos := strings.Index(first_line, "/")
	if pos == -1 {
		conn.Close()
		return
	}

	if strings.HasPrefix(first_line, "GET /register_") {
		req, err := getRequestInfo(first_line)
		if err != nil {
			log.Println("conn read error:", err)
			return
		}
		infos := strings.Split(req.RequestURI, "/")
		user_name := infos[1]
		random := infos[2]
		version := infos[3]
		log.Println("user_name:" + user_name + ";random:" + random + ";version:" + version)

		if len(user_name) == 0 {
			conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\nEmpty username is not allowed."))
			conn.Close()
			return
		}

		for _, one_phone :=range phones {
			if (one_phone.User_name == user_name &&	one_phone.Random != random ){
				conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\nUsername is already used."))
				conn.Close()
				return
			}
		}

		phone := Phone{}
		phone.User_name = user_name
		phone.Random = random
		phone.add_to_file()
		conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\nOK"))
		conn.Close()
		return

	}

	if strings.HasPrefix(first_line, "WEBKEY") {
	}

}
