package main

import (
	"sync"
)

const NCPU = 4

var a string
var once sync.Once

func setup() {
	a = "hello, world"
}

func doprint(c chan int) {
	once.Do(func() {
		setup()
		print(a)
	})

	c <- 1
}

func twoprint(c chan int) {
	go doprint(c)
	go doprint(c)
}

func main() {
	c := make(chan int, 2)
	defer close(c)
	// fmt.Println(runtime.NumCPU())
	twoprint(c)
	// c := make(chan int, NCPU) // 创建带缓冲的通道
	for i := 0; i < 2; i++ {
		<-c
	}
}
