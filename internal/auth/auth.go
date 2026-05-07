package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	configDirEnvName         = "GSHOOT_CONFIG_DIR"
	tokenEnvName             = "GSHOOT_TOKEN"
	credentialsFileEnvName   = "GSHOOT_CREDENTIALS_FILE"
	applicationCredsEnvName  = "GOOGLE_APPLICATION_CREDENTIALS"
	oauthClientFileName      = "oauth-client.json"
	oauthTokenFileName       = "oauth-token.json"
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

// Env provides environment lookup for auth resolution.
type Env map[string]string

// Options configures auth resolution.
type Options struct {
	Env     Env
	Command Command
}

// Resolved describes the chosen auth source and supporting paths.
type Resolved struct {
	ConfigDir      string
	Scopes         []string
	Source         Source
	OAuthClientPath string
	OAuthTokenPath  string
}

// Source identifies one auth source.
type Source struct {
	Kind  SourceKind
	Path  string
	Token string
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
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	Expiry       string `json:"expiry"`
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
	if dir := envLookup(env, configDirEnvName); dir != "" {
		return dir
	}
	if dir := envLookup(env, "XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "gshoot")
	}
	return filepath.Join(homeDir(env), ".config", "gshoot")
}

// Resolve picks the best auth source for a command.
func Resolve(opts Options) (Resolved, error) {
	configDir := ConfigDir(opts.Env)
	oauthClientPath := filepath.Join(configDir, oauthClientFileName)
	oauthTokenPath := filepath.Join(configDir, oauthTokenFileName)
	scopes := ScopesForCommand(opts.Command)

	if token := envLookup(opts.Env, tokenEnvName); token != "" {
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

	if path := envLookup(opts.Env, credentialsFileEnvName); path != "" {
		if _, err := LoadCredentialFile(path); err != nil {
			return Resolved{}, fmt.Errorf("load %s: %w", credentialsFileEnvName, err)
		}
		return Resolved{
			ConfigDir:       configDir,
			Scopes:          scopes,
			OAuthClientPath: oauthClientPath,
			OAuthTokenPath:  oauthTokenPath,
			Source: Source{
				Kind: SourceKindCredentialsFile,
				Path: path,
			},
		}, nil
	}

	if fileExists(oauthTokenPath) {
		if _, err := LoadOAuthToken(oauthTokenPath); err != nil {
			return Resolved{}, fmt.Errorf("load cached oauth token: %w", err)
		}
		return Resolved{
			ConfigDir:       configDir,
			Scopes:          scopes,
			OAuthClientPath: oauthClientPath,
			OAuthTokenPath:  oauthTokenPath,
			Source: Source{
				Kind: SourceKindCachedOAuth,
				Path: oauthTokenPath,
			},
		}, nil
	}

	if path := envLookup(opts.Env, applicationCredsEnvName); path != "" {
		if _, err := LoadCredentialFile(path); err != nil {
			return Resolved{}, fmt.Errorf("load %s: %w", applicationCredsEnvName, err)
		}
		return Resolved{
			ConfigDir:       configDir,
			Scopes:          scopes,
			OAuthClientPath: oauthClientPath,
			OAuthTokenPath:  oauthTokenPath,
			Source: Source{
				Kind: SourceKindApplicationDefaultCredentials,
				Path: path,
			},
		}, nil
	}

	adcPath := filepath.Join(homeDir(opts.Env), ".config", "gcloud", "application_default_credentials.json")
	if fileExists(adcPath) {
		if _, err := LoadCredentialFile(adcPath); err != nil {
			return Resolved{}, fmt.Errorf("load application default credentials: %w", err)
		}
		return Resolved{
			ConfigDir:       configDir,
			Scopes:          scopes,
			OAuthClientPath: oauthClientPath,
			OAuthTokenPath:  oauthTokenPath,
			Source: Source{
				Kind: SourceKindApplicationDefaultCredentials,
				Path: adcPath,
			},
		}, nil
	}

	checked := []string{
		tokenEnvName,
		credentialsFileEnvName,
		oauthTokenPath,
		applicationCredsEnvName,
		adcPath,
	}
	return Resolved{}, fmt.Errorf("no auth found; checked %s", strings.Join(checked, ", "))
}

// LoadCredentialFile parses a credentials file.
func LoadCredentialFile(path string) (CredentialFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return CredentialFile{}, err
	}

	var raw struct {
		Type      string       `json:"type"`
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
		var cred AuthorizedUser
		if err := json.Unmarshal(data, &cred); err != nil {
			return CredentialFile{}, err
		}
		return CredentialFile{
			Kind:           CredentialKindAuthorizedUser,
			Path:           path,
			AuthorizedUser: &cred,
		}, nil
	case raw.Type == "service_account":
		var cred ServiceAccount
		if err := json.Unmarshal(data, &cred); err != nil {
			return CredentialFile{}, err
		}
		return CredentialFile{
			Kind:           CredentialKindServiceAccount,
			Path:           path,
			ServiceAccount: &cred,
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

func envLookup(env Env, key string) string {
	if env != nil {
		return env[key]
	}
	return os.Getenv(key)
}

func homeDir(env Env) string {
	if home := envLookup(env, "HOME"); home != "" {
		return home
	}
	if home, err := os.UserHomeDir(); err == nil {
		return home
	}
	return "."
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
