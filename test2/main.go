package main

import (
	"fmt"
	"log"

	"github.com/mattn/go-tty"
)

var playerX, playerY int
var gameMap = []string{
	"###################",
	"#                 #",
	"#   #####         #",
	"#        #        #",
	"#   ###  #   ###  #",
	"#   ###  #   ###  #",
	"#                 #",
	"###################",
}

func clearScreen() {
	fmt.Print("\033[H\033[2J")
}

func drawMap() {
	for y, line := range gameMap {
		for x, char := range line {
			if x == playerX && y == playerY {
				fmt.Print("P")
			} else {
				fmt.Print(string(char))
			}
		}
		fmt.Println()
	}
}

func movePlayer(dx, dy int) {
	newX := playerX + dx
	newY := playerY + dy
	if gameMap[newY][newX] == ' ' {
		playerX = newX
		playerY = newY
	}
}

func main() {
	playerX, playerY = 1, 1

	tty, err := tty.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer tty.Close()

	for {
		clearScreen()
		drawMap()

		r, err := tty.ReadRune()
		if err != nil {
			log.Fatal(err)
		}

		switch r {
		case 'w':
			movePlayer(0, -1)
		case 'a':
			movePlayer(-1, 0)
		case 's':
			movePlayer(0, 1)
		case 'd':
			movePlayer(1, 0)
		case 'q': // 'q'キーで終了
			return
		}
	}
}
