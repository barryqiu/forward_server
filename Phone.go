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
	"errors"
)

type Phone struct {
	mu             sync.Mutex
	Conn_list      []net.TCPConn
	User_name      string
	Random         string
	Last_known     string
	Overhead       int
	Overheat_count int
}

func (phone Phone) append_conn(conn net.TCPConn) error {
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

	if len(phone.Conn_list) > 3 + phone.Overhead {
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

	address := conn.RemoteAddr()
	ip := strings.Split(address.String(), ":")[0]

	if phone.Last_known != ip {
		log.Println("not last known ip")
		log.Println("Last known IP:" + phone.Last_known + ", new:" + ip)
		for len(phone.Conn_list) > 1 {
			conn0 := phone.Conn_list[0]
			err = conn0.Close()
			if err != nil {
				log.Println("close conn error:", err)
			}
			phone.Conn_list = phone.Conn_list[1:]
		}
		phone.Last_known = ip
		phone.Overheat_count = 0
		phone.Overhead = 0
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
		return net.TCPConn{}, errors.New("no connect")
	}

	conn0 := phone.Conn_list[0]
	phone.Conn_list = phone.Conn_list[1:]

	log.Println(phone.User_name, "return conn", conn0.RemoteAddr().String())

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
	phone.Last_known = ""
	phone.Overhead = 0
	phone.Overheat_count = 0
	if conn == (net.TCPConn{}) {
		phone.append_conn(conn)
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
		log.Println(line)
		infos := strings.Split(line, " ")
		if len(infos) == 2 {
			phone := Phone{}
			phone.User_name = infos[0]
			phone.Random = infos[1]
			phones[infos[0]] = phone
		}
	}
}

/**
start phone thread to listen request from yun phone
*/
func start_phones() {

	read_phones_from_file()

	add, err := net.ResolveTCPAddr("tcp", ":110")
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
	log.Println("listen 110 ok")

	for {
		conn, err := listen.AcceptTCP()
		if err != nil {
			log.Println("accept error:", err)
		}
		go process_phone_conn(*conn)
		log.Println("accept a new phone connection")
	}
}

func process_phone_conn(conn net.TCPConn) {
	if (net.TCPConn{}) == conn {
		return
	}
	var buf = make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		log.Println("phone conn read error:", err)
		return
	}
	content := string(buf[:n])

	log.Println("phone send :", content)

	pos := strings.Index(content, "/")
	if pos == -1 {
		conn.Close()
		return
	}

	if strings.HasPrefix(content, "GET /register_") {
		req, err := getRequestInfo(content)
		if err != nil {
			log.Println("phone conn read error:", err)
			return
		}
		infos := strings.Split(req.RequestURI, "/")
		user_name := infos[2]
		random := infos[3]
		version := infos[4]
		log.Println("reg:user_name:" + user_name + ";random:" + random + ";version:" + version)

		if len(user_name) == 0 {
			conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\nEmpty username is not allowed."))
			conn.Close()
			return
		}

		if _, ok := phones[user_name]; ok && phones[user_name].Random != random {
			conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\nUsername is already exists."))
			conn.Close()
			return
		}

		phone := Phone{}
		phone.User_name = user_name
		phone.Random = random
		phone.add_to_file()
		phones[user_name] = phone
		conn.Write([]byte("HTTP/1.1 200 OK\r\n\r\nOK"))
		conn.Close()
		return
	}

	if strings.HasPrefix(content, "WEBKEY") {
		// WEBKEY username/random/version/port
		// WEBKEY username/random/1.0/888
		lines := strings.Split(content, "/r/n")
		if len(lines) > 0 {
			first_line := lines[0]
			p1 := strings.Index(first_line, "/")
			user_name := first_line[7:p1]
			p2 := strings.Index(first_line[p1 + 1:], "/") + p1 + 1
			random := first_line[p1 + 1 : p2]
			p3 := strings.Index(first_line[p2 + 1:], "/") + p2 + 1
			version := first_line[p2 + 1 : p3]
			port := first_line[p3 + 1:]
			log.Println("WEBKEY:", user_name, random, version, port)

			if len(user_name) <= 0 || len(random) <= 0 {
				log.Println("user_name or random len = 0")
				conn.Write([]byte("stop"))
				conn.Close()
				return
			}

			_, ok := phones[user_name];
			log.Println("ok:", ok)
			log.Println("random:", phones[user_name].Random == random)
			log.Printf("two random %v,%v\n", phones[user_name].Random, random)
			log.Println("equal fold ",strings.EqualFold(phones[user_name].Random, random))

			if ok && (phones[user_name].Random == random) {
				log.Println(user_name, " phone append a conn", conn.RemoteAddr().String())
				phones[user_name].append_conn(conn)
				return
			} else if !ok {
				phone := Phone{}
				phone.User_name = user_name
				phone.Random = random
				phone.add_to_file()
				phones[user_name] = phone
				phones[user_name].append_conn(conn)
				log.Println("new phone ", user_name)
			} else {
				log.Println("stop phone old phone random: ", phones[user_name].Random)
				//conn.Write([]byte("stop"))
				conn.Close()
				log.Println("no thing matched")
				return
			}
		}

	}

}
