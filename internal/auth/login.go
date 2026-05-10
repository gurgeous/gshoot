package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gurgeous/gshoot/internal/util"
	"github.com/gurgeous/gshoot/internal/ux"
	"golang.org/x/oauth2"
)

var openBrowser = util.OpenBrowserURL

const oauthReadHeaderTimeout = 5 * time.Second

// LoginOptions configures interactive browser login.
type LoginOptions struct {
	ClientSecretPath string
	Stdout           io.Writer
	Stderr           io.Writer
}

// Login runs an interactive OAuth login and persists the token.
func Login(ctx context.Context, opts LoginOptions) error {
	configDir := ConfigDir()
	clientPath := filepath.Join(configDir, oauthClientFileName)
	tokenPath := filepath.Join(configDir, oauthTokenFileName)

	if err := os.MkdirAll(configDir, 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	if opts.ClientSecretPath != "" {
		if err := importOAuthClient(opts.ClientSecretPath, clientPath); err != nil {
			return err
		}
		fmt.Fprintln(opts.Stdout, ux.Success.Render("Saved OAuth client config to "+clientPath))
	}

	cred, err := LoadCredentialFile(clientPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return missingClientSecretError(clientPath)
		}
		return fmt.Errorf("load oauth client config: %w", err)
	}
	if cred.Kind != CredentialKindOAuthClient || cred.OAuthClient == nil {
		return fmt.Errorf("oauth client config at %s is not a desktop OAuth client", clientPath)
	}

	config, err := oauthConfigForLogin(cred.OAuthClient)
	if err != nil {
		return err
	}

	token, err := browserLoginFlow(ctx, config, opts.Stdout, opts.Stderr)
	if err != nil {
		return friendlyLoginError(err)
	}

	if err := saveOAuthToken(tokenPath, token); err != nil {
		return err
	}

	fmt.Fprintln(opts.Stdout, ux.Success.Render("Login complete. Token saved to "+tokenPath))
	return nil
}

func oauthConfigForLogin(client *OAuthClient) (*oauth2.Config, error) {
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

func importOAuthClient(srcPath, dstPath string) error {
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("read client secret file: %w", err)
	}
	cred, err := parseCredentialFile(srcPath, data)
	if err != nil {
		return fmt.Errorf("load client secret file: %w", err)
	}
	if cred.Kind != CredentialKindOAuthClient || cred.OAuthClient == nil {
		return errors.New("client secret file must be a Desktop app OAuth client JSON")
	}
	if err := util.WritePrivateFile(dstPath, data); err != nil {
		return fmt.Errorf("save oauth client config: %w", err)
	}
	return nil
}

func saveOAuthToken(path string, token *oauth2.Token) error {
	data, err := json.MarshalIndent(OAuthToken{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		TokenType:    token.TokenType,
		Expiry:       token.Expiry,
	}, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal oauth token: %w", err)
	}
	if err := util.WritePrivateFile(path, append(data, '\n')); err != nil {
		return fmt.Errorf("save oauth token: %w", err)
	}
	return nil
}

func missingClientSecretError(clientPath string) error {
	return fmt.Errorf(
		"no OAuth client config found at %s\n"+
			"setup:\n"+
			"1. Open https://console.cloud.google.com/apis/credentials\n"+
			"2. Configure the OAuth consent screen if prompted\n"+
			"3. Add yourself under Test users\n"+
			"4. Create an OAuth client of type Desktop app\n"+
			"5. Download the JSON and run `gshoot auth login --client-secret /path/to/client_secret.json`\n"+
			"without Test users, Google often fails with a vague \"Access blocked\" error",
		clientPath,
	)
}

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

func browserLoginFlow(ctx context.Context, config *oauth2.Config, stdout, stderr io.Writer) (*oauth2.Token, error) {
	state, err := util.RandomHex(16)
	if err != nil {
		return nil, fmt.Errorf("generate oauth state: %w", err)
	}

	redirectURL, callbackURL, receive, err := startLoopbackReceiver(config.RedirectURL, state)
	if err != nil {
		return nil, err
	}
	config = cloneConfig(config)
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

func cloneConfig(config *oauth2.Config) *oauth2.Config {
	cloned := *config
	cloned.Scopes = append([]string(nil), config.Scopes...)
	return &cloned
}

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

func startLoopbackReceiver(redirectRaw, state string) (string, string, func(context.Context) (string, error), error) {
	redirectURL, err := url.Parse(redirectRaw)
	if err != nil {
		return "", "", nil, fmt.Errorf("parse redirect url: %w", err)
	}

	host := redirectURL.Hostname()
	if host == "" {
		host = "127.0.0.1"
	}

	listener, err := net.Listen("tcp", net.JoinHostPort(host, "0"))
	if err != nil {
		return "", "", nil, fmt.Errorf("listen for oauth callback: %w", err)
	}

	redirectURL.Host = listener.Addr().String()
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)
	server := &http.Server{ReadHeaderTimeout: oauthReadHeaderTimeout}
	mux := http.NewServeMux()
	mux.HandleFunc(redirectURL.Path, func(w http.ResponseWriter, r *http.Request) {
		defer func() { go func() { _ = server.Shutdown(context.Background()) }() }()
		q := r.URL.Query()
		if got := q.Get("state"); got != state {
			http.Error(w, "state mismatch", http.StatusBadRequest)
			errCh <- errors.New("oauth callback state mismatch")
			return
		}
		if gotErr := q.Get("error"); gotErr != "" {
			http.Error(w, "login failed", http.StatusBadRequest)
			errCh <- fmt.Errorf("oauth callback error: %s", gotErr)
			return
		}
		code := q.Get("code")
		if code == "" {
			http.Error(w, "missing code", http.StatusBadRequest)
			errCh <- errors.New("oauth callback missing code")
			return
		}
		fmt.Fprintln(w, "gshoot login complete. You can close this tab.")
		codeCh <- code
	})
	server.Handler = mux

	go func() {
		if serveErr := server.Serve(listener); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			errCh <- serveErr
		}
	}()

	wait := func(ctx context.Context) (string, error) {
		defer listener.Close()
		select {
		case code := <-codeCh:
			return code, nil
		case err := <-errCh:
			return "", err
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}

	return redirectURL.String(), "http://" + listener.Addr().String() + redirectURL.Path, wait, nil
}
