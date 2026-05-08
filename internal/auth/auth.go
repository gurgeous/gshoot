package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
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

// Env houses the environment variables gshoot uses.
type Env struct {
	values map[string]string
}

// NewEnv builds an Env from explicit values.
func NewEnv(values map[string]string) Env {
	return Env{values: values}
}

// Lookup returns an environment value.
func (e Env) Lookup(key string) string {
	if e.values != nil {
		return e.values[key]
	}
	return os.Getenv(key)
}

func (e Env) ConfigDirOverride() string {
	return e.Lookup("GSHOOT_CONFIG_DIR")
}

func (e Env) Token() string {
	return e.Lookup("GSHOOT_TOKEN")
}

func (e Env) CredentialsFile() string {
	return e.Lookup("GSHOOT_CREDENTIALS_FILE")
}

func (e Env) ApplicationCredentials() string {
	return e.Lookup("GOOGLE_APPLICATION_CREDENTIALS")
}

func (e Env) XDGConfigHome() string {
	return e.Lookup("XDG_CONFIG_HOME")
}

func (e Env) Home() string {
	if home := e.Lookup("HOME"); home != "" {
		return home
	}
	if home, err := os.UserHomeDir(); err == nil {
		return home
	}
	return "."
}

// Options configures auth resolution.
type Options struct {
	Env     Env
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
func ConfigDir(env Env) string {
	if dir := env.ConfigDirOverride(); dir != "" {
		return dir
	}
	if dir := env.XDGConfigHome(); dir != "" {
		return filepath.Join(dir, "gshoot")
	}
	return filepath.Join(env.Home(), ".config", "gshoot")
}

// Resolve picks the best auth source for a command.
func Resolve(opts Options) (Resolved, error) {
	configDir := ConfigDir(opts.Env)
	oauthClientPath := filepath.Join(configDir, oauthClientFileName)
	oauthTokenPath := filepath.Join(configDir, oauthTokenFileName)
	scopes := ScopesForCommand(opts.Command)
	if len(scopes) == 0 {
		return Resolved{}, fmt.Errorf("unknown command for auth scopes: %q", opts.Command)
	}

	if token := opts.Env.Token(); token != "" {
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

	if path := opts.Env.CredentialsFile(); path != "" {
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

	if path := opts.Env.ApplicationCredentials(); path != "" {
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

	adcPath := filepath.Join(opts.Env.Home(), ".config", "gcloud", "application_default_credentials.json")
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
		"no auth found; checked env vars $GSHOOT_TOKEN, $GSHOOT_CREDENTIALS_FILE, $GOOGLE_APPLICATION_CREDENTIALS; checked files %s, %s",
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
