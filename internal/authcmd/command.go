package authcmd

import (
	"fmt"

	"github.com/gurgeous/gshoot/internal/auth"
	"github.com/gurgeous/gshoot/internal/login"
	"github.com/gurgeous/gshoot/internal/logout"
	"github.com/spf13/cobra"
)

var (
	status      = auth.InspectStatus
	printStatus = auth.PrintStatus
)

// NewCommand creates the auth command.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Login (or logout) from Google Sheets",
	}
	cmd.AddCommand(
		login.NewCommand(),
		newStatusCommand(),
		logout.NewCommand(),
	)
	return cmd
}

func newStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show auth status",
		Args:  noArgs("gshoot auth status"),
		RunE: func(cmd *cobra.Command, _ []string) error {
			printStatus(cmd.OutOrStdout(), status())
			return nil
		},
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
