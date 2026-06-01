package commands

import (
	"context"
	"fmt"
	"io"

	"github.com/gurgeous/gshoot/google"
	"github.com/gurgeous/gshoot/ux"
)

//
// Just a little helper for showing dots and finding a spreadsheet, several
// commands start up like this
//

type srunOptions struct {
	spreadsheet string // spreadsheet file name
	create      bool   // create spreadsheet file when missing
}

// srun is the shared runtime state for spreadsheet commands.
type srun struct {
	ctx      context.Context // request context for Google calls
	client   *google.Client  // authenticated Google API client
	progress *ux.Progress    // progress indicator for the command
	file     *google.File    // resolved spreadsheet file
}

// srunStart connects to Google and opens a spreadsheet file by name.
func srunStart(w io.Writer, opts srunOptions) (*srun, error) {
	ctx := context.Background()
	progress := ux.StartProgress(w, "connecting to Google Sheets...")

	client, err := google.NewClient(ctx)
	if err != nil {
		progress.Cancel()
		return nil, err
	}

	var file *google.File
	if opts.create {
		progress.SayFindOrCreateSpreadsheet(opts.spreadsheet)
		file, err = client.FindOrCreateSpreadsheetFile(ctx, opts.spreadsheet)
	} else {
		progress.SayFindSpreadsheet(opts.spreadsheet)
		file, err = client.FindSpreadsheetFile(ctx, opts.spreadsheet)
	}
	if err != nil {
		progress.Cancel()
		return nil, err
	}
	if file == nil {
		progress.Cancel()
		return nil, fmt.Errorf("could not find spreadsheet file named '%s'", opts.spreadsheet)
	}

	return &srun{
		ctx:      ctx,
		client:   client,
		progress: progress,
		file:     file,
	}, nil
}

func (c *srun) stop(err error) {
	if err == nil {
		c.progress.Stop()
		return
	}
	c.progress.Cancel()
}
