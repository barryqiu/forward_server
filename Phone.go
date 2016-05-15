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
	Redirect       bool
	Overhead       int
	Overheat_count int
}

func (phone Phone) append_conn(conn net.TCPConn, address net.TCPAddr) {
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

	if len(phone.Conn_list) > phone.Overhead{
		conn0 :=phone.conn_list[0]
		err = conn0.Close()
		if err != nil {
			log.Println("colse conn error:", err)
		}
		phone.conn_list = phone.conn_list[1:]

		if phone.overheat_count > 100{
			phone.overheat_count = 0
			if phone.overhead > 10{
				phone.overhead = 10
			}else {
				phone.overhead = phone.overhead + 1
			}
		}

	}

	if phone.Last_known != address{

	}


}
