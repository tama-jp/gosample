package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"syscall"
	"time"
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

func generateRandomMap(width, height int) {
	rand.Seed(time.Now().UnixNano())

	// 周囲を壁で囲む
	gameMap = make([]string, height)
	for i := 0; i < height; i++ {
		line := ""
		for j := 0; j < width; j++ {
			if i == 0 || i == height-1 || j == 0 || j == width-1 {
				line += "+"
			} else if rand.Intn(5) == 0 { // 20%の確率で壁を配置
				if rand.Intn(2) == 0 {
					line += "-"
				} else {
					line += "|"
				}
			} else {
				line += " "
			}
		}
		gameMap[i] = line
	}

	// プレイヤーの初期位置を空いている場所に設定
	for {
		playerX = rand.Intn(width-2) + 1
		playerY = rand.Intn(height-2) + 1
		if gameMap[playerY][playerX] == ' ' {
			break
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
	fmt.Printf("\033[%d;%dH", playerY+1, playerX+1)
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

	// ランダムなマップを生成
	generateRandomMap(width/2, height/2)

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
