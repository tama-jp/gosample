package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

func main() {
	clearScreen()

	// 画面サイズを仮定（20行）
	screenHeight := 20

	// 初期時刻表示
	go func() {
		for {
			moveCursor(10, screenHeight-2) // 時刻表示位置（最下行から2行上）
			fmt.Print(time.Now().Format("15:04:05"))
			time.Sleep(1 * time.Second)
		}
	}()

	// キーボード入力処理
	originalState := enableRawMode()
	defer disableRawMode(originalState)

	reader := bufio.NewReader(os.Stdin)
	for {
		// キー入力を待機
		char, _ := reader.ReadByte()

		if char == 3 { // Ctrl+C
			// Ctrl+Cで終了
			break
		}

		if char == 'a' {
			// "a"キーが押されたらコマンド入力モードに入る
			moveCursor(1, screenHeight) // コマンド入力位置（最下行）
			clearLine()                 // 入力行をクリア
			fmt.Print("Enter command: ")

			// コマンドを読み取る
			command, _ := reader.ReadString('\n')
			command = strings.TrimSpace(command)

			// コマンドボックスを閉じる（行をクリア）
			clearLine()

			// コマンド結果を表示
			moveCursor(10, screenHeight-3) // 時刻の上の行にカーソルを移動
			clearLine()                    // 前の結果をクリア
			fmt.Printf("You entered: %s", command)

			// 時刻を再描画するためにカーソルを戻す
			moveCursor(10, screenHeight-2)
			fmt.Print(time.Now().Format("15:04:05"))
		}
	}
}

// 画面をクリアする
func clearScreen() {
	cmd := exec.Command("clear")
	cmd.Stdout = os.Stdout
	cmd.Run()
}

// カーソルを指定した位置に移動する
func moveCursor(x, y int) {
	fmt.Printf("\033[%d;%dH", y, x)
}

// カーソルのある行をクリアする
func clearLine() {
	fmt.Print("\033[2K")
}

// 端末をRawモードに切り替える（入力をリアルタイムで取得するため）
func enableRawMode() *syscall.Termios {
	fd := int(os.Stdin.Fd())
	// 現在の端末設定を取得
	var originalState syscall.Termios
	_, _, _ = syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), syscall.TIOCGETA, uintptr(unsafe.Pointer(&originalState)), 0, 0, 0)

	// Rawモード用に設定を変更
	rawState := originalState
	rawState.Lflag &^= syscall.ICANON | syscall.ECHO
	rawState.Iflag &^= syscall.ICRNL | syscall.INLCR | syscall.IGNCR
	rawState.Oflag &^= syscall.OPOST

	// 新しい設定を適用
	_, _, _ = syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), syscall.TIOCSETA, uintptr(unsafe.Pointer(&rawState)), 0, 0, 0)

	return &originalState
}

// 端末を通常モードに戻す
func disableRawMode(originalState *syscall.Termios) {
	fd := int(os.Stdin.Fd())
	_, _, _ = syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), syscall.TIOCSETA, uintptr(unsafe.Pointer(originalState)), 0, 0, 0)
}
