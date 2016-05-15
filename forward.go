package main

import "fmt"

func main()  {
	slice := []byte {'a', 'b', 'c', 'd'}
	slice = slice[1:]
	fmt.Printf("%s", slice)
}