package commands

import (
	"context"
	"fmt"

	"github.com/gurgeous/gshoot/app"
	"github.com/gurgeous/gshoot/google"
	"github.com/gurgeous/gshoot/ux"
)

type srunOptions struct {
	spreadsheet string // spreadsheet file name
	create      bool   // create spreadsheet file when missing
}

// srun is the shared runtime state for spreadsheet commands.
type srun struct {
	ctx    context.Context // request context for Google calls
	client *google.Client  // authenticated Google API client
	dots   *ux.Dots        // progress indicator for the command
	file   *google.File    // resolved spreadsheet file
}

// srunStart connects to Google and opens a spreadsheet file by name.
func srunStart(a *app.App, opts srunOptions) (*srun, error) {
	ctx := context.Background()
	dots := ux.StartDots(a.RawStderr(), "connecting to Google Sheets...")
	dots.SayConnectGoogle()

	client, err := google.NewClient(ctx)
	if err != nil {
		dots.Cancel()
		return nil, err
	}

	var file *google.File
	if opts.create {
		dots.SayFindOrCreateSpreadsheet(opts.spreadsheet)
		file, err = client.FindOrCreateSpreadsheetFile(ctx, opts.spreadsheet)
	} else {
		dots.SayFindSpreadsheet(opts.spreadsheet)
		file, err = client.FindSpreadsheetFile(ctx, opts.spreadsheet)
	}
	if err != nil {
		dots.Cancel()
		return nil, err
	}
	if file == nil {
		dots.Cancel()
		return nil, fmt.Errorf("could not find spreadsheet file named '%s'", opts.spreadsheet)
	}

	return &srun{
		ctx:    ctx,
		client: client,
		dots:   dots,
		file:   file,
	}, nil
}

func (c *srun) stop(err error) {
	if err == nil {
		c.dots.Stop()
		return
	}
	c.dots.Cancel()
}
