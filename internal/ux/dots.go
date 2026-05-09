package ux

import (
	"fmt"
	"io"
	"os"
	"time"

	"golang.org/x/term"
)

var (
	interval = 50 * time.Millisecond
	frames   = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
)

// StartDots starts a simple spinner and returns a function that stops it.
func StartDots(w io.Writer, label string) func(string) {
	fd := os.Stdout.Fd()
	if fd > uintptr(^uint(0)>>1) || !term.IsTerminal(int(fd)) {
		// nop
		return func(string) {}
	}

	stop := make(chan string, 1)
	stopped := make(chan struct{})
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		defer close(stopped)

		idx := 0
		for {
			select {
			case <-ticker.C:
				// tick
				fmt.Fprintf(w, "\r\033[K%s %s", Info.Render(frames[idx%len(frames)]), label)
				idx++
			case msg := <-stop:
				// stop
				fmt.Fprintf(w, "\r\033[K%s %s\n", Success.Render("✓"), msg)
				return
			}
		}
	}()

	return func(msg string) {
		stop <- msg
		<-stopped
	}
}
