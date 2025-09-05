//go:build windows
// +build windows

package input

import (
	"os"
	"syscall"
	"unsafe"
)

var (
	kernel32                       = syscall.NewLazyDLL("kernel32.dll")
	procGetConsoleScreenBufferInfo = kernel32.NewProc("GetConsoleScreenBufferInfo")
)

type consoleScreenBufferInfo struct {
	Size              coord
	CursorPosition    coord
	Attributes        uint16
	Window            smallRect
	MaximumWindowSize coord
}

type coord struct {
	X int16
	Y int16
}

type smallRect struct {
	Left   int16
	Top    int16
	Right  int16
	Bottom int16
}

// 端末サイズを取得 (Windows専用)
func getTerminalSize() (int, int) {
	// 標準出力のハンドルを取得
	handle := os.Stdout.Fd()

	var csbi consoleScreenBufferInfo
	ret, _, _ := procGetConsoleScreenBufferInfo.Call(
		uintptr(handle),
		uintptr(unsafe.Pointer(&csbi)),
	)

	if ret == 0 {
		// エラーの場合はデフォルト値を返す
		return 80, 24
	}

	// ウィンドウサイズを計算
	width := int(csbi.Window.Right - csbi.Window.Left + 1)
	height := int(csbi.Window.Bottom - csbi.Window.Top + 1)

	return width, height
}
