package ux

import (
	"fmt"
	"io"
	"time"

	"github.com/gurgeous/gshoot/internal/util"
	"github.com/schollz/progressbar/v3"
)

//
// const
//

const (
	interval = 50 * time.Millisecond
	dots     = "⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏"
)

// map from dots to Brand.Render(ch)
var dotsWithColor = func() []string {
	items := make([]string, 0, len(dots))
	for _, ch := range dots {
		items = append(items, Brand.Render(string(ch)))
	}
	return items
}()

//
// dots start
//

// Dots controls an in-progress dots spinner.
type Dots struct {
	w           io.Writer
	bar         *progressbar.ProgressBar
	stop        chan struct{}
	stopped     chan struct{}
	description string
	tty         bool
}

// StartDots displays progress and returns a controller.
func StartDots(w io.Writer, description string) *Dots {
	d := &Dots{
		w:           w,
		description: description,
		tty:         util.IsTty(w),
	}

	// fallback
	if !d.tty {
		fmt.Fprintln(w, description)
		return d
	}

	d.bar = progressbar.NewOptions(-1,
		// REMIND: dots lost color again?
		progressbar.OptionClearOnFinish(),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionSetDescription(Brand.Render(d.description)),
		progressbar.OptionSetWriter(w),
		progressbar.OptionSpinnerCustom(dotsWithColor),
		progressbar.OptionThrottle(interval),
	)

	d.stop = make(chan struct{}, 1)
	d.stopped = make(chan struct{})
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		defer close(d.stopped)

		for {
			select {
			case <-ticker.C:
				_ = d.bar.Add(1)
			case <-d.stop:
				_ = d.bar.Finish()
				fmt.Fprintf(d.w, "%s %s\n", Success.Render("✓"), Brand.Render(d.description))
				return
			}
		}
	}()

	return d
}

// SetDescription changes the current description.
func (d *Dots) SetDescription(description string) {
	d.description = description
	if !d.tty {
		fmt.Fprintln(d.w, d.description)
		return
	}
	d.bar.Describe(Brand.Render(d.description))
}

// Stop stops the spinner and prints the final description.
func (d *Dots) Stop() {
	if !d.tty {
		fmt.Fprintf(d.w, "%s %s\n", Success.Render("✓"), d.description)
		return
	}
	d.stop <- struct{}{}
	<-d.stopped
}
