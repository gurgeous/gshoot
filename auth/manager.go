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

// auth/manager.go owns on-disk browser auth state and status reporting.

// Manager manages browser auth files under one config directory.
type Manager struct {
	ConfigDir string
}

// NewManager builds an auth manager for the default config directory.
func NewManager() *Manager {
	return &Manager{ConfigDir: util.ConfigDir()}
}

//
// paths
//

// ClientPath returns the oauth-client.json path.
func (m *Manager) ClientPath() string {
	return filepath.Join(m.ConfigDir, "oauth-client.json")
}

// TokenPath returns the oauth-token.json path.
func (m *Manager) TokenPath() string {
	return filepath.Join(m.ConfigDir, "oauth-token.json")
}

//
// load from disk
//

// OClient is an installed/web OAuth client config.
type OClient struct {
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	AuthURI      string   `json:"auth_uri"`
	TokenURI     string   `json:"token_uri"`
	RedirectURIs []string `json:"redirect_uris"`
}

// OAuthToken is cached OAuth token state.
type OAuthToken struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	Expiry       time.Time `json:"expiry"`
}

// LoadOClient reads the saved OAuth client config.
func (m *Manager) LoadOClient() (*OClient, error) {
	return loadOClient(m.ClientPath())
}

// LoadOAuthToken reads the saved OAuth token.
func (m *Manager) LoadOAuthToken() (OAuthToken, error) {
	return loadOAuthToken(m.TokenPath())
}

//
// save to disk
//

// SaveOAuthToken writes the saved OAuth token.
func (m *Manager) SaveOAuthToken(token OAuthToken) error {
	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal oauth token: %w", err)
	}
	if err := util.WritePrivateFile(m.TokenPath(), append(data, '\n')); err != nil {
		return fmt.Errorf("save oauth token: %w", err)
	}
	return nil
}

// ImportOClient validates and saves a downloaded client JSON file.
func (m *Manager) ImportOClient(srcPath string) error {
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("read client secret file: %w", err)
	}
	_, err = loadOClient(srcPath)
	if err != nil {
		return fmt.Errorf("load client secret file: %w", err)
	}
	if err := util.WritePrivateFile(m.ClientPath(), data); err != nil {
		return fmt.Errorf("save oauth client config: %w", err)
	}
	return nil
}

//
// logout (delete token)
//

// Logout clears the cached OAuth token while keeping the client config.
func (m *Manager) Logout() {
	os.Remove(m.TokenPath())
}

//
// dump status to writer
//

// Status writes a short auth status summary.
func (m *Manager) Status(w io.Writer) error {
	hasOClient := util.FileExists(m.ClientPath())
	hasCachedToken := util.FileExists(m.TokenPath())
	loggedIn := false
	if hasCachedToken {
		token, err := m.LoadOAuthToken()
		loggedIn = err == nil && token.AccessToken != "" && (token.Expiry.IsZero() || token.Expiry.After(time.Now()))
	}

	fmt.Fprintln(w, ux.Subtle.Render("Config dir: "+m.ConfigDir))
	fmt.Fprintln(w, ux.Subtle.Render("OAuth client: "+presentLine(hasOClient, m.ClientPath())))
	fmt.Fprintln(w, ux.Subtle.Render("Cached token: "+presentLine(hasCachedToken, m.TokenPath())))

	switch {
	case loggedIn:
		fmt.Fprintln(w, ux.Success.Render("Status: logged in"))
	case hasOClient || hasCachedToken:
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

//
// low-level helpers for parsing our files
//

// loadOClient parses an installed/web OAuth client file from disk.
func loadOClient(path string) (*OClient, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw struct {
		Installed *OClient `json:"installed"`
		Web       *OClient `json:"web"`
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
