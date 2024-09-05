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

// Point 2D座標を表す構造体
type Point struct {
	X, Y int
}

// Player プレイヤーを表す構造体
type Player struct {
	Position Point
}

// Game 迷路ゲームを管理する構造体
type Game struct {
	Map    []string
	Player Player
	Goal   Point
	RNG    *rand.Rand
	Width  int
	Height int
}

// NewGame 新しいゲームインスタンスを作成する
func NewGame(width, height int) *Game {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	player := Player{Position: Point{X: 1, Y: 1}}

	game := &Game{
		Player: player,
		RNG:    rng,
		Width:  width,
		Height: height,
	}

	// ゴールを迷路の右下に設定
	game.Goal = Point{X: width - 2, Y: height - 2}

	return game
}

// getTerminalSize ターミナルのサイズ（列数、行数）を取得する関数
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

// shuffleDirections 上下左右の移動方向をランダムに並べ替えて返す関数
func (g *Game) shuffleDirections() []Point {
	directions := []Point{
		{X: 0, Y: -1}, // 上
		{X: 1, Y: 0},  // 右
		{X: 0, Y: 1},  // 下
		{X: -1, Y: 0}, // 左
	}
	g.RNG.Shuffle(len(directions), func(i, j int) {
		directions[i], directions[j] = directions[j], directions[i]
	})
	return directions
}

// generateMaze 迷路を生成する関数
func (g *Game) generateMaze() {
	maze := make([][]rune, g.Height)
	for i := range maze {
		maze[i] = make([]rune, g.Width)
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

	visited := make([][]bool, g.Height)
	for i := range visited {
		visited[i] = make([]bool, g.Width)
	}

	var carve func(x, y int)
	carve = func(x, y int) {
		visited[y][x] = true
		directions := g.shuffleDirections()
		for _, d := range directions {
			nx, ny := x+d.X*2, y+d.Y*2
			if nx > 0 && ny > 0 && nx < g.Width-1 && ny < g.Height-1 && !visited[ny][nx] {
				if d.X == 0 {
					maze[y+d.Y][x] = ' '
				} else {
					maze[y][x+d.X] = ' '
				}
				carve(nx, ny)
			}
		}
	}

	// スタート位置から迷路を生成開始
	carve(1, 1)

	g.Map = make([]string, g.Height)
	for i := range maze {
		g.Map[i] = string(maze[i])
	}

	// スタート地点とゴール地点を設定
	g.Map[1] = replaceRuneAtIndex(g.Map[1], 1, 'S')                      // スタート
	g.Map[g.Goal.Y] = replaceRuneAtIndex(g.Map[g.Goal.Y], g.Goal.X, 'G') // ゴール
}

// replaceRuneAtIndex 指定した位置のルーンを置き換える関数
func replaceRuneAtIndex(s string, index int, r rune) string {
	runes := []rune(s)
	runes[index] = r
	return string(runes)
}

// clearScreen ターミナルの画面をクリアする関数
func clearScreen() {
	cmd := exec.Command("clear")
	cmd.Stdout = os.Stdout
	cmd.Run()
}

// drawMap 迷路を描画し、プレイヤーの位置を表示する関数
func (g *Game) drawMap() {
	for _, line := range g.Map {
		fmt.Println(line)
	}
	fmt.Printf("\033[%d;%dH", g.Player.Position.Y+1, g.Player.Position.X+1)
	fmt.Print("P")
	fmt.Print("\033[H") // カーソルを再度戻す
}

// movePlayer プレイヤーを指定された方向に移動させる関数
func (g *Game) movePlayer(dx, dy int) {
	newX := g.Player.Position.X + dx
	newY := g.Player.Position.Y + dy

	if newY >= 0 && newY < len(g.Map) && newX >= 0 && newX < len(g.Map[0]) {
		if g.Map[newY][newX] == ' ' || g.Map[newY][newX] == 'G' {
			g.Player.Position.X = newX
			g.Player.Position.Y = newY
		}
	}
}

// checkGoal プレイヤーがゴールに到達したかをチェックする関数
func (g *Game) checkGoal() bool {
	return g.Player.Position.X == g.Goal.X && g.Player.Position.Y == g.Goal.Y
}

// run ゲームを実行する関数
func (g *Game) run() {
	tty, err := tty.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer tty.Close()

	for {
		clearScreen()
		g.drawMap()

		if g.checkGoal() {
			fmt.Println("Congratulations! You've reached the goal!")
			return
		}

		r, err := tty.ReadRune()
		if err != nil {
			log.Fatal(err)
		}

		switch r {
		case 'w':
			g.movePlayer(0, -1)
		case 'a':
			g.movePlayer(-1, 0)
		case 's':
			g.movePlayer(0, 1)
		case 'd':
			g.movePlayer(1, 0)
		case '\033': // 矢印キーのコードはエスケープシーケンスで始まる
			r1, _ := tty.ReadRune()
			r2, _ := tty.ReadRune()
			if r1 == '[' {
				switch r2 {
				case 'A': // 上矢印
					g.movePlayer(0, -1)
				case 'B': // 下矢印
					g.movePlayer(0, 1)
				case 'C': // 右矢印
					g.movePlayer(1, 0)
				case 'D': // 左矢印
					g.movePlayer(-1, 0)
				}
			}
		case 'q': // 'q'キーでゲーム終了
			return
		}
	}
}

func main() {
	width, height, err := getTerminalSize()
	if err != nil {
		log.Fatal(err)
	}

	game := NewGame(width/2, height/2)
	game.generateMaze()
	game.run()
}
