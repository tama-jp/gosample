package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"
	"unsafe"

	"github.com/mattn/go-tty"
)

var playerX, playerY int
var gameMap []string

func getTerminalSize() (int, int, error) {
	var sz struct {
		rows uint16
		cols uint16
		x    uint16
		y    uint16
	}
	_, _, err := syscall.Syscall6(syscall.SYS_IOCTL,
		uintptr(syscall.Stdout),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(&sz)),
		0, 0, 0)
	if err != 0 {
		return 0, 0, fmt.Errorf("failed to get terminal size")
	}
	return int(sz.cols), int(sz.rows), nil
}

func initMap(width, height int) {
	// Initialize the map with empty spaces and borders
	gameMap = make([]string, height)
	for i := 0; i < height; i++ {
		if i == 0 || i == height-1 {
			gameMap[i] = "+" + horizontalLine(width-2) + "+"
		} else {
			gameMap[i] = "|" + spaces(width-2) + "|"
		}
	}
}

func horizontalLine(length int) string {
	line := make([]rune, length)
	for i := 0; i < length; i++ {
		line[i] = '-'
	}
	return string(line)
}

func clearScreen() {
	cmd := exec.Command("clear")
	cmd.Stdout = os.Stdout
	cmd.Run()
}

func drawMap() {
	for _, line := range gameMap {
		fmt.Println(line)
	}
	fmt.Print("\033[H") // プレイヤーの位置にカーソルを戻すため
	for y, line := range gameMap {
		for x := range line {
			if x == playerX && y == playerY {
				fmt.Print("\033[" + fmt.Sprintf("%d;%d", y+1, x+1) + "H")
				fmt.Print("P")
				break
			}
		}
	}
	fmt.Print("\033[H") // カーソルを再度戻す
}

func spaces(n int) string {
	return string(make([]rune, n))
}

func movePlayer(dx, dy int) {
	newX := playerX + dx
	newY := playerY + dy
	if newY > 0 && newY < len(gameMap)-1 && newX > 0 && newX < len(gameMap[0])-1 {
		playerX = newX
		playerY = newY
	}
}

func main() {
	// ターミナルサイズを取得
	width, height, err := getTerminalSize()
	if err != nil {
		log.Fatal(err)
	}

	// 画面の半分のサイズに設定
	width /= 2
	height /= 2

	playerX, playerY = 1, 1
	initMap(width, height)

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
