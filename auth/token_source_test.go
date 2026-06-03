package auth

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gurgeous/gshoot/util"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
)

// auth/token_source_test.go covers cached-token loading, refresh, and logout.

// TestLoadOClient parses installed and web OAuth client files.
func TestLoadOClient(t *testing.T) {
	for _, body := range []string{
		`{"installed":{"client_id":"cid","redirect_uris":["http://127.0.0.1/oauth2/callback"]}}`,
	} {
		path := filepath.Join(t.TempDir(), "client.json")
		assert.NoError(t, os.WriteFile(path, []byte(body), 0o600))

		client, err := loadOClient(path)
		assert.NoError(t, err)
		if err == nil {
			assert.Equal(t, "cid", client.ClientID)
			assert.Equal(t, "http://127.0.0.1/oauth2/callback", client.LocalhostRedirect.String())
		}
	}
}

// TestLoadOClientUnsupported rejects unsupported credential JSON.
func TestLoadOClientUnsupported(t *testing.T) {
	path := filepath.Join(t.TempDir(), "client.json")
	assert.NoError(t, os.WriteFile(path, []byte(`{"type":"service_account"}`), 0o600))

	_, err := loadOClient(path)
	assert.Error(t, err)
	if err != nil {
		assert.Contains(t, err.Error(), "missing `installed:`")
	}
}

// TestLoadOAuthToken parses a cached OAuth token file.
func TestLoadOAuthToken(t *testing.T) {
	path := filepath.Join(t.TempDir(), "oauth-token.json")
	assert.NoError(t, os.WriteFile(path, []byte(`{"access_token":"a","refresh_token":"r","token_type":"Bearer","expiry":"2026-05-07T22:00:00Z"}`), 0o600))

	token, err := loadOAuthToken(path)
	assert.NoError(t, err)
	if err == nil {
		assert.Equal(t, "a", token.AccessToken)
		assert.Equal(t, "r", token.RefreshToken)
		assert.Equal(t, "Bearer", token.TokenType)
		assert.Equal(t, time.Date(2026, 5, 7, 22, 0, 0, 0, time.UTC), token.Expiry)
	}
}

func TestLoadOAuthTokenMalformedIncludesPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "oauth-token.json")
	assert.NoError(t, os.WriteFile(path, []byte(`{`), 0o600))

	_, err := loadOAuthToken(path)
	assert.Error(t, err)
	if err != nil {
		assert.Contains(t, err.Error(), path)
		assert.Contains(t, err.Error(), "not JSON")
	}
}

// TestLoadOAuthTokenRejectsAccessOnlyToken rejects non-refreshable token files.
func TestLoadOAuthTokenRejectsAccessOnlyToken(t *testing.T) {
	path := filepath.Join(t.TempDir(), "oauth-token.json")
	assert.NoError(t, os.WriteFile(path, []byte(`{"access_token":"access","token_type":"Bearer","expiry":"2026-05-07T22:00:00Z"}`), 0o600))

	_, err := loadOAuthToken(path)
	assert.Error(t, err)
	if err != nil {
		assert.Contains(t, err.Error(), "missing `refresh_token`")
	}
}

// TestSavingTokenSourceSavesToken checks refreshed tokens are persisted.
func TestSavingTokenSourceSavesToken(t *testing.T) {
	client := withAuthHome(t)
	previous := &oauth2.Token{
		AccessToken:  "expired",
		RefreshToken: "refresh-token",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(-time.Hour),
	}
	writeAuthToken(t, *previous)

	source := &saveTokenSource{
		manager: client,
		src: staticTokenSource{token: &oauth2.Token{
			AccessToken: "refreshed-access",
			TokenType:   "Bearer",
			Expiry:      time.Now().Add(time.Hour),
		}},
		previous: previous,
	}

	token, err := source.Token()
	assert.NoError(t, err)

	assert.Equal(t, "refreshed-access", token.AccessToken)
	assert.Equal(t, "refresh-token", token.RefreshToken)

	saved, err := NewManager()
	assert.NoError(t, err)
	assert.Equal(t, "refreshed-access", saved.token.AccessToken)
	assert.Equal(t, "refresh-token", saved.token.RefreshToken)
}

type staticTokenSource struct {
	token *oauth2.Token
}

func (s staticTokenSource) Token() (*oauth2.Token, error) {
	return s.token, nil
}

// TestLogout removes the cached token file.
func TestLogout(t *testing.T) {
	client := withAuthHome(t)
	writeAuthToken(t, futureToken())
	client.Logout(false)
	assert.False(t, util.FileExists(client.TokenPath))
}

func TestLogoutPurgeDeletesClientSecrets(t *testing.T) {
	client := withAuthHome(t)
	writeClient(t, `{"installed":{"client_id":"cid","redirect_uris":["http://127.0.0.1/oauth2/callback"]}}`)
	writeAuthToken(t, futureToken())

	client.Logout(true)

	assert.False(t, util.FileExists(client.TokenPath))
	assert.False(t, util.FileExists(client.ClientPath))
}
