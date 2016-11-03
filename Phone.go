package main

import (
	"log"
	"net"
	"strings"
	"sync"
	"time"
	"os"
	"bufio"
	"errors"
	"fmt"
	"path/filepath"
	"github.com/garyburd/redigo/redis"
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

func (phone *Phone) append_conn(conn net.TCPConn) error {
	phone.mu.Lock()

	err := conn.SetKeepAlive(true)
	if err != nil {
		phone.log_to_file("set keep alive error:", err)
	}

	err = conn.SetKeepAlivePeriod(time.Second * 960)
	if err != nil {
		phone.log_to_file("set keep alive period error:", err)
	}

	phone.Conn_list = append(phone.Conn_list, conn)

	if len(phone.Conn_list) > 4 {
		conn0 := phone.Conn_list[0]
		err = conn0.Close()
		if err != nil {
			phone.log_to_file("close conn error:", err)
		}
		phone.Conn_list = phone.Conn_list[1:]
	}

	address := conn.RemoteAddr()
	ip := strings.Split(address.String(), ":")[0]

	if phone.Last_known != ip {
		phone.log_to_file("not last known ip")
		phone.log_to_file("Last known IP:" + phone.Last_known + ", new:" + ip)
		for len(phone.Conn_list) > 1 {
			conn0 := phone.Conn_list[0]
			err = conn0.Close()
			if err != nil {
				phone.log_to_file("close conn error:", err)
			}
			phone.Conn_list = phone.Conn_list[1:]
		}
		phone.Last_known = ip
		phone.Overheat_count = 0
		phone.Overhead = 0
	}

	phone.log_to_file("append a conn", conn.RemoteAddr().String())
	phone.log_to_file("phone conn list size______", len(phone.Conn_list))

	phone.mu.Unlock()
	return nil
}

func (phone *Phone) get_conn() (conn net.TCPConn, err error) {
	exp_time := time.Now().Add(time.Second * 1)
	phone.mu.Lock()
	for len(phone.Conn_list) == 0 {
		time.Sleep(time.Millisecond * 500)
		if exp_time.Before(time.Now()) {
			break
		}
	}

	if len(phone.Conn_list) == 0 {
		phone.log_to_file("Got signal , bu no connection " + phone.User_name)
		phone.mu.Unlock()
		return net.TCPConn{}, errors.New("no connect")
	}

	conn0 := phone.Conn_list[0]
	phone.Conn_list = phone.Conn_list[1:]

	phone.log_to_file(phone.User_name, "return conn", conn0.RemoteAddr().String())

	phone.mu.Unlock()
	return conn0, nil
}

func (phone *Phone) close_all_conn() error {
	phone.mu.Lock()
	for len(phone.Conn_list) > 0 {
		conn0 := phone.Conn_list[0]
		err := conn0.Close()
		if err != nil {
			phone.log_to_file("close conn error:", err)
		}
		phone.Conn_list = phone.Conn_list[1:]
	}
	phone.mu.Unlock()
	return nil
}

func (phone *Phone) init(user_name string, random string, conn net.TCPConn, address net.TCPAddr) error {
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

func (phone *Phone) add_to_file() error {
	fl, err := os.OpenFile(db_file_name, os.O_CREATE | os.O_APPEND | os.O_RDWR, 0660)
	defer fl.Close()
	if (err != nil) {
		phone.log_to_file("open file error", err)
		return err
	}
	fl.WriteString(phone.User_name + " " + phone.Random + "\n")
	return nil
}

func (phone *Phone) log_to_file(v ...interface{}) error {
	string_date := current_date_string()
	string_time := current_time_string()
	os.MkdirAll("log" + string(filepath.Separator) + string_date, 06660)
	log_file_name := "log" + string(filepath.Separator) + string_date + string(filepath.Separator) + phone.User_name + ".log"
	fl, err := os.OpenFile(log_file_name, os.O_CREATE | os.O_APPEND | os.O_RDWR, 0660)
	defer fl.Close()
	if (err != nil) {
		log.Println("open file error", err)
		return err
	}
	fl.WriteString("[" + string_time + "]" + fmt.Sprintln(v...))
	return nil
}

/**
trans phone  address
 */
func trans_phone_address(address_map string) (string, error) {
	redis_key := fmt.Sprintf("YUNPHONE:DEVICE:MAP:%s", address_map)
	redis_key = strings.ToUpper(redis_key)
	redis_conn, err := getRedisConn()
	defer redis_conn.Close()
	if err != nil {
		log.Println("REDIS CONN ERROR", redis_key, err)
		return "", err;
	}
	device_name, err := redis.String(redis_conn.Do("GET", redis_key))
	if err != nil {
		log.Println("REDIS GET ERROR", redis_key, err)
		return "", err;
	}
	log.Println("REDIS GET ", redis_key, ": ", device_name)
	return device_name, err
}

/**
trans phone  address
 */
func set_phone_ws_state_in_redis(phone_name string, state int) (error) {
	if _, ok := phones[phone_name]; !ok {
		return errors.New("phone not exist")
	}
	redis_key := fmt.Sprintf("YUNPHONE:DEVICE:WS:STATE:%s", phone_name)
	redis_key = strings.ToUpper(redis_key)
	redis_conn, err := getRedisConn()
	defer redis_conn.Close()
	if err != nil {
		phones[phone_name].log_to_file("REDIS CONN ERROR", redis_key, err)
		return err;
	}
	_, err = redis.String(redis_conn.Do("SET", redis_key, state))
	if err != nil {
		phones[phone_name].log_to_file("REDIS SET ERROR", redis_key, err)
		return err;
	}
	phones[phone_name].log_to_file("REDIS SET ", redis_key, ": ", phone_name, ":", state)
	return  err
}

func get_phone_ws_state_in_redis(phone_name string) (int) {
	if _, ok := phones[phone_name]; !ok {
		return 0
	}
	redis_key := fmt.Sprintf("YUNPHONE:DEVICE:WS:STATE:%s", phone_name)
	redis_key = strings.ToUpper(redis_key)
	redis_conn, err := getRedisConn()
	defer redis_conn.Close()
	if err != nil {
		phones[phone_name].log_to_file("REDIS CONN ERROR", redis_key, err)
		return 0;
	}
	state, err := redis.Int(redis_conn.Do("GET", redis_key))
	if err != nil {
		phones[phone_name].log_to_file("REDIS GET ERROR", redis_key, err)
		return 0;
	}
	phones[phone_name].log_to_file("REDIS GET WS State ", redis_key, ": ", state)
	return  state
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

	scanner := bufio.NewScanner(fl)
	for scanner.Scan() {
		line := scanner.Text()
		log.Println(line)
		infos := strings.Split(line, " ")
		if len(infos) == 2 {
			phone := Phone{}
			phone.User_name = infos[0]
			phone.Random = infos[1]
			phones[infos[0]] = &phone
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
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
		phones[user_name] = &phone
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

			if len(user_name) <= 0 || len(random) <= 0 {
				log.Println("user_name or random len = 0")
				conn.Write([]byte("stop"))
				conn.Close()
				return
			}

			_, ok := phones[user_name];

			if ok && (phones[user_name].Random == random) {
				//log.Println(user_name, " phone append a conn", conn.RemoteAddr().String())
				phones[user_name].append_conn(conn)
				return
			} else if !ok {
				phone := Phone{}
				phone.User_name = user_name
				phone.Random = random
				phone.add_to_file()
				phones[user_name] = &phone
				phones[user_name].append_conn(conn)
				log.Println("new phone ", user_name)
			} else {
				log.Println(user_name, random, "stop phone old phone random: ", phones[user_name].Random)
				//conn.Write([]byte("stop"))
				conn.Close()
				log.Println("no thing matched")
				return
			}
		}

	}

}
