package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ygdcz/golang-learning/src/music/library"
	"github.com/ygdcz/golang-learning/src/music/mp"
)

var lib *library.MusicManager
var id int = 1

func handleLibCommands(tokens []string) {
	switch tokens[1] {
	case "list":
		for i := range lib.Len() {
			music, _ := lib.Get(i)
			fmt.Println(i+1, ":", music.Name, music.Artist, music.Source, music.Type)
		}
	case "add":
		if len(tokens) == 6 {
			id++
			lib.Add(&library.MusicEntry{
				Id:     strconv.Itoa(id),
				Name:   tokens[2],
				Artist: tokens[3],
				Source: tokens[4],
				Type:   tokens[5],
			})
		} else {
			fmt.Println("USAGE: lib add <name> <artist> <source> <type>")
		}
	case "remove":
		if len(tokens) == 3 {
			id, err := strconv.Atoi(tokens[2])
			if err != nil {
				fmt.Println("Invalid ID")
				return
			}
			lib.Remove(id - 1)
		} else {
			fmt.Println("USAGE: lib remove <id>")
		}
	default:
		fmt.Println("Unrecognized lib command:", tokens[1])
	}
}

func handlePlayerCommands(tokens []string) {
	if len(tokens) != 2 {
		fmt.Println("USAGE: player <name>")
		return
	}
	music, err := lib.Find(tokens[1])
	if err != nil {
		fmt.Println("The music", tokens[1], "does not exist.")
		return
	}

	mp.Play(music.Source, music.Type)
}

func main() {
	//打印操作菜单
	fmt.Println(`
		Enter following commands to control the player:
		lib list -- View the existing music lib
		lib add <name><artist><source><type> -- Add a music to the music lib
		lib remove <name> -- Remove the specified music from the lib
		play <name> -- Play the specified music
	`)
	lib = library.NewMusicManager()

	r := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("Enter Command-> ")

		rawLine, _, _ := r.ReadLine()
		line := string(rawLine)
		if line == "q" || line == "e" {
			break
		}

		tokens := strings.Split(line, " ")
		if tokens[0] == "lib" {
			handleLibCommands(tokens)
		} else if tokens[0] == "play" {
			handlePlayerCommands(tokens)
		} else {
			fmt.Println("Unrecognized command:", tokens[0])
		}

	}
}
