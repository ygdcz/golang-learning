package cg

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ygdcz/golang-learning/src/cgss/ipc"
)

type CenterClient struct {
	*ipc.IpcClient
}

func (client *CenterClient) AddPlayer(player *Player) error {
	b, err := json.Marshal(*player)

	if err != nil {
		return err
	}

	resp, err := client.Call("addPlayer", string(b))

	if err == nil && resp.Code == "200" {
		return nil
	}
	return err
}

func (client *CenterClient) RemovePlayer(name string) error {
	resp, err := client.Call("removePlayer", name)

	if err == nil && resp.Code == "200" {
		return nil
	}
	return errors.New(resp.Body)
}

func (client *CenterClient) ListPlayer() (ps []*Player, err error) {
	resp, err := client.Call("listPlayer", "")
	if err != nil {
		err = errors.New(resp.Body)
		return
	}

	if err = json.Unmarshal([]byte(resp.Body), &ps); err != nil {
		err = errors.New(resp.Body)
		return
	}
	return
}

func (client *CenterClient) Broadcast(message string) error {
	m := &Message{Content: message}

	b, err := json.Marshal(m)

	if err != nil {
		return err
	}

	resp, _ := client.Call("broadcast", string(b))

	fmt.Println(resp.Code, resp.Code == "200")

	if resp.Code == "200" {
		return nil
	}
	return errors.New(resp.Body)
}
