package main

import "fmt"

var HOST = ""
var PORT = 8000

func main() {
	slice := []byte{'a'}
	slice = slice[1:]
	fmt.Printf("%d", len(slice))

	fmt.Println(HOST,PORT)

	start_phones()

}
