package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/gurgeous/gshoot/util"
	"golang.org/x/oauth2"
)

//
// Manager manages auth flow, client secrets and token
//

var (
	AuthReadmeURL = "https://github.com/gurgeous/gshoot#authentication"
	Scopes        = []string{
		"https://www.googleapis.com/auth/drive",
		"https://www.googleapis.com/auth/spreadsheets",
	}
)

type Manager struct {
	ClientPath string // saved OAuth client JSON path
	TokenPath  string // saved OAuth token JSON path

	// internal
	client *OClient
	token  *oauth2.Token
}

// NewManager builds an auth manager for the default config directory.
func NewManager() (*Manager, error) {
	dir := util.ConfigDir()
	m := &Manager{
		ClientPath: filepath.Join(dir, "oauth-client.json"),
		TokenPath:  filepath.Join(dir, "oauth-token.json"),
	}

	// load client/token. there are uncommon edge cases where the files exist but
	// are invalid for some reasonm, so handle errors carefully
	var err error
	m.client, err = loadOClient(m.ClientPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	m.token, err = loadOAuthToken(m.TokenPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	return m, nil
}

// does this manager have client secrets already?
func (m *Manager) HasClientSecrets() bool {
	return m.client != nil
}

func (m *Manager) LoggedIn() bool {
	return m.client != nil && m.token != nil
}

//
// save
//

func (m *Manager) SaveOClient(srcPath string) error {
	var err error

	var data []byte
	if data, err = os.ReadFile(srcPath); err != nil {
		return err
	}
	client, err := loadOClient(srcPath)
	if err != nil {
		return err
	}
	if err := util.WritePrivateFile(m.ClientPath, data); err != nil {
		return err
	}
	m.client = client

	return nil
}

func (m *Manager) SaveOAuthToken(token *oauth2.Token) error {
	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return err
	}
	if _, err := parseOAuthToken(data); err != nil {
		return err
	}
	if err := util.WritePrivateFile(m.TokenPath, append(data, '\n')); err != nil {
		return err
	}
	m.token = token
	return nil
}

//
// low-level helpers for parsing our files
//

func loadOClient(path string) (*OClient, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	client, err := parseOClient(data)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	return client, nil
}

func loadOAuthToken(path string) (*oauth2.Token, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	token, err := parseOAuthToken(data)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	return token, nil
}

func parseOClient(data []byte) (*OClient, error) {
	var raw struct {
		Installed *OClient `json:"installed"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, errors.New("not JSON") // uncommon
	}

	// maybe they set it up as Web by mistake?
	client := raw.Installed
	if client == nil {
		return nil, errors.New("missing `installed:`") // common?
	}

	redirect, err := findLocalhostRedirect(client.RedirectURIs)
	if err != nil {
		return nil, errors.New("no localhost/127.0.0.1 redirect") // uncommon
	}
	client.LocalhostRedirect = redirect

	return client, nil
}

func parseOAuthToken(data []byte) (*oauth2.Token, error) {
	var token oauth2.Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, errors.New("not JSON") // uncommon
	}
	if token.AccessToken == "" {
		return nil, errors.New("missing `access_token`") // uncommon
	}
	if token.RefreshToken == "" {
		return nil, errors.New("missing `refresh_token`") // uncommon
	}
	if token.Expiry.IsZero() {
		return nil, errors.New("missing `expiry`") // uncommon
	}
	return &token, nil
}

// findLocalhostRedirect picks the first localhost redirect URI from the client config.
func findLocalhostRedirect(redirectURIs []string) (*url.URL, error) {
	for _, raw := range redirectURIs {
		u, err := url.Parse(raw)
		if err != nil {
			return nil, err
		}
		if u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" {
			if u.Path == "" {
				u.Path = "/"
			}
			return u, nil
		}
	}
	return nil, errors.New("client secrets JSON needs a localhost or 127.0.0.1 redirect URI")
}

// OClient is an installed OAuth client config from Google client secrets
type OClient struct {
	ClientID          string   `json:"client_id"`     // Google OAuth client id
	ClientSecret      string   `json:"client_secret"` // Google OAuth client secret
	RedirectURIs      []string `json:"redirect_uris"` // redirect URIs from the client JSON
	LocalhostRedirect *url.URL `json:"-"`             // validated localhost redirect selected at load time
}
