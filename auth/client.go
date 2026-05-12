package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/gurgeous/gshoot/util"
	"github.com/gurgeous/gshoot/ux"
)

// auth/client.go owns on-disk browser auth state and status reporting.

// AuthClient manages browser auth files under one config directory.
type AuthClient struct {
	ConfigDir string
}

// NewAuthClient builds an auth client for the default config directory.
func NewAuthClient() *AuthClient {
	return &AuthClient{ConfigDir: util.ConfigDir()}
}

// ClientPath returns the oauth-client.json path.
func (c *AuthClient) ClientPath() string {
	return filepath.Join(c.ConfigDir, "oauth-client.json")
}

// TokenPath returns the oauth-token.json path.
func (c *AuthClient) TokenPath() string {
	return filepath.Join(c.ConfigDir, "oauth-token.json")
}

// LoadOAuthClient reads the saved OAuth client config.
func (c *AuthClient) LoadOAuthClient() (*OAuthClient, error) {
	return loadOAuthClient(c.ClientPath())
}

// LoadOAuthToken reads the saved OAuth token.
func (c *AuthClient) LoadOAuthToken() (OAuthToken, error) {
	return loadOAuthToken(c.TokenPath())
}

// SaveOAuthToken writes the saved OAuth token.
func (c *AuthClient) SaveOAuthToken(token OAuthToken) error {
	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal oauth token: %w", err)
	}
	if err := util.WritePrivateFile(c.TokenPath(), append(data, '\n')); err != nil {
		return fmt.Errorf("save oauth token: %w", err)
	}
	return nil
}

// Logout clears the cached OAuth session while keeping the client config.
func (c *AuthClient) Logout() (bool, error) {
	err := os.Remove(c.TokenPath())
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, fmt.Errorf("remove cached oauth token: %w", err)
}

// Status writes a short auth status summary.
func (c *AuthClient) Status(w io.Writer) error {
	hasOAuthClient := util.FileExists(c.ClientPath())
	hasCachedToken := util.FileExists(c.TokenPath())
	loggedIn := false
	if hasCachedToken {
		token, err := c.LoadOAuthToken()
		loggedIn = err == nil && token.AccessToken != "" && (token.Expiry.IsZero() || token.Expiry.After(time.Now()))
	}

	fmt.Fprintln(w, ux.Subtle.Render("Config dir: "+c.ConfigDir))
	fmt.Fprintln(w, ux.Subtle.Render("OAuth client: "+presentLine(hasOAuthClient, c.ClientPath())))
	fmt.Fprintln(w, ux.Subtle.Render("Cached token: "+presentLine(hasCachedToken, c.TokenPath())))

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

	return nil
}

// presentLine formats one status line for an auth file path.
func presentLine(ok bool, path string) string {
	if ok {
		return "present (" + path + ")"
	}
	return "missing (" + path + ")"
}

// loadOAuthClient parses an installed/web OAuth client file from disk.
func loadOAuthClient(path string) (*OAuthClient, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw struct {
		Installed *OAuthClient `json:"installed"`
		Web       *OAuthClient `json:"web"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	switch {
	case raw.Installed != nil:
		return raw.Installed, nil
	case raw.Web != nil:
		return raw.Web, nil
	default:
		return nil, errors.New("unsupported credential file")
	}
}

// loadOAuthToken parses a cached OAuth token file from disk.
func loadOAuthToken(path string) (OAuthToken, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return OAuthToken{}, err
	}

	var token OAuthToken
	if err := json.Unmarshal(data, &token); err != nil {
		return OAuthToken{}, err
	}
	return token, nil
}
