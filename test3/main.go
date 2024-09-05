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
		case 'w', 'W', 'A', 'a', 's', 'S', 'd', 'D':
			switch r {
			case 'w', 'W':
				movePlayer(0, -1)
			case 'a', 'A':
				movePlayer(-1, 0)
			case 's', 'S':
				movePlayer(0, 1)
			case 'd', 'D':
				movePlayer(1, 0)
			}
		case '\033': // 矢印キーのコードはエスケープシーケンスで始まります
			r1, _ := tty.ReadRune()
			r2, _ := tty.ReadRune()
			if r1 == '[' {
				switch r2 {
				case 'A': // 上矢印
					movePlayer(0, -1)
				case 'B': // 下矢印
					movePlayer(0, 1)
				case 'C': // 右矢印
					movePlayer(1, 0)
				case 'D': // 左矢印
					movePlayer(-1, 0)
				}
			}
		case 'q': // 'q'キーで終了
			return
		}
	}
}
