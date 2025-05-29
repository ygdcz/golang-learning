package cg

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sync"

	"github.com/ygdcz/golang-learning/src/cgss/ipc"
)

var _ ipc.Server = &CenterServer{}

type Message struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Content string `json:"content"`
}

type CenterServer struct {
	servers map[string]ipc.Server
	players []*Player
	// rooms   []*Room
	mutex sync.RWMutex
}

func NewCenterServer() *CenterServer {
	return &CenterServer{
		servers: make(map[string]ipc.Server),
		players: make([]*Player, 0),
	}
}

func (server *CenterServer) addPlayer(params string) error {
	player := NewPlayer()

	err := json.Unmarshal([]byte(params), &player)
	if err != nil {
		return err
	}
	server.mutex.Lock()
	defer server.mutex.Unlock()
	server.players = append(server.players, player)
	return nil
}

func (server *CenterServer) removePlayer(params string) error {
	server.mutex.Lock()
	defer server.mutex.Unlock()

	for i, p := range server.players {
		if p.Name == params {
			if len(server.players) == 1 {
				server.players = make([]*Player, 0)
			}
			server.players = slices.Delete(server.players, i, i+1)
			p.mq <- &Message{
				Content: fmt.Sprintf("%s 离开了", params),
				From:    "center",
				To:      "all",
			}
			return nil
		}
	}
	return errors.New("player not found")
}

func (server *CenterServer) listPlayer() (players string, err error) {
	server.mutex.RLock()
	defer server.mutex.RUnlock()

	if len(server.players) > 0 {
		b, _ := json.Marshal(server.players)
		players = string(b)
	} else {
		err = errors.New("no player online")
	}
	return
}

func (server *CenterServer) broadcast(params string) error {
	var message Message
	err := json.Unmarshal([]byte(params), &message)
	if err != nil {
		return err
	}

	server.mutex.Lock()
	defer server.mutex.Unlock()

	if len(server.players) > 0 {
		for _, player := range server.players {
			player.mq <- &message
		}
	} else {
		err = errors.New("no player online")
	}
	return err
}

func (server *CenterServer) Handle(method, params string) *ipc.Response {
	switch method {
	case "addPlayer":
		err := server.addPlayer(params)
		if err != nil {
			return &ipc.Response{Code: "500", Body: err.Error()}
		}
		return &ipc.Response{Code: "200", Body: "ok"}
	case "removePlayer":
		err := server.removePlayer(params)
		if err != nil {
			return &ipc.Response{Code: "500", Body: err.Error()}
		}
	case "listPlayer":
		players, err := server.listPlayer()
		if err != nil {
			return &ipc.Response{Code: "500", Body: err.Error()}
		}
		return &ipc.Response{Code: "200", Body: players}
	case "broadcast":
		err := server.broadcast(params)
		if err != nil {
			return &ipc.Response{Code: "500", Body: err.Error()}
		}
		return &ipc.Response{Code: "200", Body: "ok"}
	default:
		return &ipc.Response{Code: "404", Body: method + ":" + params + " not found"}
	}
	return &ipc.Response{Code: "200", Body: "ok"}
}

func (server *CenterServer) Name() string {
	return "CenterServer"
}
