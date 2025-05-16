package cg

import "fmt"

type Player struct {
	Name  string `json:"name"`
	Level int    `json:"level"`
	Exp   int    `json:"exp"`
	Room  int    `json:"room"`

	mq chan *Message // wait for message
}

// 为每个玩家都起了一个独立的goroutine，监听所有发送给他们的聊天信息，一旦收到就即时打印到控制台上。
func NewPlayer() *Player {
	m := make(chan *Message, 1024)
	player := &Player{"", 0, 0, 0, m}

	go func(p *Player) {
		for {
			msg := <-p.mq
			fmt.Println(p.Name, "received", msg.Content)
		}
	}(player)

	return player
}
