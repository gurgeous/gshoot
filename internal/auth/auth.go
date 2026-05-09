package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/adrg/xdg"
	"github.com/gurgeous/gshoot/internal/env"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
)

const (
	oauthClientFileName = "oauth-client.json"
	oauthTokenFileName  = "oauth-token.json"
)

// Command identifies a gshoot subcommand for scope selection.
type Command string

const (
	CommandUp   Command = "up"
	CommandDown Command = "down"
	CommandList Command = "list"
)

// SourceKind describes where auth material came from.
type SourceKind string

const (
	SourceKindRawToken                      SourceKind = "raw_token"
	SourceKindCredentialsFile               SourceKind = "credentials_file"
	SourceKindCachedOAuth                   SourceKind = "cached_oauth"
	SourceKindApplicationDefaultCredentials SourceKind = "application_default_credentials"
)

// CredentialKind describes the credential file shape.
type CredentialKind string

const (
	CredentialKindAuthorizedUser CredentialKind = "authorized_user"
	CredentialKindOAuthClient    CredentialKind = "oauth_client"
	CredentialKindServiceAccount CredentialKind = "service_account"
)

// Options configures auth resolution.
type Options struct {
	Command Command
}

// Resolved describes the chosen auth source and supporting paths.
type Resolved struct {
	ConfigDir       string
	Scopes          []string
	Source          Source
	OAuthClientPath string
	OAuthTokenPath  string
}

// Source identifies one auth source.
type Source struct {
	Kind           SourceKind
	Path           string
	Token          string
	CredentialFile *CredentialFile
	OAuthToken     *OAuthToken
}

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

// OAuthToken is cached OAuth token state.
type OAuthToken struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	Expiry       time.Time `json:"expiry"`
}

// ScopesForCommand returns Google API scopes for a CLI command.
func ScopesForCommand(cmd Command) []string {
	switch cmd {
	case CommandUp:
		return []string{
			"https://www.googleapis.com/auth/drive",
			"https://www.googleapis.com/auth/spreadsheets",
		}
	case CommandDown, CommandList:
		return []string{
			"https://www.googleapis.com/auth/drive.readonly",
			"https://www.googleapis.com/auth/spreadsheets.readonly",
		}
	default:
		return nil
	}
}

// ConfigDir returns the config directory for gshoot.
func ConfigDir() string {
	if dir := env.GSHOOT_CONFIG_DIR; dir != "" {
		return dir
	}
	xdg.Reload()
	return filepath.Join(xdg.ConfigHome, "gshoot")
}

// Resolve picks the best auth source for a command.
func Resolve(opts Options) (Resolved, error) {
	configDir := ConfigDir()
	oauthClientPath := filepath.Join(configDir, oauthClientFileName)
	oauthTokenPath := filepath.Join(configDir, oauthTokenFileName)
	scopes := ScopesForCommand(opts.Command)
	if len(scopes) == 0 {
		return Resolved{}, fmt.Errorf("unknown command for auth scopes: %q", opts.Command)
	}

	if token := env.GSHOOT_TOKEN; token != "" {
		return Resolved{
			ConfigDir:       configDir,
			Scopes:          scopes,
			OAuthClientPath: oauthClientPath,
			OAuthTokenPath:  oauthTokenPath,
			Source: Source{
				Kind:  SourceKindRawToken,
				Token: token,
			},
		}, nil
	}

	if path := env.GSHOOT_CREDENTIALS_FILE; path != "" {
		cred, err := LoadCredentialFile(path)
		if err != nil {
			return Resolved{}, fmt.Errorf("load $GSHOOT_CREDENTIALS_FILE: %w", err)
		}
		return Resolved{
			ConfigDir:       configDir,
			Scopes:          scopes,
			OAuthClientPath: oauthClientPath,
			OAuthTokenPath:  oauthTokenPath,
			Source: Source{
				Kind:           SourceKindCredentialsFile,
				Path:           path,
				CredentialFile: &cred,
			},
		}, nil
	}

	token, err := LoadOAuthToken(oauthTokenPath)
	if err == nil {
		return Resolved{
			ConfigDir:       configDir,
			Scopes:          scopes,
			OAuthClientPath: oauthClientPath,
			OAuthTokenPath:  oauthTokenPath,
			Source: Source{
				Kind:       SourceKindCachedOAuth,
				Path:       oauthTokenPath,
				OAuthToken: &token,
			},
		}, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return Resolved{}, fmt.Errorf("load cached oauth token: %w", err)
	}

	if path := env.GOOGLE_APPLICATION_CREDENTIALS; path != "" {
		cred, loadErr := LoadCredentialFile(path)
		if loadErr != nil {
			return Resolved{}, fmt.Errorf("load $GOOGLE_APPLICATION_CREDENTIALS: %w", loadErr)
		}
		return Resolved{
			ConfigDir:       configDir,
			Scopes:          scopes,
			OAuthClientPath: oauthClientPath,
			OAuthTokenPath:  oauthTokenPath,
			Source: Source{
				Kind:           SourceKindApplicationDefaultCredentials,
				Path:           path,
				CredentialFile: &cred,
			},
		}, nil
	}

	xdg.Reload()
	adcPath := filepath.Join(xdg.Home, ".config", "gcloud", "application_default_credentials.json")
	cred, err := LoadCredentialFile(adcPath)
	if err == nil {
		return Resolved{
			ConfigDir:       configDir,
			Scopes:          scopes,
			OAuthClientPath: oauthClientPath,
			OAuthTokenPath:  oauthTokenPath,
			Source: Source{
				Kind:           SourceKindApplicationDefaultCredentials,
				Path:           adcPath,
				CredentialFile: &cred,
			},
		}, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return Resolved{}, fmt.Errorf("load application default credentials: %w", err)
	}

	return Resolved{}, fmt.Errorf(
		"%w\nchecked env vars: $GSHOOT_TOKEN, $GSHOOT_CREDENTIALS_FILE, $GOOGLE_APPLICATION_CREDENTIALS\nchecked files: %s, %s",
		&NoAuthError{Command: opts.Command, ConfigDir: configDir},
		oauthTokenPath,
		adcPath,
	)
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

// LoadOAuthToken parses cached OAuth token state.
func LoadOAuthToken(path string) (OAuthToken, error) {
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

// NewTokenSource creates an oauth2 token source from resolved auth state.
func NewTokenSource(ctx context.Context, resolved Resolved) (oauth2.TokenSource, error) {
	switch resolved.Source.Kind {
	case SourceKindRawToken:
		return oauth2.StaticTokenSource(&oauth2.Token{
			AccessToken: resolved.Source.Token,
			TokenType:   "Bearer",
		}), nil
	case SourceKindCredentialsFile, SourceKindApplicationDefaultCredentials:
		if resolved.Source.CredentialFile == nil {
			return nil, errors.New("missing parsed credential file")
		}
		return tokenSourceFromCredentialFile(ctx, resolved.Source.CredentialFile, resolved.Scopes)
	case SourceKindCachedOAuth:
		if resolved.Source.OAuthToken == nil {
			return nil, errors.New("missing parsed oauth token")
		}

		token := &oauth2.Token{
			AccessToken:  resolved.Source.OAuthToken.AccessToken,
			RefreshToken: resolved.Source.OAuthToken.RefreshToken,
			TokenType:    resolved.Source.OAuthToken.TokenType,
			Expiry:       resolved.Source.OAuthToken.Expiry,
		}

		client, err := LoadCredentialFile(resolved.OAuthClientPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				if !token.Valid() {
					return nil, errors.New("cached oauth token is expired and oauth-client.json is missing")
				}
				return oauth2.StaticTokenSource(token), nil
			}
			return nil, fmt.Errorf("load oauth client config: %w", err)
		}
		if client.Kind != CredentialKindOAuthClient || client.OAuthClient == nil {
			return nil, errors.New("oauth client config is not an oauth client")
		}

		config := &oauth2.Config{
			ClientID:     client.OAuthClient.ClientID,
			ClientSecret: client.OAuthClient.ClientSecret,
			Endpoint:     google.Endpoint,
			Scopes:       resolved.Scopes,
			RedirectURL:  firstRedirectURI(client.OAuthClient.RedirectURIs),
		}
		return config.TokenSource(ctx, token), nil
	default:
		return nil, fmt.Errorf("unsupported auth source: %q", resolved.Source.Kind)
	}
}

func tokenSourceFromCredentialFile(ctx context.Context, cred *CredentialFile, scopes []string) (oauth2.TokenSource, error) {
	switch cred.Kind {
	case CredentialKindAuthorizedUser:
		if cred.AuthorizedUser == nil {
			return nil, errors.New("missing authorized user credentials")
		}
		config := &oauth2.Config{
			ClientID:     cred.AuthorizedUser.ClientID,
			ClientSecret: cred.AuthorizedUser.ClientSecret,
			Endpoint:     google.Endpoint,
			Scopes:       scopes,
		}
		token := &oauth2.Token{
			RefreshToken: cred.AuthorizedUser.RefreshToken,
		}
		return config.TokenSource(ctx, token), nil
	case CredentialKindServiceAccount:
		if cred.ServiceAccount == nil {
			return nil, errors.New("missing service account credentials")
		}
		config := &jwt.Config{
			Email:      cred.ServiceAccount.ClientEmail,
			PrivateKey: []byte(cred.ServiceAccount.PrivateKey),
			TokenURL:   google.JWTTokenURL,
			Scopes:     scopes,
		}
		if cred.ServiceAccount.TokenURI != "" {
			config.TokenURL = cred.ServiceAccount.TokenURI
		}
		return config.TokenSource(ctx), nil
	case CredentialKindOAuthClient:
		return nil, errors.New("oauth client config requires browser login")
	default:
		return nil, fmt.Errorf("unsupported credential kind: %q", cred.Kind)
	}
}

func firstRedirectURI(redirectURIs []string) string {
	if len(redirectURIs) == 0 {
		return ""
	}
	return redirectURIs[0]
}
