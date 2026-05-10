package authcmd

import (
	"github.com/gurgeous/gshoot/internal/login"
	"github.com/gurgeous/gshoot/internal/logout"
	"github.com/gurgeous/gshoot/internal/status"
	"github.com/spf13/cobra"
)

// NewAuthCommand creates the auth command. This doesn't do much, just houses
// the login/logout/status subcommands
func NewAuthCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Login (or logout) from Google Sheets",
	}
	cmd.AddCommand(
		login.NewLoginCommand(),
		logout.NewLogoutCommand(),
		status.NewStatusCommand(),
	)
	return cmd
}
