package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ygdcz/golang-learning/src/cgss/cg"
	"github.com/ygdcz/golang-learning/src/cgss/ipc"
)

var centerClient *cg.CenterClient

func startCenterService() error {
	server := ipc.NewIpcServer(cg.NewCenterServer())
	client := ipc.NewIpcClient(server)
	centerClient = &cg.CenterClient{IpcClient: client}

	return nil
}

func Help(args []string) int {
	fmt.Println(`
		Commands:
			login <username><level><exp>
			logout <username>
			send  <message>
			listplayer
			quit(q)
			help(h)
		}
	`)
	return 0
}

func Quit(args []string) int {
	return 1
}

func Logout(args []string) int {
	if len(args) != 2 {
		fmt.Println("Usage: logout <username>")
		return 0
	}

	centerClient.RemovePlayer(args[1])
	return 0
}

func Login(args []string) int {
	if len(args) != 4 {
		fmt.Println("Usage: login <username><level><exp>")
		return 0
	}
	level, err := strconv.Atoi(args[2])
	if err != nil {
		fmt.Println("Invalid Parameter: <level> should be an interger.")
		return 0
	}

	exp, err := strconv.Atoi(args[3])
	if err != nil {
		fmt.Println("Invalid Parameter: <exp> should be an interger.")
		return 0
	}

	player := cg.NewPlayer()
	player.Name = args[1]
	player.Level = level
	player.Exp = exp

	err = centerClient.AddPlayer(player)
	if err != nil {
		fmt.Println("Add Player Failed:", err)
		return 0
	}
	return 0
}

func ListPlayer(args []string) int {
	ps, err := centerClient.ListPlayer()
	if err != nil {
		fmt.Println("List Player Failed:", err)
		return 0
	}

	for _, p := range ps {
		fmt.Println(p)
	}
	return 0
}

func Send(args []string) int {
	message := strings.Join(args[1:], " ")

	err := centerClient.Broadcast(message)

	if err != nil {
		fmt.Println("Send Message Failed:", err)
	}
	return 0
}

func getCommandHandlers() map[string]func([]string) int {
	return map[string]func([]string) int{
		"list":   ListPlayer,
		"send":   Send,
		"help":   Help,
		"quit":   Quit,
		"q":      Quit,
		"login":  Login,
		"logout": Logout,
	}
}

func main() {
	fmt.Println("Casual Game Server Solution")

	startCenterService()

	Help(nil)

	r := bufio.NewReader(os.Stdin)

	handlers := getCommandHandlers()

	for {
		fmt.Print("Enter Command-> ")
		b, _, _ := r.ReadLine()
		line := string(b)

		tokens := strings.Split(line, " ")

		if handler, ok := handlers[tokens[0]]; ok {
			ret := handler(tokens)
			if ret != 0 {
				break
			}
		} else {
			fmt.Println("Unrecognized command:", tokens[0])
		}
	}
}
