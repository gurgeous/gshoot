package auth

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var (
	login       = Login
	logout      = Logout
	status      = InspectStatus
	printStatus = PrintStatus
)

// NewCommand creates the auth command.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Login (or logout) from Google Sheets",
	}
	cmd.AddCommand(
		newLoginCommand(),
		newStatusCommand(),
		newLogoutCommand(),
	)
	return cmd
}

func newLoginCommand() *cobra.Command {
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
			return login(context.Background(), LoginOptions{
				ClientSecretPath: clientSecretPath,
				Stdout:           cmd.OutOrStdout(),
				Stderr:           cmd.ErrOrStderr(),
			})
		},
	}
	cmd.Flags().StringVar(&clientSecretPath, "client-secret", "", "path to a downloaded Google Desktop app OAuth client JSON")
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

func newLogoutCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Clear cached OAuth token",
		Args:  noArgs("gshoot auth logout"),
		RunE: func(cmd *cobra.Command, _ []string) error {
			removed, err := logout()
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

func noArgs(usage string) cobra.PositionalArgs {
	return func(_ *cobra.Command, args []string) error {
		if len(args) == 0 {
			return nil
		}
		return fmt.Errorf("expected `%s`", usage)
	}
}
