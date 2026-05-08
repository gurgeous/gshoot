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
	resolveAuth     = auth.Resolve
	newTokenSource  = auth.NewTokenSource
	loginAuth       = auth.Login
	logoutAuth      = auth.Logout
	statusAuth      = auth.InspectStatus
	printAuthStatus = auth.PrintStatus
	newDownClient   = func(ctx context.Context, tokenSource oauth2.TokenSource) (down.Client, error) {
		return down.NewGoogleClient(ctx, tokenSource)
	}
	newListingClient = func(ctx context.Context, tokenSource oauth2.TokenSource) (listing.Client, error) {
		return listing.NewGoogleClient(ctx, tokenSource)
	}
)

const rootHelpTemplate = `{{with (or .Long .Short)}}{{.}}
{{end}}
gshoot [command]

{{range .Commands}}{{if (and .IsAvailableCommand (not .IsAdditionalHelpTopicCommand))}}  {{rpad .Name .NamePadding }}{{.Short}}
{{end}}{{end}}
`

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
		Short:         "Magically import/export CSVs from Google Sheets.",
		SilenceUsage:  true,
		SilenceErrors: true,
		CompletionOptions: cobra.CompletionOptions{
			HiddenDefaultCmd: true,
		},
	}

	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetHelpTemplate(rootHelpTemplate)
	cmd.AddCommand(
		newAuthCmd(stdout, stderr),
		newStubCmd("up", "Upload a local CSV file to a Google Sheet"),
		newDownCmd(stdout, stderr),
		newListCmd(stdout, stderr),
	)

	return cmd
}

func newAuthCmd(stdout, stderr io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Login (or logout) from Google Sheets",
	}
	cmd.AddCommand(
		newAuthLoginCmd(stdout, stderr),
		newAuthStatusCmd(stdout, stderr),
		newAuthLogoutCmd(stdout, stderr),
	)
	return cmd
}

func newAuthLoginCmd(stdout, stderr io.Writer) *cobra.Command {
	var clientSecretPath string

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Run browser OAuth login",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return loginAuth(context.Background(), auth.LoginOptions{
				Env:              auth.NewEnv(nil),
				ClientSecretPath: clientSecretPath,
				Stdout:           stdout,
				Stderr:           stderr,
			})
		},
	}
	cmd.Flags().StringVar(&clientSecretPath, "client-secret", "", "path to a downloaded Google Desktop app OAuth client JSON")
	return cmd
}

func newAuthStatusCmd(stdout, _ io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show auth status",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			printAuthStatus(stdout, statusAuth(auth.NewEnv(nil)))
			return nil
		},
	}
}

func newAuthLogoutCmd(stdout, _ io.Writer) *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Clear cached OAuth token",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			removed, err := logoutAuth(auth.NewEnv(nil))
			if err != nil {
				return err
			}
			if removed {
				fmt.Fprintln(stdout, "Removed cached OAuth token. OAuth client config was kept.")
			} else {
				fmt.Fprintln(stdout, "No cached OAuth token was present.")
			}
			return nil
		},
	}
}

func newDownCmd(stdout, stderr io.Writer) *cobra.Command {
	var outputPath string

	cmd := &cobra.Command{
		Use:   "down <spreadsheet> [sheet]",
		Short: "Download a Google Sheet as CSV",
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
		Short: "List your Google Sheets",
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
