package auth

import (
	"encoding/json"
	"errors"
	"os"
)

// CredentialKind describes the credential file shape.
type CredentialKind string

const (
	CredentialKindAuthorizedUser CredentialKind = "authorized_user"
	CredentialKindOAuthClient    CredentialKind = "oauth_client"
	CredentialKindServiceAccount CredentialKind = "service_account"
)

// CredentialFile is a parsed credential file.
type CredentialFile struct {
	Kind           CredentialKind
	Path           string
	AuthorizedUser *AuthorizedUser
	ServiceAccount *ServiceAccount
	OAuthClient    *OAuthClient
}

// AuthorizedUser represents an authorized-user credentials file.
type AuthorizedUser struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RefreshToken string `json:"refresh_token"`
	TokenURI     string `json:"token_uri"`
	Type         string `json:"type"`
}

// ServiceAccount represents a service-account credentials file.
type ServiceAccount struct {
	Type         string `json:"type"`
	ClientEmail  string `json:"client_email"`
	PrivateKey   string `json:"private_key"`
	PrivateKeyID string `json:"private_key_id"`
	TokenURI     string `json:"token_uri"`
}

// OAuthClient is an installed/web OAuth client config.
type OAuthClient struct {
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	AuthURI      string   `json:"auth_uri"`
	TokenURI     string   `json:"token_uri"`
	RedirectURIs []string `json:"redirect_uris"`
}

// LoadCredentialFile parses a credentials file.
func LoadCredentialFile(path string) (CredentialFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return CredentialFile{}, err
	}
	return parseCredentialFile(path, data)
}

func parseCredentialFile(path string, data []byte) (CredentialFile, error) {
	var raw struct {
		Type string `json:"type"`

		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
		RefreshToken string `json:"refresh_token"`

		ClientEmail  string `json:"client_email"`
		PrivateKey   string `json:"private_key"`
		PrivateKeyID string `json:"private_key_id"`
		TokenURI     string `json:"token_uri"`

		Installed *OAuthClient `json:"installed"`
		Web       *OAuthClient `json:"web"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return CredentialFile{}, err
	}

	switch {
	case raw.Installed != nil:
		return CredentialFile{
			Kind:        CredentialKindOAuthClient,
			Path:        path,
			OAuthClient: raw.Installed,
		}, nil
	case raw.Web != nil:
		return CredentialFile{
			Kind:        CredentialKindOAuthClient,
			Path:        path,
			OAuthClient: raw.Web,
		}, nil
	case raw.Type == "authorized_user":
		return CredentialFile{
			Kind: CredentialKindAuthorizedUser,
			Path: path,
			AuthorizedUser: &AuthorizedUser{
				ClientID:     raw.ClientID,
				ClientSecret: raw.ClientSecret,
				RefreshToken: raw.RefreshToken,
				TokenURI:     raw.TokenURI,
				Type:         raw.Type,
			},
		}, nil
	case raw.Type == "service_account":
		return CredentialFile{
			Kind: CredentialKindServiceAccount,
			Path: path,
			ServiceAccount: &ServiceAccount{
				Type:         raw.Type,
				ClientEmail:  raw.ClientEmail,
				PrivateKey:   raw.PrivateKey,
				PrivateKeyID: raw.PrivateKeyID,
				TokenURI:     raw.TokenURI,
			},
		}, nil
	default:
		return CredentialFile{}, errors.New("unsupported credential file")
	}
}
