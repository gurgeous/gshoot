package commands

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/gurgeous/gshoot/auth"
	"github.com/gurgeous/gshoot/util"
	"github.com/gurgeous/gshoot/ux"
)

type AuthCmd struct {
	Login  AuthLoginCmd  `cmd:"" help:"Run browser OAuth login."`
	Logout AuthLogoutCmd `cmd:"" help:"Clear the cached OAuth token."`
	Status AuthStatusCmd `cmd:"" help:"Show auth status."`
}

type AuthLoginCmd struct {
	ClientSecretPath string `name:"client-secret" type:"path" help:"Path to a Google Desktop app OAuth client JSON."`
}

func (c *AuthLoginCmd) Run() error {
	return auth.Login(context.Background(), auth.LoginOptions{
		ClientSecretPath: c.ClientSecretPath,
		Stdout:           os.Stdout,
		Stderr:           os.Stderr,
	})
}

type AuthLogoutCmd struct{}

func (c *AuthLogoutCmd) Run() error {
	removed, err := auth.Logout()
	if err != nil {
		return err
	}
	if removed {
		fmt.Fprintln(os.Stdout, "Removed cached OAuth token. OAuth client config was kept.")
		return nil
	}
	fmt.Fprintln(os.Stdout, "No cached OAuth token was present.")
	return nil
}

type AuthStatusCmd struct{}

func (c *AuthStatusCmd) Run() error {
	writeStatus(os.Stdout)
	return nil
}

func writeStatus(w io.Writer) {
	configDir := util.ConfigDir()
	oauthClientPath := filepath.Join(configDir, "oauth-client.json")
	oauthTokenPath := filepath.Join(configDir, "oauth-token.json")
	hasOAuthClient := util.FileExists(oauthClientPath)
	hasCachedToken := util.FileExists(oauthTokenPath)
	loggedIn := false
	if hasCachedToken {
		token, err := auth.LoadOAuthToken(oauthTokenPath)
		loggedIn = err == nil && token.AccessToken != "" && (token.Expiry.IsZero() || token.Expiry.After(time.Now()))
	}

	fmt.Fprintln(w, ux.Subtle.Render("Config dir: "+configDir))
	fmt.Fprintln(w, ux.Subtle.Render("OAuth client: "+presentLine(hasOAuthClient, oauthClientPath)))
	fmt.Fprintln(w, ux.Subtle.Render("Cached token: "+presentLine(hasCachedToken, oauthTokenPath)))

	switch {
	case loggedIn:
		fmt.Fprintln(w, ux.Success.Render("Status: logged in"))
	case hasOAuthClient || hasCachedToken:
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
