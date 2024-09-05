package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"time"

	"github.com/mattn/go-tty"
)

const (
	width  = 10
	height = 20
)

var shapes = [][][]int{
	// I
	{
		{1, 1, 1, 1},
	},
	// O
	{
		{1, 1},
		{1, 1},
	},
	// T
	{
		{0, 1, 0},
		{1, 1, 1},
	},
	// L
	{
		{1, 0, 0},
		{1, 1, 1},
	},
	// J
	{
		{0, 0, 1},
		{1, 1, 1},
	},
	// Z
	{
		{1, 1, 0},
		{0, 1, 1},
	},
	// S
	{
		{0, 1, 1},
		{1, 1, 0},
	},
}

type Game struct {
	Field     [][]int
	Shape     [][]int
	NextShape [][]int
	PosX      int
	PosY      int
	GameOver  bool
	Input     chan rune
}

func NewGame() *Game {
	field := make([][]int, height)
	for i := range field {
		field[i] = make([]int, width)
	}
	return &Game{
		Field:     field,
		Shape:     getRandomShape(),
		NextShape: getRandomShape(),
		PosX:      width/2 - 1,
		PosY:      0,
		Input:     make(chan rune),
	}
}

func getRandomShape() [][]int {
	return shapes[rand.Intn(len(shapes))]
}

func (g *Game) rotateShape() {
	newShape := make([][]int, len(g.Shape[0]))
	for i := range newShape {
		newShape[i] = make([]int, len(g.Shape))
		for j := range newShape[i] {
			newShape[i][j] = g.Shape[len(g.Shape)-j-1][i]
		}
	}

	if !g.isCollision(newShape, g.PosX, g.PosY) {
		g.Shape = newShape
	}
}

func (g *Game) isCollision(shape [][]int, offsetX, offsetY int) bool {
	for y, row := range shape {
		for x, cell := range row {
			if cell == 0 {
				continue
			}
			newX := offsetX + x
			newY := offsetY + y
			if newX < 0 || newX >= width || newY >= height || (newY >= 0 && g.Field[newY][newX] != 0) {
				return true
			}
		}
	}
	return false
}

func (g *Game) mergeShape() {
	for y, row := range g.Shape {
		for x, cell := range row {
			if cell != 0 {
				g.Field[g.PosY+y][g.PosX+x] = cell
			}
		}
	}
}

func (g *Game) clearLines() {
	newField := make([][]int, height)
	newRow := height - 1

	for y := height - 1; y >= 0; y-- {
		fullLine := true
		for x := 0; x < width; x++ {
			if g.Field[y][x] == 0 {
				fullLine = false
				break
			}
		}
		if !fullLine {
			newField[newRow] = g.Field[y]
			newRow--
		}
	}

	for i := 0; i <= newRow; i++ {
		newField[i] = make([]int, width)
	}

	g.Field = newField
}

func (g *Game) dropShape() {
	g.PosY++
	if g.isCollision(g.Shape, g.PosX, g.PosY) {
		g.PosY--
		g.mergeShape()
		g.clearLines()
		g.Shape = g.NextShape
		g.NextShape = getRandomShape()
		g.PosX = width/2 - 1
		g.PosY = 0
		if g.isCollision(g.Shape, g.PosX, g.PosY) {
			g.GameOver = true
		}
	}
}

func (g *Game) moveShape(dx int) {
	if !g.isCollision(g.Shape, g.PosX+dx, g.PosY) {
		g.PosX += dx
	}
}

func clearScreen() {
	cmd := exec.Command("clear")
	cmd.Stdout = os.Stdout
	cmd.Run()
}

func (g *Game) draw() {
	clearScreen()
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if g.Field[y][x] != 0 {
				fmt.Print("#")
			} else {
				fmt.Print(".")
			}
		}
		fmt.Println()
	}

	for y, row := range g.Shape {
		for x, cell := range row {
			if cell != 0 && g.PosY+y >= 0 {
				fmt.Printf("\033[%d;%dH#", g.PosY+y+1, g.PosX+x+1)
			}
		}
	}
	// 次のテトリミノを表示
	fmt.Println("\nNext:")
	for _, row := range g.NextShape { // yの代わりに空配列で代用
		fmt.Print(" ")
		for _, cell := range row {
			if cell != 0 {
				fmt.Print("#")
			} else {
				fmt.Print(" ")
			}
		}
		fmt.Println()
	}

	fmt.Printf("\033[%d;%dH", height+1, 0)
}

func (g *Game) run() {
	go func() {
		tty, err := tty.Open()
		if err != nil {
			log.Fatal(err)
		}
		defer tty.Close()

		for {
			r, err := tty.ReadRune()
			if err != nil {
				log.Fatal(err)
			}
			g.Input <- r
		}
	}()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for !g.GameOver {
		select {
		case <-ticker.C:
			g.dropShape()
		case r := <-g.Input:
			switch r {
			case 'w':
				g.rotateShape()
			case 'a':
				g.moveShape(-1)
			case 'd':
				g.moveShape(1)
			case 's':
				g.dropShape()
			}
		}
		g.draw()
	}
	fmt.Println("Game Over!")
}

func main() {
	rand.Seed(time.Now().UnixNano())
	game := NewGame()
	game.run()
}
