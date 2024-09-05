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

// プレイヤーの現在位置を保持するための変数
var playerX, playerY int

// 迷路のマップデータを保持するスライス
var gameMap []string

// 2D座標を表すための構造体。迷路の生成やプレイヤーの移動に使用される。
type Point struct {
	X, Y int
}

// ターミナルのサイズ（列数、行数）を取得する関数
func getTerminalSize() (int, int, error) {
	var sz struct {
		rows uint16
		cols uint16
		x    uint16
		y    uint16
	}
	// システムコールを使用してターミナルのサイズを取得
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

// 上下左右の移動方向をランダムに並べ替えて返す関数
func shuffleDirections(rng *rand.Rand) []Point {
	directions := []Point{
		{X: 0, Y: -1}, // 上
		{X: 1, Y: 0},  // 右
		{X: 0, Y: 1},  // 下
		{X: -1, Y: 0}, // 左
	}
	// 移動方向をランダムにシャッフル
	rng.Shuffle(len(directions), func(i, j int) {
		directions[i], directions[j] = directions[j], directions[i]
	})
	return directions
}

// 与えられた幅と高さで迷路を生成する関数
func generateMaze(width, height int, rng *rand.Rand) {
	// 迷路データを初期化
	maze := make([][]rune, height)
	for i := range maze {
		maze[i] = make([]rune, width)
		for j := range maze[i] {
			if i%2 == 0 && j%2 == 0 {
				maze[i][j] = '+'
			} else if i%2 == 0 {
				maze[i][j] = '-'
			} else if j%2 == 0 {
				maze[i][j] = '|'
			} else {
				maze[i][j] = ' '
			}
		}
	}

	// 訪問済みのセルを追跡するための2次元スライス
	visited := make([][]bool, height)
	for i := range visited {
		visited[i] = make([]bool, width)
	}

	// 再帰的に通路を掘り進める関数
	var carve func(x, y int)
	carve = func(x, y int) {
		visited[y][x] = true
		directions := shuffleDirections(rng)
		for _, d := range directions {
			nx, ny := x+d.X*2, y+d.Y*2
			if nx > 0 && ny > 0 && nx < width-1 && ny < height-1 && !visited[ny][nx] {
				if d.X == 0 {
					maze[y+d.Y][x] = ' '
				} else {
					maze[y][x+d.X] = ' '
				}
				carve(nx, ny)
			}
		}
	}

	// ランダムな位置から迷路を生成開始
	carve(1, 1)

	// 生成された迷路をgameMapに変換
	gameMap = make([]string, height)
	for i := range maze {
		gameMap[i] = string(maze[i])
	}

	// プレイヤーの初期位置を設定
	playerX, playerY = 1, 1
}

// ターミナルの画面をクリアする関数
func clearScreen() {
	cmd := exec.Command("clear")
	cmd.Stdout = os.Stdout
	cmd.Run()
}

// 迷路を描画し、プレイヤーの位置を表示する関数
func drawMap() {
	for _, line := range gameMap {
		fmt.Println(line)
	}
	// プレイヤーの位置にカーソルを移動して描画
	fmt.Printf("\033[%d;%dH", playerY+1, playerX+1)
	fmt.Print("P")
	fmt.Print("\033[H") // カーソルを再度戻す
}

// プレイヤーを指定された方向に移動させる関数
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

// プログラムのエントリーポイント
func main() {
	// 個別の乱数生成器を初期化
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// ターミナルサイズに基づいて迷路を生成
	width, height, err := getTerminalSize()
	if err != nil {
		log.Fatal(err)
	}

	generateMaze(width/2, height/2, rng)

	// ttyを開いてユーザー入力を監視
	tty, err := tty.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer tty.Close()

	for {
		clearScreen() // 画面をクリア
		drawMap()     // 迷路とプレイヤーを描画

		// キー入力を読み取る
		r, err := tty.ReadRune()
		if err != nil {
			log.Fatal(err)
		}

		// キー入力に応じてプレイヤーを移動させる
		switch r {
		case 'w':
			movePlayer(0, -1)
		case 'a':
			movePlayer(-1, 0)
		case 's':
			movePlayer(0, 1)
		case 'd':
			movePlayer(1, 0)
		case '\033': // 矢印キーのコードはエスケープシーケンスで始まる
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
		case 'q': // 'q'キーでゲーム終了
			return
		}
	}
}
