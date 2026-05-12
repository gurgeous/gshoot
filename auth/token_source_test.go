package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gurgeous/gshoot/util"
	"github.com/stretchr/testify/assert"
)

// auth/token_source_test.go covers cached-token loading, refresh, and logout.

// TestLoadOClient parses installed and web OAuth client files.
func TestLoadOClient(t *testing.T) {
	for _, body := range []string{
		`{"installed":{"client_id":"cid"}}`,
		`{"web":{"client_id":"cid"}}`,
	} {
		path := filepath.Join(t.TempDir(), "client.json")
		assert.NoError(t, os.WriteFile(path, []byte(body), 0o600))

		client, err := loadOClient(path)
		assert.NoError(t, err)
		if err == nil {
			assert.Equal(t, "cid", client.ClientID)
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
		assert.Contains(t, err.Error(), "unsupported credential file")
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

// TestNewTokenSourceMissingAuth reports a missing auth setup clearly.
func TestNewTokenSourceMissingAuth(t *testing.T) {
	client := withAuthHome(t)

	_, err := client.TokenSource(context.Background(), []string{"scope"})
	assert.Error(t, err)
	if err != nil {
		assert.Contains(t, err.Error(), "no auth found")
	}
}

// TestNewTokenSourceValidCachedOAuthWithoutClientConfig uses a still-valid cached token directly.
func TestNewTokenSourceValidCachedOAuthWithoutClientConfig(t *testing.T) {
	client := withAuthHome(t)
	writeAuthToken(t, futureToken())

	src, err := client.TokenSource(context.Background(), []string{"scope"})
	assert.NoError(t, err)
	if err != nil {
		return
	}

	token, err := src.Token()
	assert.NoError(t, err)
	if err == nil {
		assert.Equal(t, "access", token.AccessToken)
	}
}

// TestNewTokenSourceExpiredCachedOAuthWithoutClientConfig rejects expired cached tokens without a client file.
func TestNewTokenSourceExpiredCachedOAuthWithoutClientConfig(t *testing.T) {
	client := withAuthHome(t)
	writeAuthToken(t, OAuthToken{
		AccessToken:  "expired",
		RefreshToken: "refresh",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(-time.Hour),
	})

	_, err := client.TokenSource(context.Background(), []string{"scope"})
	assert.Error(t, err)
	if err != nil {
		assert.Contains(t, err.Error(), "expired")
		assert.Contains(t, err.Error(), "oauth-client.json")
	}
}

// TestNewTokenSourceRefreshesCachedOAuth checks refresh through the token endpoint.
func TestNewTokenSourceRefreshesCachedOAuth(t *testing.T) {
	client := withAuthHome(t)

	var tokenEndpointHit bool
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenEndpointHit = true
		assert.NoError(t, r.ParseForm())
		assert.Equal(t, "refresh_token", r.Form.Get("grant_type"))
		assert.Equal(t, "refresh-token", r.Form.Get("refresh_token"))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "refreshed",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	}))
	defer tokenServer.Close()

	writeClient(t, `{"installed":{"client_id":"cid","client_secret":"secret","auth_uri":"https://accounts.google.com/o/oauth2/auth","token_uri":"`+tokenServer.URL+`","redirect_uris":["http://127.0.0.1/oauth2/callback"]}}`)
	writeAuthToken(t, OAuthToken{
		AccessToken:  "expired",
		RefreshToken: "refresh-token",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(-time.Hour),
	})

	src, err := client.TokenSource(context.Background(), []string{"scope"})
	assert.NoError(t, err)
	if err != nil {
		return
	}

	token, err := src.Token()
	assert.NoError(t, err)
	if err == nil {
		assert.True(t, tokenEndpointHit)
		assert.Equal(t, "refreshed", token.AccessToken)
	}
}

// TestNewTokenSourceCachedOAuthRefreshFailure surfaces token refresh failures.
func TestNewTokenSourceCachedOAuthRefreshFailure(t *testing.T) {
	client := withAuthHome(t)

	var tokenEndpointHit bool
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenEndpointHit = true
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_grant","error_description":"Token has been expired or revoked."}`))
	}))
	defer tokenServer.Close()

	writeClient(t, `{"installed":{"client_id":"cid","client_secret":"secret","auth_uri":"https://accounts.google.com/o/oauth2/auth","token_uri":"`+tokenServer.URL+`","redirect_uris":["http://127.0.0.1/oauth2/callback"]}}`)
	writeAuthToken(t, OAuthToken{
		AccessToken:  "expired",
		RefreshToken: "refresh-token",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(-time.Hour),
	})

	src, err := client.TokenSource(context.Background(), []string{"scope"})
	assert.NoError(t, err)
	if err != nil {
		return
	}

	_, err = src.Token()
	assert.Error(t, err)
	assert.True(t, tokenEndpointHit)
	if err != nil {
		assert.Contains(t, err.Error(), "invalid_grant")
		assert.Contains(t, err.Error(), "expired or revoked")
	}
}

// TestLogout removes the cached token file.
func TestLogout(t *testing.T) {
	client := withAuthHome(t)
	writeAuthToken(t, futureToken())
	client.Logout()
	assert.False(t, util.FileExists(client.TokenPath()))
}

// TestLogoutMissingToken is a no-op when there is no cached token.
func TestLogoutMissingToken(t *testing.T) {
	client := withAuthHome(t)
	client.Logout()
	assert.False(t, util.FileExists(client.TokenPath()))
}
