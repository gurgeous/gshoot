package auth

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/gurgeous/gshoot/util"
	"github.com/gurgeous/gshoot/ux"
	"golang.org/x/oauth2"
)

// auth/login.go implements the interactive browser login flow.

var openBrowser = util.OpenBrowserURL

// LoginOptions configures interactive browser login.
type LoginOptions struct {
	Stdout io.Writer
	Stderr io.Writer
}

// Login runs an interactive OAuth login and persists the token.
func (c *Manager) Login(ctx context.Context, opts LoginOptions) error {
	client, err := c.LoadOClient()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf(
				"no OAuth client config found at %s\n"+
					"setup:\n"+
					"1. Open https://console.cloud.google.com/apis/credentials\n"+
					"2. Configure the OAuth consent screen if prompted\n"+
					"3. Add yourself under Test users\n"+
					"4. Create an OAuth client of type Desktop app\n"+
					"5. Download the JSON and run `gshoot auth login --client-secret /path/to/client_secret.json`\n"+
					"without Test users, Google often fails with a vague \"Access blocked\" error",
				c.ClientPath(),
			)
		}
		return fmt.Errorf("load oauth client config: %w", err)
	}

	config, err := oauthConfigForLogin(client)
	if err != nil {
		return err
	}

	token, err := browserLoginFlow(ctx, config, opts.Stdout, opts.Stderr)
	if err != nil {
		return friendlyLoginError(err)
	}

	err = c.SaveOAuthToken(OAuthToken{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenType:    token.TokenType,
		Expiry:       token.Expiry,
	})
	if err != nil {
		return err
	}

	fmt.Fprintln(opts.Stdout, ux.Success.Render("Login complete. Token saved to "+c.TokenPath()))
	return nil
}

// oauthConfigForLogin builds an OAuth config for the browser login flow.
func oauthConfigForLogin(client *OClient) (*oauth2.Config, error) {
	redirect, err := selectLoopbackRedirect(client.RedirectURIs)
	if err != nil {
		return nil, err
	}

	return &oauth2.Config{
		ClientID:     client.ClientID,
		ClientSecret: client.ClientSecret,
		Endpoint:     oauthEndpoint(client.AuthURI, client.TokenURI),
		RedirectURL:  redirect.String(),
		Scopes: []string{
			"https://www.googleapis.com/auth/drive",
			"https://www.googleapis.com/auth/spreadsheets",
		},
	}, nil
}

// friendlyLoginError adds actionable hints to common Google OAuth failures.
func friendlyLoginError(err error) error {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "access_denied"), strings.Contains(msg, "Access blocked"):
		return fmt.Errorf("%w\nhint: add yourself under Test users on the OAuth consent screen and retry\nhint: console: https://console.cloud.google.com/apis/credentials/consent", err)
	case strings.Contains(msg, "redirect_uri_mismatch"):
		return fmt.Errorf("%w\nhint: use a Desktop app OAuth client, not a Web app client", err)
	case strings.Contains(msg, "invalid_client"):
		return fmt.Errorf("%w\nhint: re-download the Desktop app OAuth client JSON and retry `gshoot auth login --client-secret ...`", err)
	default:
		return err
	}
}

// browserLoginFlow performs the browser round trip and code exchange.
func browserLoginFlow(ctx context.Context, config *oauth2.Config, stdout, stderr io.Writer) (*oauth2.Token, error) {
	state, err := util.RandomHex(16)
	if err != nil {
		return nil, fmt.Errorf("generate oauth state: %w", err)
	}

	redirectURL, callbackURL, receive, err := startLoopback(config.RedirectURL, state)
	if err != nil {
		return nil, err
	}
	cloned := *config
	cloned.Scopes = append([]string(nil), config.Scopes...)
	config = &cloned
	config.RedirectURL = redirectURL

	authURL := config.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.SetAuthURLParam("prompt", "consent"))
	fmt.Fprintln(stdout, ux.Info.Render("Open this URL if the browser does not open:"))
	fmt.Fprintln(stdout, ux.Subtle.Render(authURL))
	if err := openBrowser(authURL); err != nil {
		fmt.Fprintln(stderr, ux.Warn.Render("Could not open browser automatically: "+err.Error()))
	}
	fmt.Fprintln(stdout, ux.Info.Render("Waiting for Google login at "+callbackURL+" ..."))

	code, err := receive(ctx)
	if err != nil {
		return nil, err
	}

	token, err := config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("exchange authorization code: %w", err)
	}
	return token, nil
}

// selectLoopbackRedirect picks the first localhost redirect URI from the client config.
func selectLoopbackRedirect(redirectURIs []string) (*url.URL, error) {
	for _, raw := range redirectURIs {
		u, err := url.Parse(raw)
		if err != nil {
			continue
		}
		host := u.Hostname()
		if host == "localhost" || host == "127.0.0.1" {
			if u.Path == "" {
				u.Path = "/"
			}
			return u, nil
		}
	}
	return nil, errors.New("oauth client config needs a localhost or 127.0.0.1 redirect URI")
}
