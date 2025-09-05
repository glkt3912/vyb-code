//go:build !windows
// +build !windows

package input

import (
	"os"
	"syscall"
	"unsafe"
)

// 端末サイズを取得 (Unix系専用)
func getTerminalSize() (int, int) {
	type winsize struct {
		Row    uint16
		Col    uint16
		Xpixel uint16
		Ypixel uint16
	}

	ws := &winsize{}
	ret, _, _ := syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(os.Stdout.Fd()),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(ws)))

	if int(ret) == -1 {
		return 80, 24 // デフォルト値
	}

	return int(ws.Col), int(ws.Row)
}
