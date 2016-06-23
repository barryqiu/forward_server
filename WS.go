package main

import (
	"flag"
	"github.com/gorilla/websocket"
	"net/http"
	"log"
)

var address = flag.String("addr", "localhost:8001", "http service address")

var upGrader = websocket.Upgrader{} // use default options

func get_screen(w http.ResponseWriter, r *http.Request) {
	log.Println("test ws")
	c, err := upGrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}
		log.Printf("recv: %s", message)
		err = c.WriteMessage(mt, message)
		if err != nil {
			log.Println("write:", err)
			break
		}
	}
}

func start_ws() {
	http.HandleFunc("/", get_screen)
	log.Println("listen web socket 8001 success")
	http.ListenAndServe(*address, nil)
}