package http_learning

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

func say(s string) {
	for range 5 {
		runtime.Gosched()
		fmt.Println(s)
	}
}

func GoroutineDemo() {
	fmt.Println("开始执行...")

	// 启动新的 Goroutine
	go say("world")

	// 添加一个小延迟，观察是否会影响执行顺序
	time.Sleep(time.Millisecond * 100)

	// 主 Goroutine 执行
	say("hello")

	fmt.Println("程序结束")
}

func BetterGoroutineDemo() {
	var wg sync.WaitGroup
	wg.Add(1) // 等待一个 Goroutine

	go func() {
		defer wg.Done()
		say("world")
	}()

	say("hello")
	wg.Wait() // 等待 Goroutine 完成
}
