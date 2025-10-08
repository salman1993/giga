package main

import (
	"bufio"
	"fmt"
	"os"
	"unicode"

	"golang.org/x/term"
)

func die(err error) {
	fmt.Fprintf(os.Stderr, "%v\n", err)
	os.Exit(1)
}

func editorReadKey(reader *bufio.Reader) byte {
	b, err := reader.ReadByte()
	if err != nil {
		die(err)
	}
	return b
}

func main() {
	// Put terminal into raw mode
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		panic(err)
	}
	// Disable raw mode when main() function exits
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	// Read from stdin and print. In raw mode, we need to add carriage returns and line feeds with '\r\n'.
	reader := bufio.NewReader(os.Stdin)
	for {
		b, err := reader.ReadByte()
		if err != nil {
			fmt.Printf("Error reading byte: %v\r\n", err)
			break
		}

		if unicode.IsControl(rune(b)) {
			fmt.Printf("Read: %d\r\n", b)

		} else {
			fmt.Printf("Read: %d ('%c')\r\n", b, b)

		}

		if b == 'q' {
			break
		}
	}
}
