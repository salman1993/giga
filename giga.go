package main

import (
	"fmt"
	"os"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/term"
)

/*** data ***/

type editorConfig struct {
	screenrows  int
	screencols  int
	origTermios *term.State
}

type WinSize struct {
	Row    uint16
	Col    uint16
	Xpixel uint16
	Ypixel uint16
}

var E editorConfig

const clearScreenSeq = "\x1b[2J"
const moveCursorTopLeft = "\x1b[H"

/*** terminal ***/

func die(s string, err error) {
	clearScreen()
	disableRawMode()

	fmt.Fprintf(os.Stderr, "%s: %v\n", s, err)
	os.Exit(1)
}

func enableRawMode() {
	prevState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		die("enableRawMode", err)
	}
	E.origTermios = prevState
}

func disableRawMode() {
	if err := term.Restore(int(os.Stdin.Fd()), E.origTermios); err != nil {
		die("disableRawMode", err)
	}
}

func CTRL_KEY(b byte) byte {
	return b & 0x1f
}

// wait for one keypress, and return it
func editorReadKey() byte {
	var c byte
	buf := make([]byte, 1)

	for {
		nread, err := os.Stdin.Read(buf)

		if err != nil {
			die("read", err)
		}

		fmt.Printf("nread: %v, buf: %v\r\n", nread, buf)
		if nread == 1 {
			c = buf[0]
			return c
		}
	}
}

func getWindowSize(rows, cols *int) int {
	var ws WinSize

	_, _, err := syscall.Syscall(
		syscall.SYS_IOCTL,            // System call number
		uintptr(os.Stdout.Fd()),      // File descriptor
		uintptr(syscall.TIOCGWINSZ),  // ioctl request code
		uintptr(unsafe.Pointer(&ws)), // Pointer to winsize struct
	)

	if err != 0 {
		return -1
	}

	*cols = int(ws.Col)
	*rows = int(ws.Row)
	return 0
}

/*** input ***/

// waits for a keypress, and then handles it
func editorProcessKeypress() {
	c := editorReadKey()

	switch c {
	case CTRL_KEY('q'):
		clearScreen()
		disableRawMode()
		os.Exit(0)
	default:
		fmt.Printf("Read: %d ('%c')\r\n", c, c)
	}
}

/*** output ***/

func clearScreen() {
	os.Stdout.WriteString(clearScreenSeq)
	os.Stdout.WriteString(moveCursorTopLeft)
}

// handle drawing each row of the buffer of text being edited
func editorDrawRows() {
	for range E.screenrows {
		fmt.Printf("~\r\n")
	}
}

func editorRefreshScreen() {
	clearScreen()

	editorDrawRows()

	os.Stdout.WriteString(moveCursorTopLeft)
}

/*** init ***/

func initEditor() {

	if getWindowSize(&E.screenrows, &E.screencols) == -1 {
		die("getWindowSize", nil)
	}

}

func main() {
	enableRawMode()
	defer disableRawMode()

	initEditor()

	for {
		editorRefreshScreen()
		editorProcessKeypress()
		time.Sleep(1 * time.Second)
	}

}
