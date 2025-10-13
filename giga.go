package main

import (
	"fmt"
	"os"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/term"
)

/*** data ***/

const GIGA_VERSION = "0.0.1"

// editor keys
const (
	ARROW_LEFT = 1000 + iota
	ARROW_RIGHT
	ARROW_UP
	ARROW_DOWN
	DEL_KEY
	HOME_KEY
	END_KEY
	PAGE_UP
	PAGE_DOWN
)

type editorConfig struct {
	cx          int // E.cx is the horizontal coordinate of the cursor (the column)
	cy          int // E.cy is the vertical coordinate (the row)
	screenrows  int // window size - # rows
	screencols  int // window size - # cols
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

func CTRL_KEY(b int) int {
	return b & 0x1f
}

// wait for one keypress, and return it
func editorReadKey() int {
	buf := make([]byte, 1)

	for {
		nread, err := os.Stdin.Read(buf)

		if err != nil {
			die("read", err)
		}

		if nread == 1 {
			break
		}
	}

	c := buf[0]

	// Check if it's an escape sequence
	if c == '\x1b' {
		seq := make([]byte, 3)

		// Try to read second byte
		if nread, _ := os.Stdin.Read(seq[0:1]); nread != 1 {
			return '\x1b' // Timeout, just ESC key
		}

		// Try to read third byte
		if nread, _ := os.Stdin.Read(seq[1:2]); nread != 1 {
			return '\x1b' // Timeout, just ESC key
		}

		// Check if it's an arrow key sequence
		switch seq[0] {
		case '[':
			if seq[1] >= '0' && seq[1] <= '9' {
				// Try to read fourth byte
				if nread, _ := os.Stdin.Read(seq[2:3]); nread != 1 {
					return '\x1b' // Timeout, just ESC key
				}

				if seq[2] == '~' {
					switch seq[1] {
					case '1':
						return HOME_KEY
					case '3':
						return DEL_KEY
					case '4':
						return END_KEY
					case '5':
						return PAGE_UP
					case '6':
						return PAGE_DOWN
					case '7':
						return HOME_KEY
					case '8':
						return END_KEY

					}

				}

			} else {
				// 3 bytes read
				switch seq[1] {
				case 'A':
					return ARROW_UP
				case 'B':
					return ARROW_DOWN
				case 'C':
					return ARROW_RIGHT
				case 'D':
					return ARROW_LEFT

				case 'H':
					return HOME_KEY
				case 'F':
					return END_KEY
				}
			}

		case 'O':
			switch seq[1] {
			case 'H':
				return HOME_KEY
			case 'F':
				return END_KEY
			}
		}

		return '\x1b' // Unknown escape sequence
	}

	return int(c)
}

func getCursorPosition(rows *int, cols *int) int {
	buf := make([]byte, 32)
	i := 0

	// Write the cursor position query escape sequence
	if n, err := os.Stdout.Write([]byte("\x1b[6n")); err != nil || n != 4 {
		return -1
	}

	// Read response until we get 'R' or buffer is full
	for i < len(buf)-1 {
		n, err := os.Stdin.Read(buf[i : i+1])
		if n != 1 || err != nil {
			break
		}
		if buf[i] == 'R' {
			break
		}
		i++
	}

	// Check for expected escape sequence prefix
	if i < 2 || buf[0] != '\x1b' || buf[1] != '[' {
		return -1
	}

	// Parse the row and column values
	n, err := fmt.Sscanf(string(buf[2:i]), "%d;%d", rows, cols)
	if err != nil || n != 2 {
		return -1
	}

	return 0
}

func getWindowSize(rows, cols *int) int {
	var ws WinSize

	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,            // System call number
		uintptr(os.Stdout.Fd()),      // File descriptor
		uintptr(syscall.TIOCGWINSZ),  // ioctl request code
		uintptr(unsafe.Pointer(&ws)), // Pointer to winsize struct
	)

	if errno != 0 {
		if _, err := os.Stdout.WriteString("\x1b[999C\x1b[999B"); err != nil {
			return -1
		}
		return getCursorPosition(rows, cols)
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

	case ARROW_UP, ARROW_DOWN, ARROW_LEFT, ARROW_RIGHT:
		editorMoveCursor(c)
	case PAGE_UP, PAGE_DOWN:
		times := E.screenrows
		for times > 0 {
			if c == PAGE_UP {
				editorMoveCursor(ARROW_UP)
			} else {
				editorMoveCursor(ARROW_DOWN)
			}
			times--
		}
	case HOME_KEY:
		E.cx = 0
	case END_KEY:
		E.cx = E.screencols - 1

	}
}

func editorMoveCursor(key int) {
	switch key {
	case ARROW_LEFT:
		if E.cx != 0 {
			E.cx--
		}
	case ARROW_RIGHT:
		if E.cx != E.screencols-1 {
			E.cx++
		}
	case ARROW_UP:
		if E.cy != 0 {
			E.cy--
		}
	case ARROW_DOWN:
		if E.cx != E.screenrows-1 {
			E.cy++
		}
	}
}

/*** output ***/

func clearScreen() {
	os.Stdout.WriteString(clearScreenSeq)
	os.Stdout.WriteString(moveCursorTopLeft)
}

// handle drawing each row of the buffer of text being edited
func editorDrawRows(buffer *strings.Builder) {
	for y := range E.screenrows {

		if y == E.screenrows/3 {
			welcome := fmt.Sprintf("Kilo editor -- version %s", GIGA_VERSION)
			welcomelen := min(len(welcome), E.screencols)

			padding := (E.screencols - welcomelen) / 2
			if padding > 0 {
				buffer.WriteString("~")
				padding--
			}
			for padding > 0 {
				buffer.WriteString(" ")
				padding--
			}

			// Write only the portion that fits on screen
			buffer.WriteString(welcome[:welcomelen])
		} else {
			buffer.WriteString("~")
		}

		buffer.WriteString("\x1b[K") // erases part of the line right of the cursor

		if y < E.screenrows-1 {
			buffer.WriteString("\r\n")
		}
	}
}

func editorRefreshScreen() {
	var buffer strings.Builder

	buffer.WriteString("\x1b[?25l") // hide cursor before refreshing the screen
	buffer.WriteString(moveCursorTopLeft)

	editorDrawRows(&buffer)

	// move cursor
	cursorPosSeq := fmt.Sprintf("\x1b[%d;%dH", E.cy+1, E.cx+1)
	buffer.WriteString(cursorPosSeq)

	buffer.WriteString("\x1b[?25h") // show cursor after the refresh finishes

	os.Stdout.WriteString(buffer.String())
}

/*** init ***/

func initEditor() {
	E.cx = 0
	E.cy = 0

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
		// time.Sleep(time.Second)
	}

}
