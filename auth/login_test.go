package auth

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gurgeous/gshoot/app"
	"github.com/gurgeous/gshoot/env"
	"github.com/gurgeous/gshoot/util"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
)

// auth/login_test.go covers the browser login flow and login-specific helpers.

// TestLoginRejectsNonOClientSecret rejects non-browser client JSON files.
func TestLoginRejectsNonOClientSecret(t *testing.T) {
	client := withAuthHome(t)

	clientSecretPath := filepath.Join(t.TempDir(), "service-account.json")
	assert.NoError(t, os.WriteFile(clientSecretPath, []byte(`{"type":"service_account","client_email":"robot@example.com","private_key":"abc"}`), 0o600))

	err := client.SaveOClient(clientSecretPath)

	assert.Error(t, err)
	if err != nil {
		assert.Contains(t, err.Error(), "missing `installed:`")
	}
}

// TestSelectLoopbackRedirect picks the first supported localhost redirect.
func TestSelectLoopbackRedirect(t *testing.T) {
	redirect, err := findLocalhostRedirect([]string{
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
	_, err := findLocalhostRedirect([]string{"https://example.com/callback"})
	assert.Error(t, err)
}

// TestLoginRunsBrowserFlow exercises the loopback OAuth login path.
func TestLoginRunsBrowserFlow(t *testing.T) {
	_ = withAuthHome(t)
	writeClient(t, `{"installed":{"client_id":"cid","client_secret":"secret","redirect_uris":["http://127.0.0.1/oauth2/callback"]}}`)
	manager, err := NewManager()
	assert.NoError(t, err)

	var stdout, stderr bytes.Buffer
	a := app.NewWithWriters(&stdout, &stderr, env.Config{})
	authURLCh := stubOpenBrowser(t)
	sawTokenExchange := stubTokenExchange(t)

	errCh := make(chan error, 1)
	go func() {
		errCh <- manager.Login(context.Background(), a)
	}()

	select {
	case authURL := <-authURLCh:
		sendOAuthCallback(t, authURL, "oauth-code")
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for auth URL")
	}

	select {
	case err = <-errCh:
		assert.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for login")
	}

	token, err := loadOAuthToken(manager.TokenPath)
	assert.NoError(t, err)
	assert.Equal(t, "access", token.AccessToken)
	assert.Equal(t, "refresh", token.RefreshToken)
	assert.Contains(t, stdout.String(), "success")
	assert.True(t, sawTokenExchange())
}

// stubOpenBrowser captures the OAuth URL instead of opening a real browser.
func stubOpenBrowser(t *testing.T) <-chan string {
	t.Helper()

	authURLCh := make(chan string, 1)
	orig := openBrowser
	openBrowser = func(rawURL string) {
		authURLCh <- rawURL
	}
	t.Cleanup(func() {
		openBrowser = orig
	})
	return authURLCh
}

type authRoundTripper func(*http.Request) (*http.Response, error)

func (fn authRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

// stubTokenExchange replaces Google's token endpoint for OAuth exchange tests.
func stubTokenExchange(t *testing.T) func() bool {
	t.Helper()

	sawTokenExchange := false
	orig := http.DefaultTransport
	http.DefaultTransport = authRoundTripper(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host != "oauth2.googleapis.com" {
			return orig.RoundTrip(req)
		}
		if req.URL.Path != "/token" {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(strings.NewReader("not found")),
				Request:    req,
			}, nil
		}
		if err := req.ParseForm(); err != nil {
			return nil, err
		}
		if req.Form.Get("code") != "oauth-code" {
			return &http.Response{
				StatusCode: http.StatusBadRequest,
				Body:       io.NopCloser(strings.NewReader("bad code")),
				Request:    req,
			}, nil
		}
		sawTokenExchange = true

		body := `{"access_token":"access","refresh_token":"refresh","token_type":"Bearer","expires_in":3600}`
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    req,
		}, nil
	})
	t.Cleanup(func() { http.DefaultTransport = orig })
	return func() bool { return sawTokenExchange }
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
	t.Setenv("HOME", home)
	manager, err := NewManager()
	assert.NoError(t, err)
	return manager
}

// writeClient saves a test OAuth client file in the current auth config dir.
func writeClient(t *testing.T, body string) {
	t.Helper()

	path := filepath.Join(util.ConfigDir(), "oauth-client.json")
	assert.NoError(t, util.WritePrivateFile(path, []byte(body)))
}

// writeAuthToken saves a test OAuth token file in the current auth config dir.
func writeAuthToken(t *testing.T, token oauth2.Token) {
	t.Helper()

	manager, err := NewManager()
	assert.NoError(t, err)
	assert.NoError(t, manager.SaveOAuthToken(&token))
}

// futureToken returns a valid cached token for auth tests.
func futureToken() oauth2.Token {
	return oauth2.Token{
		AccessToken:  "access",
		RefreshToken: "refresh",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour),
	}
}
