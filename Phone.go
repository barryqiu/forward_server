package main

import (
	"log"
	"net"
	"sync"
	"time"
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

	append(phone.Conn_list, conn)

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

	if phone.Last_known != address {
		log.Println("not last knoen ip")
		log.Println("Last known IP:" + phone.Last_known + "new:" + address)
		for len(phone.Conn_list) > 1 {
			conn0 := phone.Conn_list[0]
			err = conn0.Close()
			if err != nil {
				log.Println("close conn error:", err)
			}
			phone.Conn_list = phone.Conn_list[1:]
		}
		phone.Last_known = address
		phone.Redirect = nil
		phone.Overheat_count = 0
		phone.Overhead = 0
		go test_phone(address, phone)
	}

	if phone.Redirect {
		conn.Write("stop")
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
	exp_time := time.Now() + time.Second*1
	phone.mu.Lock()
	for len(phone.Conn_list) == 0 {
		time.Sleep(time.Millisecond * 500)
		if exp_time < time.Now() {
			break
		}
	}

	if len(phone.Conn_list) == 0 {
		log.Println("Got signal , bu no connection " + phone.User_name)
		phone.mu.Unlock()
		return nil, nil
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
	phone.Last_known = nil
	phone.Redirect = nil
	phone.Overhead = 0
	phone.Overheat_count = 0
	if conn != nil {
		phone.append_conn(conn, address)
	}

	return nil
}

/**
start phone thread to listen request from yun phone
*/
func start_phones() {

	listen, err := net.Listen("tcp", ":110")
	if err != nil {
		log.Println("error listen:", err)
		return
	}
	defer listen.Close()
	log.Println("listen ok")

	var phones []Phone

	var i int
	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Println("accept error:", err)
		}
		conn = net.TCPConn{conn}
		go process_phone_conn(conn, phones)
		log.Printf("%d: accept a new connection\n", i)
	}
}

/**
test the phone conn
*/
func test_phone(address net.TCPAddr, phone Phone) {
	log.Println("test routine start")
	conn, err := net.DialTimeout("tcp", address.String(), 2*time.Second)
	defer conn.Close()
	if err != nil {
		log.Println("dial  error:", err)
	}
	conn.Write("GET /test HTTP/1.1\r\nHOST: anything\r\n\r\n")
	var data []byte
	conn.Read(data)
	log.Println("test routine receive data ", data)
	if data == "Webkey" {
		phone.Redirect = address
	}
	log.Println("it can be redirected to ", address)
	// read or write on conn
}

func process_phone_conn(conn net.TCPConn, phone Phone) {

}
