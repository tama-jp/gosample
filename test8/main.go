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

type Point struct {
	X, Y int
}

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

func shuffleDirections() []Point {
	directions := []Point{
		{X: 0, Y: -1}, // 上
		{X: 1, Y: 0},  // 右
		{X: 0, Y: 1},  // 下
		{X: -1, Y: 0}, // 左
	}
	rand.Shuffle(len(directions), func(i, j int) {
		directions[i], directions[j] = directions[j], directions[i]
	})
	return directions
}

func generateMaze(width, height int) {
	maze := make([][]rune, height)
	for i := range maze {
		maze[i] = make([]rune, width)
		for j := range maze[i] {
			maze[i][j] = '+'
		}
	}

	var carve func(x, y int)
	carve = func(x, y int) {
		maze[y][x] = ' '
		for _, d := range shuffleDirections() {
			nx, ny := x+d.X*2, y+d.Y*2
			if nx > 0 && ny > 0 && nx < width-1 && ny < height-1 && maze[ny][nx] == '+' {
				maze[ny-d.Y][nx-d.X] = ' '
				carve(nx, ny)
			}
		}
	}

	// ランダムな位置から迷路を生成
	carve(1, 1)

	gameMap = make([]string, height)
	for i := range maze {
		gameMap[i] = string(maze[i])
	}

	// プレイヤーの初期位置を設定
	playerX, playerY = 1, 1
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
	rand.Seed(time.Now().UnixNano())
	width, height, err := getTerminalSize()
	if err != nil {
		log.Fatal(err)
	}

	// 迷路を生成
	generateMaze(width/2, height/2)

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
