package sub

import (
	"context"
	"strings"

	"github.com/gurgeous/gshoot/internal/auth"
	"github.com/spf13/cobra"
)

var runLogin = auth.Login

func init() { authCmd.AddCommand(newLoginCommand()) }

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
			return runLogin(context.Background(), auth.LoginOptions{
				ClientSecretPath: clientSecretPath,
				Stdout:           cmd.OutOrStdout(),
				Stderr:           cmd.ErrOrStderr(),
			})
		},
	}
	cmd.Flags().StringVar(&clientSecretPath, "client-secret", "", "path to a downloaded Google Desktop app OAuth client JSON")
	return cmd
}
