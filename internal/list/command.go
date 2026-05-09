package list

import (
	"context"
	"fmt"

	"github.com/gurgeous/gshoot/internal/auth"
	"github.com/gurgeous/gshoot/internal/google"
	"github.com/gurgeous/gshoot/internal/ux"
	"github.com/spf13/cobra"
	"google.golang.org/api/drive/v3"
)

var (
	// grrr, dep injection
	listRecent     = recent
	newGoogle      = google.New
	newTokenSource = auth.NewTokenSource
	resolveAuth    = auth.Resolve
)

// NewCommand creates the list command.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "list",
		Short:         "List your Google Sheets",
		SilenceErrors: true,
		SilenceUsage:  true,
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) > 0 {
				return fmt.Errorf("expected `gshoot list`")
			}
			return nil
		},
		RunE: run,
	}
	return cmd
}

func run(cmd *cobra.Command, _ []string) error {
	ctx := context.Background()

	// auth
	resolved, err := resolveAuth(auth.Options{Command: auth.CommandList})
	if err != nil {
		return err
	}
	tokenSource, err := newTokenSource(ctx, resolved)
	if err != nil {
		return err
	}

	// create client
	client, err := newGoogle(ctx, tokenSource)
	if err != nil {
		return err
	}

	// list
	stopDots := ux.Start(cmd.ErrOrStderr(), "listing spreadsheets...")
	files, err := listRecent(ctx, client, 10)
	if err != nil {
		stopDots("list failed")
		return err
	}

	// done
	stopDots(fmt.Sprintf("%d recent spreadsheets", len(files)))
	for _, file := range files {
		fmt.Fprintf(cmd.OutOrStdout(), "%s  %s\n", file.ModifiedTime, file.Name)
	}
	return nil
}

// Recent returns recent spreadsheets ordered by modified time.
func recent(ctx context.Context, client *google.Client, limit int) ([]*drive.File, error) {
	res, err := client.Drive.Files.List().
		Context(ctx).
		Q("mimeType='application/vnd.google-apps.spreadsheet' and trashed=false").
		OrderBy("modifiedTime desc,name").
		PageSize(int64(limit)).
		Fields("files(id,name,modifiedTime)").
		Do()
	if err != nil {
		return nil, fmt.Errorf("list spreadsheets: %w", err)
	}
	return res.Files, nil
}
