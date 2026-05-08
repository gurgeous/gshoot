package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/gurgeous/gshoot/internal/auth"
	"github.com/gurgeous/gshoot/internal/down"
	"github.com/gurgeous/gshoot/internal/listing"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

var (
	resolveAuth    = auth.Resolve
	newTokenSource = auth.NewTokenSource
	newDownClient  = func(ctx context.Context, tokenSource oauth2.TokenSource) (down.Client, error) {
		return down.NewGoogleClient(ctx, tokenSource)
	}
	newListingClient = func(ctx context.Context, tokenSource oauth2.TokenSource) (listing.Client, error) {
		return listing.NewGoogleClient(ctx, tokenSource)
	}
)

// Run executes the gshoot CLI.
func Run(args []string, stdout, stderr io.Writer) int {
	cmd := newRootCmd(stdout, stderr)
	cmd.SetArgs(args)

	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	return 0
}

func newRootCmd(stdout, stderr io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "gshoot",
		Short:         "CSV to Google Sheets workflows",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.AddCommand(
		newStubCmd("up", "Upload CSV data to Google Sheets"),
		newDownCmd(stdout, stderr),
		newListCmd(stdout, stderr),
	)

	return cmd
}

func newDownCmd(stdout, stderr io.Writer) *cobra.Command {
	var outputPath string

	cmd := &cobra.Command{
		Use:   "down <spreadsheet> [sheet]",
		Short: "Download sheet data",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(_ *cobra.Command, args []string) error {
			ctx := context.Background()
			resolved, err := resolveAuth(auth.Options{
				Env:     auth.NewEnv(nil),
				Command: auth.CommandDown,
			})
			if err != nil {
				return err
			}

			tokenSource, err := newTokenSource(ctx, resolved)
			if err != nil {
				return err
			}

			client, err := newDownClient(ctx, tokenSource)
			if err != nil {
				return err
			}

			sheetName := ""
			if len(args) == 2 {
				sheetName = args[1]
			}

			result, err := down.NewService(client).Download(ctx, args[0], sheetName)
			if err != nil {
				var spreadsheetErr *down.SpreadsheetNotFoundError
				if errors.As(err, &spreadsheetErr) {
					return spreadsheetNotFoundError(spreadsheetErr.Name)
				}

				var sheetErr *down.SheetNotFoundError
				if errors.As(err, &sheetErr) {
					return sheetNotFoundError(sheetErr.Spreadsheet, sheetErr.Sheet)
				}

				var noSheetsErr *down.NoSheetsError
				if errors.As(err, &noSheetsErr) {
					return noSheetsError(noSheetsErr.Spreadsheet)
				}
				return err
			}

			writer := stdout
			if outputPath != "" {
				file, err := os.Create(outputPath)
				if err != nil {
					return fmt.Errorf("create output file: %w", err)
				}
				defer file.Close()
				writer = file
			}

			return down.WriteCSV(writer, result.Values)
		},
	}
	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "where to write the CSV")
	return cmd
}

func newStubCmd(use, short string) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		Run:   func(*cobra.Command, []string) {},
	}
}

func newListCmd(stdout, stderr io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List recent spreadsheets",
		Args:  cobra.NoArgs,
		RunE: func(*cobra.Command, []string) error {
			ctx := context.Background()
			resolved, err := resolveAuth(auth.Options{
				Env:     auth.NewEnv(nil),
				Command: auth.CommandList,
			})
			if err != nil {
				return err
			}

			tokenSource, err := newTokenSource(ctx, resolved)
			if err != nil {
				return err
			}

			client, err := newListingClient(ctx, tokenSource)
			if err != nil {
				return err
			}

			items, err := listing.NewService(client).ListRecent(ctx, 10)
			if err != nil {
				return err
			}

			for _, item := range items {
				fmt.Fprintf(stdout, "%s  %s\n", item.ModifiedTime.UTC().Format(time.RFC3339), item.Name)
				if len(item.SheetNames) > 1 {
					preview := item.SheetNames
					suffix := ""
					if len(preview) > 3 {
						preview = preview[:3]
						suffix = ", ..."
					}
					fmt.Fprintf(stdout, "  %s%s\n", strings.Join(preview, ", "), suffix)
				}
			}

			return nil
		},
	}
}
