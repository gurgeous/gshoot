package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/adrg/xdg"
	"github.com/gurgeous/gshoot/util"
	"github.com/stretchr/testify/assert"
)

// auth/login_test.go covers the browser login flow and login-specific helpers.

// TestLoginMissingClientSecretGuidance checks the setup hint when no client JSON exists.
func TestLoginMissingClientSecretGuidance(t *testing.T) {
	client := withAuthHome(t)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := client.Login(context.Background(), LoginOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	})

	assert.Error(t, err)
	if err != nil {
		assert.Contains(t, err.Error(), "oauth-client.json")
		assert.Contains(t, err.Error(), "Desktop app")
		assert.Contains(t, err.Error(), "Test users")
		assert.Contains(t, err.Error(), "gshoot auth login --client-secret")
	}
}

// TestLoginImportsClientAndSavesToken checks the happy-path browser login flow.
func TestLoginImportsClientAndSavesToken(t *testing.T) {
	client := withAuthHome(t)

	var tokenEndpointHit bool
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenEndpointHit = true
		assert.NoError(t, r.ParseForm())
		assert.Equal(t, "test-code", r.Form.Get("code"))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "access",
			"refresh_token": "refresh",
			"token_type":    "Bearer",
			"expires_in":    3600,
		})
	}))
	defer tokenServer.Close()

	clientSecretPath := filepath.Join(t.TempDir(), "client_secret.json")
	assert.NoError(t, os.WriteFile(clientSecretPath, []byte(`{"installed":{"client_id":"cid","client_secret":"secret","auth_uri":"https://accounts.google.com/o/oauth2/auth","token_uri":"`+tokenServer.URL+`","redirect_uris":["http://127.0.0.1/oauth2/callback"]}}`), 0o600))
	assert.NoError(t, client.ImportOClient(clientSecretPath))

	authURLCh := stubOpenBrowser(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	errCh := make(chan error, 1)
	go func() {
		errCh <- client.Login(context.Background(), LoginOptions{
			Stdout: &stdout,
			Stderr: &stderr,
		})
	}()

	sendOAuthCallback(t, <-authURLCh, "test-code")
	err := <-errCh
	assert.NoError(t, err)
	assert.True(t, tokenEndpointHit)
	assert.Contains(t, stdout.String(), "Login complete")
	assert.Empty(t, stderr.String())

	clientData, readErr := os.ReadFile(client.ClientPath())
	assert.NoError(t, readErr)
	assert.Contains(t, string(clientData), `"client_id":"cid"`)

	token, loadErr := client.LoadOAuthToken()
	assert.NoError(t, loadErr)
	assert.Equal(t, "access", token.AccessToken)
	assert.Equal(t, "refresh", token.RefreshToken)
}

// TestLoginFlowErrorAddsGoogleGuidance checks the friendly OAuth failure hints.
func TestLoginFlowErrorAddsGoogleGuidance(t *testing.T) {
	client := withAuthHome(t)

	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"access_denied","error_description":"Access blocked"}`))
	}))
	defer tokenServer.Close()

	writeClient(t, `{"installed":{"client_id":"cid","client_secret":"secret","auth_uri":"https://accounts.google.com/o/oauth2/auth","token_uri":"`+tokenServer.URL+`","redirect_uris":["http://127.0.0.1/oauth2/callback"]}}`)
	authURLCh := stubOpenBrowser(t)

	errCh := make(chan error, 1)
	go func() {
		errCh <- client.Login(context.Background(), LoginOptions{
			Stdout: new(bytes.Buffer),
			Stderr: new(bytes.Buffer),
		})
	}()

	sendOAuthCallback(t, <-authURLCh, "denied-code")
	err := <-errCh
	assert.Error(t, err)
	if err != nil {
		assert.Contains(t, err.Error(), "Access blocked")
		assert.Contains(t, err.Error(), "Test users")
		assert.Contains(t, err.Error(), "OAuth consent screen")
	}
	assert.False(t, util.FileExists(client.TokenPath()))
}

// TestLoginRejectsNonOClientSecret rejects non-browser client JSON files.
func TestLoginRejectsNonOClientSecret(t *testing.T) {
	client := withAuthHome(t)

	clientSecretPath := filepath.Join(t.TempDir(), "service-account.json")
	assert.NoError(t, os.WriteFile(clientSecretPath, []byte(`{"type":"service_account","client_email":"robot@example.com","private_key":"abc"}`), 0o600))

	err := client.ImportOClient(clientSecretPath)

	assert.Error(t, err)
	if err != nil {
		assert.Contains(t, err.Error(), "unsupported credential file")
	}
}

// TestSelectLoopbackRedirect picks the first supported localhost redirect.
func TestSelectLoopbackRedirect(t *testing.T) {
	redirect, err := selectLoopbackRedirect([]string{
		"https://example.com/callback",
		"http://127.0.0.1/oauth2/callback",
	})

	assert.NoError(t, err)
	if err == nil {
		assert.Equal(t, "http://127.0.0.1/oauth2/callback", redirect.String())
	}
}

// TestSelectLoopbackRedirectMissing rejects configs without a loopback redirect.
func TestSelectLoopbackRedirectMissing(t *testing.T) {
	_, err := selectLoopbackRedirect([]string{"https://example.com/callback"})
	assert.Error(t, err)
}

// TestFriendlyLoginErrorInvalidClient adds the invalid-client recovery hint.
func TestFriendlyLoginErrorInvalidClient(t *testing.T) {
	base := errors.New("invalid_client")
	err := friendlyLoginError(base)

	assert.ErrorIs(t, err, base)
	assert.Contains(t, err.Error(), "re-download")
}

// TestOAuthConfigForLoginDefaultsToGoogleEndpoints keeps Google's default endpoints.
func TestOAuthConfigForLoginDefaultsToGoogleEndpoints(t *testing.T) {
	config, err := oauthConfigForLogin(&OClient{
		ClientID:     "cid",
		ClientSecret: "secret",
		RedirectURIs: []string{"http://127.0.0.1/oauth2/callback"},
	})

	assert.NoError(t, err)
	if err == nil {
		assert.Equal(t, "https://accounts.google.com/o/oauth2/auth", config.Endpoint.AuthURL)
		assert.Equal(t, "https://oauth2.googleapis.com/token", config.Endpoint.TokenURL)
	}
}

// stubOpenBrowser captures the OAuth URL instead of opening a real browser.
func stubOpenBrowser(t *testing.T) <-chan string {
	t.Helper()

	authURLCh := make(chan string, 1)
	orig := openBrowser
	openBrowser = func(rawURL string) error {
		authURLCh <- rawURL
		return nil
	}
	t.Cleanup(func() {
		openBrowser = orig
	})
	return authURLCh
}

// sendOAuthCallback delivers an auth code to the temporary loopback server.
func sendOAuthCallback(t *testing.T, authURL, code string) {
	t.Helper()

	u, err := url.Parse(authURL)
	assert.NoError(t, err)
	if err != nil {
		return
	}

	callbackURL := u.Query().Get("redirect_uri")
	state := u.Query().Get("state")
	assert.NotEmpty(t, callbackURL)
	assert.NotEmpty(t, state)
	if callbackURL == "" || state == "" {
		return
	}

	callback, err := url.Parse(callbackURL)
	assert.NoError(t, err)
	if err != nil {
		return
	}

	query := callback.Query()
	query.Set("code", code)
	query.Set("state", state)
	callback.RawQuery = query.Encode()

	resp, err := http.Get(callback.String())
	assert.NoError(t, err)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// withAuthHome points auth at a fresh temporary HOME directory.
func withAuthHome(t *testing.T) *Manager {
	t.Helper()

	home := t.TempDir()
	t.Cleanup(xdg.Reload)
	t.Setenv("HOME", home)
	xdg.Reload()
	return NewClient()
}

// writeClient saves a test OAuth client file in the current auth config dir.
func writeClient(t *testing.T, body string) {
	t.Helper()

	assert.NoError(t, util.WritePrivateFile(NewClient().ClientPath(), []byte(body)))
}

// writeAuthToken saves a test OAuth token file in the current auth config dir.
func writeAuthToken(t *testing.T, token OAuthToken) {
	t.Helper()

	assert.NoError(t, NewClient().SaveOAuthToken(token))
}

// futureToken returns a valid cached token for auth tests.
func futureToken() OAuthToken {
	return OAuthToken{
		AccessToken:  "access",
		RefreshToken: "refresh",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour),
	}
}
