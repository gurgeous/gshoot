package sub

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/gurgeous/gshoot/internal/auth"
	"github.com/gurgeous/gshoot/internal/util"
	"github.com/gurgeous/gshoot/internal/ux"
	"github.com/spf13/cobra"
)

//
// this has no behavior, it just houses login/logout/status.
//

var (
	runLogin    = auth.Login
	runLogout   = auth.Logout
	resolveAuth = auth.Resolve
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Login (or logout) from Google Sheets",
}

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(
		newLoginCommand(),
		newLogoutCommand(),
		newStatusCommand(),
	)
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

func newLogoutCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Clear cached OAuth token",
		Args:  noArgs("gshoot auth logout"),
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

func newStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show auth status",
		Args:  noArgs("gshoot auth status"),
		RunE: func(cmd *cobra.Command, _ []string) error {
			writeStatus(cmd.OutOrStdout())
			return nil
		},
	}
	return cmd
}

func writeStatus(w io.Writer) {
	configDir := auth.ConfigDir()
	oauthClientPath := filepath.Join(configDir, "oauth-client.json")
	oauthTokenPath := filepath.Join(configDir, "oauth-token.json")
	hasOAuthClient := util.FileExists(oauthClientPath)
	hasCachedToken := util.FileExists(oauthTokenPath)

	fmt.Fprintln(w, ux.Subtle.Render("Config dir: "+configDir))
	fmt.Fprintln(w, ux.Subtle.Render("OAuth client: "+presentLine(hasOAuthClient, oauthClientPath)))
	fmt.Fprintln(w, ux.Subtle.Render("Cached token: "+presentLine(hasCachedToken, oauthTokenPath)))

	resolved, err := resolveAuth()
	switch {
	case err == nil:
		msg := fmt.Sprintf("Status: authenticated via %s", resolved.Source.Kind)
		if resolved.Source.Path != "" {
			msg += " (" + resolved.Source.Path + ")"
		}
		fmt.Fprintln(w, ux.Success.Render(msg))
	case hasOAuthClient:
		fmt.Fprintln(w, ux.Warn.Render("Status: not logged in yet"))
		fmt.Fprintln(w, ux.Info.Render("Next step: run `gshoot auth login`"))
	default:
		fmt.Fprintln(w, ux.Warn.Render("Status: no auth configured"))
		fmt.Fprintln(w, ux.Info.Render("Next step: run `gshoot auth login --client-secret /path/to/client_secret.json`"))
	}
}

func presentLine(ok bool, path string) string {
	if ok {
		return "present (" + path + ")"
	}
	return "missing (" + path + ")"
}
