package auth

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/gurgeous/gshoot/env"
)

const (
	oauthClientFileName = "oauth-client.json"
	oauthTokenFileName  = "oauth-token.json"
)

// SourceKind describes where auth material came from.
type SourceKind string

const (
	SourceKindRawToken                      SourceKind = "raw_token"
	SourceKindCredentialsFile               SourceKind = "credentials_file"
	SourceKindCachedOAuth                   SourceKind = "cached_oauth"
	SourceKindApplicationDefaultCredentials SourceKind = "application_default_credentials"
)

// Resolved describes the chosen auth source and supporting paths.
type Resolved struct {
	ConfigDir       string
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

// NoAuthError reports that no usable auth source exists.
type NoAuthError struct{}

func (e *NoAuthError) Error() string {
	return "gshoot [no auth found]\n"
}

// ConfigDir returns the config directory for gshoot.
func ConfigDir() string {
	if dir := env.GSHOOT_CONFIG_DIR(); dir != "" {
		return dir
	}
	return filepath.Join(xdg.ConfigHome, "gshoot")
}

// Resolve picks the best available auth source.
func Resolve() (Resolved, error) {
	configDir := ConfigDir()
	oauthClientPath := filepath.Join(configDir, oauthClientFileName)
	oauthTokenPath := filepath.Join(configDir, oauthTokenFileName)

	if token := env.GSHOOT_TOKEN(); token != "" {
		return Resolved{
			ConfigDir:       configDir,
			OAuthClientPath: oauthClientPath,
			OAuthTokenPath:  oauthTokenPath,
			Source: Source{
				Kind:  SourceKindRawToken,
				Token: token,
			},
		}, nil
	}

	if path := env.GSHOOT_CREDENTIALS_FILE(); path != "" {
		cred, err := LoadCredentialFile(path)
		if err != nil {
			return Resolved{}, fmt.Errorf("load $GSHOOT_CREDENTIALS_FILE: %w", err)
		}
		return Resolved{
			ConfigDir:       configDir,
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

	if path := env.GOOGLE_APPLICATION_CREDENTIALS(); path != "" {
		cred, loadErr := LoadCredentialFile(path)
		if loadErr != nil {
			return Resolved{}, fmt.Errorf("load $GOOGLE_APPLICATION_CREDENTIALS: %w", loadErr)
		}
		return Resolved{
			ConfigDir:       configDir,
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
	adcPath := filepath.Join(xdg.ConfigHome, "gcloud", "application_default_credentials.json")
	cred, err := LoadCredentialFile(adcPath)
	if err == nil {
		return Resolved{
			ConfigDir:       configDir,
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
		&NoAuthError{},
		oauthTokenPath,
		adcPath,
	)
}
