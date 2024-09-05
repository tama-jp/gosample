package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
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
	_, _, errno := syscall.Syscall6(syscall.SYS_IOCTL,
		uintptr(syscall.Stdout),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(&sz)),
		0, 0, 0)
	if errno != 0 {
		return 0, 0, fmt.Errorf("failed to get terminal size: %v", errno)
	}
	return int(sz.cols), int(sz.rows), nil
}

func initMap(width, height int) {
	// Base map pattern
	baseMap := []string{
		"+-------------------+",
		"|   |               |",
		"|   |---+  +---+    |",
		"|       |     |     |",
		"|   +---+     +---+ |",
		"|   |             | |",
		"|   +---+  +---+  | |",
		"|       |  |      | |",
		"|  +----+  +------+ |",
		"|                 | |",
		"+-------------------+",
	}

	baseWidth := len(baseMap[0])
	baseHeight := len(baseMap)

	// マップを中央に配置
	xOffset := (width - baseWidth) / 2
	yOffset := (height - baseHeight) / 2

	gameMap = make([]string, height)
	for i := range gameMap {
		if i < yOffset || i >= yOffset+baseHeight {
			gameMap[i] = strings.Repeat(" ", width)
		} else {
			line := baseMap[i-yOffset]
			gameMap[i] = strings.Repeat(" ", xOffset) + line + strings.Repeat(" ", width-len(line)-xOffset)
		}
	}
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
	// プレイヤーの位置にカーソルを移動してから描画
	fmt.Printf("\033[%d;%dH", playerY, playerX+1)
	fmt.Print("P")
	fmt.Print("\033[H") // カーソルを再度戻す
}

func movePlayer(dx, dy int) {
	newX := playerX + dx
	newY := playerY + dy

	// 移動先がマップの範囲内かつ壁でないことを確認
	if newY >= 0 && newY < len(gameMap) && newX >= 0 && newX < len(gameMap[0]) {
		if gameMap[newY][newX] == ' ' {
			playerX = newX
			playerY = newY
		}
	}
}

func main() {
	width, height, err := getTerminalSize()
	if err != nil {
		log.Fatal(err)
	}

	// マップをターミナルのサイズに基づいて初期化
	initMap(width, height)

	// プレイヤーの初期位置をマップの中央付近に設定
	playerX = width/2 - 10 + 4
	playerY = height/2 - 5 + 1

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
