package cli

import (
	"context"
	"fmt"
	"io"
	"path/filepath"

	"github.com/gurgeous/gshoot/auth"
	"github.com/gurgeous/gshoot/util"
	"github.com/gurgeous/gshoot/ux"
)

var (
	runLogin    = auth.Login
	runLogout   = auth.Logout
	resolveAuth = auth.Resolve
)

type AuthCmd struct {
	Login  AuthLoginCmd  `cmd:"" help:"Run browser OAuth login."`
	Logout AuthLogoutCmd `cmd:"" help:"Clear cached OAuth token."`
	Status AuthStatusCmd `cmd:"" help:"Show auth status."`
}

type AuthLoginCmd struct {
	ClientSecretPath string `name:"client-secret" help:"Path to a downloaded Google Desktop app OAuth client JSON."`
}

func (c *AuthLoginCmd) Run(app *app) error {
	return runLogin(context.Background(), auth.LoginOptions{
		ClientSecretPath: c.ClientSecretPath,
		Stdout:           app.stdout,
		Stderr:           app.stderr,
	})
}

type AuthLogoutCmd struct{}

func (c *AuthLogoutCmd) Run(app *app) error {
	removed, err := runLogout()
	if err != nil {
		return err
	}
	if removed {
		fmt.Fprintln(app.stdout, "Removed cached OAuth token. OAuth client config was kept.")
		return nil
	}
	fmt.Fprintln(app.stdout, "No cached OAuth token was present.")
	return nil
}

type AuthStatusCmd struct{}

func (c *AuthStatusCmd) Run(app *app) error {
	writeStatus(app.stdout)
	return nil
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
