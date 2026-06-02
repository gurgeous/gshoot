package ux

import (
	"fmt"
	"io"
	"time"

	"github.com/gurgeous/gshoot/util"
	"github.com/schollz/progressbar/v3"
)

//
// Progress spinner and w/ text
//

//
// const
//

const (
	interval = 50 * time.Millisecond
	dots     = "⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏"
)

//
// start
//

// Progress controls an active spinner.
type Progress struct {
	w           io.Writer
	bar         *progressbar.ProgressBar
	stopCh      chan bool
	stopped     chan struct{}
	description string
	tty         bool
}

// StartProgress displays progress and returns a controller.
func StartProgress(w io.Writer, description string) *Progress {
	p := &Progress{
		w:           w,
		description: description,
		tty:         util.IsTty(w),
	}

	// fallback
	if !p.tty {
		_, _ = fmt.Fprintln(w, description)
		return p
	}

	// hide cursor
	util.SetCursorVisible(w, false)

	// render spinner frames
	frames := make([]string, 0, len(dots))
	for _, ch := range dots {
		frames = append(frames, Warn.Render(string(ch)))
	}

	p.bar = progressbar.NewOptions(-1,
		progressbar.OptionClearOnFinish(),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionSetDescription(p.description),
		progressbar.OptionSetWriter(w),
		progressbar.OptionSpinnerCustom(frames),
		progressbar.OptionThrottle(interval),
	)

	p.stopCh = make(chan bool, 1)
	p.stopped = make(chan struct{})
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		defer close(p.stopped)

		for {
			select {
			case <-ticker.C:
				_ = p.bar.Add(1)
			case success := <-p.stopCh:
				_ = p.bar.Finish()
				if success {
					_, _ = fmt.Fprintf(w, "%s %s\n", Success.Render("✓"), p.description)
				}
				util.SetCursorVisible(w, true)
				return
			}
		}
	}()

	return p
}

// Stop stops the spinner and prints the final description.
func (p *Progress) Stop() {
	p.finish(true)
}

// Cancel stops the spinner without printing success.
func (p *Progress) Cancel() {
	p.finish(false)
}

func (p *Progress) finish(success bool) {
	if !p.tty {
		if success {
			_, _ = fmt.Fprintf(p.w, "%s %s\n", Success.Render("✓"), p.description)
		}
		return
	}
	p.stopCh <- success
	<-p.stopped
}

// set changes the current description.
func (p *Progress) set(description string) {
	p.description = description
	if !p.tty {
		_, _ = fmt.Fprintln(p.w, p.description)
		return
	}
	p.bar.Describe(p.description)
}

//
// SayXXX for output
//

//
// files
//

func (p *Progress) SayListFiles() {
	p.set("listing spreadsheet files...")
}

func (p *Progress) SayListedSpreadsheets(n int) {
	p.set(fmt.Sprintf("%d most recent spreadsheets", n))
}

//
// spreadsheets
//

func (p *Progress) SayFetchSpreadsheet(spreadsheet string) {
	p.set(fmt.Sprintf("fetching spreadsheet file %s...", Brand.Render(spreadsheet)))
}

func (p *Progress) SayFindSpreadsheet(spreadsheet string) {
	p.set(fmt.Sprintf("finding spreadsheet file %s...", Brand.Render(spreadsheet)))
}

func (p *Progress) SayFindOrCreateSpreadsheet(name string) {
	p.set(fmt.Sprintf("finding or creating spreadsheet file %s...", Brand.Render(name)))
}

func (p *Progress) SayWipeSpreadsheet(spreadsheet string) {
	p.set(fmt.Sprintf("wiping spreadsheet file %s...", Brand.Render(spreadsheet)))
}

//
// sheets
//

func (p *Progress) SayFindSheet(sheet string) {
	if sheet == "" {
		p.set("finding first sheet...")
		return
	}
	p.set(fmt.Sprintf("finding sheet %s...", Success.Render(sheet)))
}

func (p *Progress) SayPeekSheets(file string) {
	p.set(fmt.Sprintf("peeking in %s...", Brand.Render(file)))
}

func (p *Progress) SayUploadRows(n int, file, sheet string) {
	p.set(fmt.Sprintf("uploading %d rows to %s / %s...", n, Brand.Render(file), Success.Render(sheet)))
}

//
// rows
//

func (p *Progress) SayDownloadRows(spreadsheet string) {
	p.set(fmt.Sprintf("downloading rows from %s...", Brand.Render(spreadsheet)))
}

func (p *Progress) SaySaveRows(n int, path string) {
	if path == "" {
		p.set(fmt.Sprintf("fetching %d rows...", n))
	} else {
		p.set(fmt.Sprintf("writing %d rows to %s...", n, Brand.Render(path)))
	}
}
