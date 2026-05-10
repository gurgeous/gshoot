package status

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/gurgeous/gshoot/internal/auth"
	"github.com/gurgeous/gshoot/internal/cmdutil"
	"github.com/gurgeous/gshoot/internal/util"
	"github.com/gurgeous/gshoot/internal/ux"
	"github.com/spf13/cobra"
)

var resolveAuth = auth.Resolve

// NewStatusCommand creates the auth status command.
func NewStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show auth status",
		Args:  cmdutil.NoArgs("gshoot auth status"),
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
