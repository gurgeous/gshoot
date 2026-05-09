package down

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/gurgeous/gshoot/internal/auth"
	"github.com/gurgeous/gshoot/internal/google"
	"github.com/spf13/cobra"
	"google.golang.org/api/drive/v3"
)

var (
	resolveAuth    = auth.Resolve
	newTokenSource = auth.NewTokenSource
	newGoogle      = google.New
	downloadSheet  = Download
)

// NewCommand creates the down command.
func NewCommand() *cobra.Command {
	var outputPath string

	cmd := &cobra.Command{
		Use:   "down <spreadsheet> [sheet]",
		Short: "Download a Google Sheet as CSV",
		Example: strings.Join([]string{
			"gshoot down Budget",
			"  gshoot down Budget Q1 --output q1.csv",
		}, "\n"),
		Args: args,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			resolved, err := resolveAuth(auth.Options{Command: auth.CommandDown})
			if err != nil {
				return err
			}

			tokenSource, err := newTokenSource(ctx, resolved)
			if err != nil {
				return err
			}

			client, err := newGoogle(ctx, tokenSource)
			if err != nil {
				return err
			}

			sheetName := ""
			if len(args) == 2 {
				sheetName = args[1]
			}

			values, err := downloadSheet(ctx, client, args[0], sheetName)
			if err != nil {
				return err
			}

			writer := cmd.OutOrStdout()
			if outputPath != "" {
				file, err := os.Create(outputPath)
				if err != nil {
					return fmt.Errorf("create output file: %w", err)
				}
				defer file.Close()
				writer = file
			}

			return WriteCSV(writer, values)
		},
	}
	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "where to write the CSV")
	return cmd
}

func args(_ *cobra.Command, args []string) error {
	if len(args) >= 1 && len(args) <= 2 {
		return nil
	}
	return fmt.Errorf("expected `gshoot down <spreadsheet> [sheet]`")
}

// Recent returns recent spreadsheets ordered by modified time.
func Recent(ctx context.Context, client *google.Client, limit int) ([]*drive.File, error) {
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
