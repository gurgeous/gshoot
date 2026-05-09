package ux

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/gurgeous/gshoot/internal/util"
	"github.com/schollz/progressbar/v3"
)

const interval = 50 * time.Millisecond

// Start displays progress and returns a stop function for the final message.
func Start(w io.Writer, label string) func(string) {
	// check to see if we can actually show progress
	file, ok := w.(*os.File)
	if ok && !util.IsTty(w) {
		ok = false
	}
	if !ok {
		return fallback(w, label)
	}

	// create bar
	bar := progressbar.NewOptions(-1,
		progressbar.OptionClearOnFinish(),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionSetDescription(Brand.Render(label)),
		progressbar.OptionSetWriter(file),
		progressbar.OptionSpinnerCustom([]string{
			Brand.Render("⠋"),
			Brand.Render("⠙"),
			Brand.Render("⠹"),
			Brand.Render("⠸"),
			Brand.Render("⠼"),
			Brand.Render("⠴"),
			Brand.Render("⠦"),
			Brand.Render("⠧"),
			Brand.Render("⠇"),
			Brand.Render("⠏"),
		}),
		progressbar.OptionThrottle(interval),
	)

	// goroutine
	stop := make(chan string, 1)
	stopped := make(chan struct{})
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		defer close(stopped)

		for {
			select {
			case <-ticker.C:
				_ = bar.Add(1)
			case msg := <-stop:
				_ = bar.Finish()
				// REMIND: is this a better way to do this?
				fmt.Fprintf(file, "%s %s\n", Success.Render("✓"), msg)
				return
			}
		}
	}()

	// wait for user to call stop
	return func(done string) {
		stop <- done
		<-stopped
	}
}

func fallback(w io.Writer, label string) func(string) {
	fmt.Fprintln(w, label)
	return func(done string) {
		fmt.Fprintf(w, "%s %s\n", Success.Render("✓"), done)
	}
}
