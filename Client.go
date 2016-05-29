package main

import (
	"log"
	"net/http"
)
// 策略:
// 使用一个全局的slice数组存储所有的Phone
// 接受客户端请求的部分使用GO语言的HTTP来处理,在这部分利用全局变量访问云端设备的链接
// 接受云端设备请求的部分使用Go的socket编程处理,同样使用全局变量来操作云端设备对象,实现添加链接等操作


func sayHelloName(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("asdddssd"))
}

func main() {
	http.HandleFunc("/", sayHelloName) //设置访问的路由
	err := http.ListenAndServe(":9090", nil) //设置监听的端口
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

