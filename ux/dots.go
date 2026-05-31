package ux

import (
	"fmt"
	"io"
	"time"

	lipgloss "charm.land/lipgloss/v2"
	"github.com/gurgeous/gshoot/util"
	"github.com/schollz/progressbar/v3"
)

//
// const
//

const (
	interval = 50 * time.Millisecond
	dots     = "⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏"
)

// dotsWithColor holds Brand-rendered spinner frames.
var dotsWithColor []string

// renderDots applies the current Brand style to spinner frames.
func renderDots() []string {
	items := make([]string, 0, len(dots))
	for _, ch := range dots {
		items = append(items, Warn.Render(string(ch)))
	}
	return items
}

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
func StartDots(w io.Writer) *Dots {
	description := "connecting..."
	d := &Dots{
		w:           w,
		description: description,
		tty:         util.IsTty(w),
	}

	// fallback
	if !d.tty {
		_, _ = lipgloss.Fprintln(w, description)
		return d
	}

	// hide cursor
	util.SetCursorVisible(w, false)

	d.bar = progressbar.NewOptions(-1,
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
				_, _ = lipgloss.Fprintf(w, "%s %s\n", Success.Render("✓"), Brand.Render(d.description))
				util.SetCursorVisible(w, true)
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
		_, _ = lipgloss.Fprintln(d.w, d.description)
		return
	}
	d.bar.Describe(Brand.Render(d.description))
}

//
// SayXXX for output
//

// connect

func (d *Dots) SayConnectGoogle() {
	d.SetDescription("connecting to Google Sheets...")
}

//
// files
//

func (d *Dots) SayListFiles() {
	d.SetDescription("listing spreadsheet files...")
}

func (d *Dots) SayListedSpreadsheets(n int) {
	d.SetDescription(fmt.Sprintf("%d most recent spreadsheets", n))
}

//
// spreadsheets
//

func (d *Dots) SayFetchSpreadsheet(spreadsheet string) {
	d.SetDescription(fmt.Sprintf("fetching spreadsheet file %s...", spreadsheet))
}

func (d *Dots) SayFindSpreadsheet(spreadsheet string) {
	d.SetDescription(fmt.Sprintf("finding spreadsheet file '%s'...", spreadsheet))
}

func (d *Dots) SayFindOrCreateSpreadsheet(name string) {
	d.SetDescription(fmt.Sprintf("finding or creating spreadsheet file '%s'...", name))
}

func (d *Dots) SayWipeSpreadsheet(spreadsheet string) {
	d.SetDescription(fmt.Sprintf("wiping spreadsheet file %s...", spreadsheet))
}

func (d *Dots) SayWipedSpreadsheet(spreadsheet string) {
	d.SetDescription(fmt.Sprintf("wiped spreadsheet file %s", spreadsheet))
}

//
// sheets
//

func (d *Dots) SayFindSheet(sheet string) {
	if sheet == "" {
		d.SetDescription("finding first sheet...")
		return
	}
	d.SetDescription(fmt.Sprintf("finding sheet '%s'...", sheet))
}

func (d *Dots) SayPeekSheets(file string) {
	d.SetDescription(fmt.Sprintf("peeking in %s...", file))
}

func (d *Dots) SayUploadRows(n int, file, sheet string) {
	d.SetDescription(fmt.Sprintf("uploading %d rows to %s sheet %s...", n, file, sheet))
}

//
// rows
//

func (d *Dots) SayDownloadRows(spreadsheet string) {
	d.SetDescription(fmt.Sprintf("downloading rows from %s...", spreadsheet))
}

func (d *Dots) SaySaveRows(n int, path string) {
	d.SetDescription(fmt.Sprintf("saving %d rows to %s...", n, path))
}

// Stop stops the spinner and prints the final description.
func (d *Dots) Stop() {
	if !d.tty {
		_, _ = lipgloss.Fprintf(d.w, "%s %s\n", Success.Render("✓"), d.description)
		return
	}
	d.stop <- struct{}{}
	<-d.stopped
}
