package status

import (
	"fmt"

	"github.com/gurgeous/gshoot/internal/auth"
	"github.com/spf13/cobra"
)

var (
	runStatus   = auth.InspectStatus
	writeStatus = auth.PrintStatus
)

// NewStatusCommand creates the auth status command.
func NewStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show auth status",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) == 0 {
				return nil
			}
			return fmt.Errorf("expected `gshoot auth status`")
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			writeStatus(cmd.OutOrStdout(), runStatus())
			return nil
		},
	}
	return cmd
}
