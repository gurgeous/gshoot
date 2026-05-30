package ux

import (
	"bufio"
	"fmt"
	"os"
)

// Confirm asks a y/N question on stderr.
func Confirm(prompt string) {
	fmt.Fprintf(os.Stderr, "%s %s ", Warn.Render(prompt), Muted.Render("(y/n)"))
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	ok := len(line) > 0 && (line[0] == 'y' || line[0] == 'Y')
	if !ok {
		os.Exit(0)
	}
}
