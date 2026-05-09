package list

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/gurgeous/gshoot/internal/auth"
	"github.com/gurgeous/gshoot/internal/google"
	"github.com/gurgeous/gshoot/internal/ux"
	"github.com/spf13/cobra"
)

var (
	resolveAuth    = auth.Resolve
	newTokenSource = auth.NewTokenSource
	newGoogle      = google.New
	listRecent     = Recent
)

// NewCommand creates the list command.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "list",
		Short:         "List your Google Sheets",
		Example:       "  gshoot list",
		SilenceErrors: true,
		SilenceUsage:  true,
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) == 0 {
				return nil
			}
			return fmt.Errorf("expected `gshoot list`")
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := context.Background()
			stderr := cmd.ErrOrStderr()
			totalStart := time.Now()

			debugf(stderr, "resolving auth...")
			authStart := time.Now()
			resolved, err := resolveAuth(auth.Options{Command: auth.CommandList})
			if err != nil {
				return err
			}
			debugf(stderr, "auth ready in %s", time.Since(authStart).Round(time.Millisecond))

			debugf(stderr, "creating google client...")
			clientStart := time.Now()
			tokenSource, err := newTokenSource(ctx, resolved)
			if err != nil {
				return err
			}
			client, err := newGoogle(ctx, tokenSource)
			if err != nil {
				return err
			}
			debugf(stderr, "client ready in %s", time.Since(clientStart).Round(time.Millisecond))

			debugf(stderr, "listing recent spreadsheets...")
			stopDots := ux.StartDots(stderr, "listing spreadsheets...")
			files, driveDuration, err := listRecent(ctx, client, 10)
			if err != nil {
				stopDots("list failed")
				return err
			}
			stopDots(fmt.Sprintf("listed %d spreadsheets in %s", len(files), driveDuration.Round(time.Millisecond)))
			debugf(stderr, "drive returned %d spreadsheets in %s", len(files), driveDuration.Round(time.Millisecond))

			stdout := cmd.OutOrStdout()
			for _, file := range files {
				fmt.Fprintf(stdout, "%s  %s\n", file.ModifiedTime, file.Name)
			}
			debugf(stderr, "done in %s (%d spreadsheets)", time.Since(totalStart).Round(time.Millisecond), len(files))
			return nil
		},
	}
	return cmd
}

func debugf(w io.Writer, format string, args ...any) {
	fmt.Fprintf(w, "gshoot: "+format+"\n", args...)
}
