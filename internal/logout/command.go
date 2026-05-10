package logout

import (
	"fmt"

	"github.com/gurgeous/gshoot/internal/auth"
	"github.com/spf13/cobra"
)

var runLogout = auth.Logout

// NewLogoutCommand creates the auth logout command.
func NewLogoutCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Clear cached OAuth token",
		Args: func(_ *cobra.Command, args []string) error {
			if len(args) == 0 {
				return nil
			}
			return fmt.Errorf("gshoot auth logout")
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			removed, err := runLogout()
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
