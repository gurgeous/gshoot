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

//
// dots start
//

// Dots controls an in-progress dots spinner.
type Dots struct {
	w           io.Writer
	bar         *progressbar.ProgressBar
	stopCh      chan bool
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
		_, _ = lipgloss.Fprintln(w, description)
		return d
	}

	// hide cursor
	util.SetCursorVisible(w, false)

	// render dots
	dotsWithColor := make([]string, 0, len(dots))
	for _, ch := range dots {
		dotsWithColor = append(dotsWithColor, Warn.Render(string(ch)))
	}

	d.bar = progressbar.NewOptions(-1,
		progressbar.OptionClearOnFinish(),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionSetDescription(Brand.Render(d.description)),
		progressbar.OptionSetWriter(w),
		progressbar.OptionSpinnerCustom(dotsWithColor),
		progressbar.OptionThrottle(interval),
	)

	d.stopCh = make(chan bool, 1)
	d.stopped = make(chan struct{})
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		defer close(d.stopped)

		for {
			select {
			case <-ticker.C:
				_ = d.bar.Add(1)
			case success := <-d.stopCh:
				_ = d.bar.Finish()
				if success {
					_, _ = lipgloss.Fprintf(w, "%s %s\n", Success.Render("✓"), Brand.Render(d.description))
				}
				util.SetCursorVisible(w, true)
				return
			}
		}
	}()

	return d
}

// Stop stops the spinner and prints the final description.
func (d *Dots) Stop() {
	d.finish(true)
}

// Cancel stops the spinner without printing success.
func (d *Dots) Cancel() {
	d.finish(false)
}

func (d *Dots) finish(success bool) {
	if !d.tty {
		if success {
			_, _ = lipgloss.Fprintf(d.w, "%s %s\n", Success.Render("✓"), d.description)
		}
		return
	}
	d.stopCh <- success
	<-d.stopped
}

// set changes the current description.
func (d *Dots) set(description string) {
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
	d.set("connecting to Google Sheets...")
}

//
// files
//

func (d *Dots) SayListFiles() {
	d.set("listing spreadsheet files...")
}

func (d *Dots) SayListedSpreadsheets(n int) {
	d.set(fmt.Sprintf("%d most recent spreadsheets", n))
}

//
// spreadsheets
//

func (d *Dots) SayFetchSpreadsheet(spreadsheet string) {
	d.set(fmt.Sprintf("fetching spreadsheet file %s...", spreadsheet))
}

func (d *Dots) SayFindSpreadsheet(spreadsheet string) {
	d.set(fmt.Sprintf("finding spreadsheet file '%s'...", spreadsheet))
}

func (d *Dots) SayFindOrCreateSpreadsheet(name string) {
	d.set(fmt.Sprintf("finding or creating spreadsheet file '%s'...", name))
}

func (d *Dots) SayWipeSpreadsheet(spreadsheet string) {
	d.set(fmt.Sprintf("wiping spreadsheet file %s...", spreadsheet))
}

func (d *Dots) SayWipedSpreadsheet(spreadsheet string) {
	d.set(fmt.Sprintf("wiped spreadsheet file %s", spreadsheet))
}

//
// sheets
//

func (d *Dots) SayFindSheet(sheet string) {
	if sheet == "" {
		d.set("finding first sheet...")
		return
	}
	d.set(fmt.Sprintf("finding sheet '%s'...", sheet))
}

func (d *Dots) SayPeekSheets(file string) {
	d.set(fmt.Sprintf("peeking in %s...", file))
}

func (d *Dots) SayUploadRows(n int, file, sheet string) {
	if sheet == "" {
		d.set(fmt.Sprintf("uploading %d rows to %s...", n, file))
		return
	}
	d.set(fmt.Sprintf("uploading %d rows to %s sheet %s...", n, file, sheet))
}

//
// rows
//

func (d *Dots) SayDownloadRows(spreadsheet string) {
	d.set(fmt.Sprintf("downloading rows from %s...", spreadsheet))
}

func (d *Dots) SaySaveRows(n int, path string) {
	d.set(fmt.Sprintf("saving %d rows to %s...", n, path))
}
