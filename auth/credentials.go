package auth

import (
	"encoding/json"
	"errors"
	"os"
)

// OAuthClient is an installed/web OAuth client config.
type OAuthClient struct {
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	AuthURI      string   `json:"auth_uri"`
	TokenURI     string   `json:"token_uri"`
	RedirectURIs []string `json:"redirect_uris"`
}

// LoadOAuthClient parses an installed/web OAuth client file.
func LoadOAuthClient(path string) (*OAuthClient, error) {
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
