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
	"github.com/gurgeous/gshoot/internal/ux"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

var (
	version         = "dev"
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

// Run executes the gshoot CLI.
func Run(args []string, stdout, stderr io.Writer) int {
	ux.Init()

	cmd := newRootCmd()
	cmd.SetArgs(args)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)

	if err := cmd.Execute(); err != nil {
		writeError(stderr, err)
		return 1
	}

	return 0
}

//
// root
//

func newRootCmd() *cobra.Command {
	var showVersion bool

	cmd := &cobra.Command{
		Use:           "gshoot",
		Short:         fmt.Sprintf("Magically %s from Google Sheets.", ux.Brand.Render("import/export CSVs")),
		SilenceErrors: true,
		SilenceUsage:  true,
		CompletionOptions: cobra.CompletionOptions{
			HiddenDefaultCmd: true,
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			if showVersion {
				fmt.Fprintf(cmd.OutOrStdout(), "gshoot %s\n", version)
				return nil
			}
			writeHelp(cmd.OutOrStdout(), cmd)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&showVersion, "version", "v", false, "print version number")
	cmd.SetHelpFunc(func(command *cobra.Command, _ []string) {
		writeHelp(command.OutOrStdout(), command)
	})
	cmd.AddCommand(
		newAuthCmd(),
		newStubCmd("up", "Upload a local CSV file to a Google Sheet"),
		newDownCmd(),
		newListCmd(),
	)

	return cmd
}

//
// commands
//

func newAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Login (or logout) from Google Sheets",
	}
	cmd.AddCommand(
		newAuthLoginCmd(),
		newAuthStatusCmd(),
		newAuthLogoutCmd(),
	)
	return cmd
}

func newAuthLoginCmd() *cobra.Command {
	var clientSecretPath string

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Run browser OAuth login",
		Example: strings.Join([]string{
			"gshoot auth login",
			"  gshoot auth login --client-secret ~/Downloads/client_secret.json",
		}, "\n"),
		Args: noArgs("gshoot auth login"),
		RunE: func(cmd *cobra.Command, _ []string) error {
			return loginAuth(context.Background(), auth.LoginOptions{
				ClientSecretPath: clientSecretPath,
				Stdout:           cmd.OutOrStdout(),
				Stderr:           cmd.ErrOrStderr(),
			})
		},
	}
	cmd.Flags().StringVar(&clientSecretPath, "client-secret", "", "path to a downloaded Google Desktop app OAuth client JSON")
	return cmd
}

func newAuthStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show auth status",
		Args:  noArgs("gshoot auth status"),
		RunE: func(cmd *cobra.Command, _ []string) error {
			printAuthStatus(cmd.OutOrStdout(), statusAuth())
			return nil
		},
	}
	return cmd
}

func newAuthLogoutCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Clear cached OAuth token",
		Args:  noArgs("gshoot auth logout"),
		RunE: func(cmd *cobra.Command, _ []string) error {
			removed, err := logoutAuth()
			if err != nil {
				return err
			}
			if removed {
				fmt.Fprintln(cmd.OutOrStdout(), "Removed cached OAuth token. OAuth client config was kept.")
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "No cached OAuth token was present.")
			}
			return nil
		},
	}
	return cmd
}

func newDownCmd() *cobra.Command {
	var outputPath string

	cmd := &cobra.Command{
		Use:   "down <spreadsheet> [sheet]",
		Short: "Download a Google Sheet as CSV",
		Example: strings.Join([]string{
			"gshoot down Budget",
			"  gshoot down Budget Q1 --output q1.csv",
		}, "\n"),
		Args: downArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			resolved, err := resolveAuth(auth.Options{
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

			writer := cmd.OutOrStdout()
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
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Run:   func(*cobra.Command, []string) {},
	}
	return cmd
}

func noArgs(usage string) cobra.PositionalArgs {
	return func(_ *cobra.Command, args []string) error {
		if len(args) == 0 {
			return nil
		}
		return fmt.Errorf("expected `%s`", usage)
	}
}

func downArgs(_ *cobra.Command, args []string) error {
	if len(args) >= 1 && len(args) <= 2 {
		return nil
	}
	return fmt.Errorf("expected `gshoot down <spreadsheet> [sheet]`")
}

func newListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List your Google Sheets",
		Example: "  gshoot list",
		Args:    noArgs("gshoot list"),
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := context.Background()
			resolved, err := resolveAuth(auth.Options{
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

			stdout := cmd.OutOrStdout()
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
	return cmd
}

func isRootCmd(cmd *cobra.Command) bool {
	return cmd != nil && !cmd.HasParent()
}
