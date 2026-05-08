package cli

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/gurgeous/gshoot/internal/auth"
	"github.com/gurgeous/gshoot/internal/listing"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

var (
	resolveAuth = auth.Resolve
	newTokenSource = auth.NewTokenSource
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
		newStubCmd("down", "Download sheet data"),
		newListCmd(stdout, stderr),
	)

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
