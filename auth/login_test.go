package auth

import (
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

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

// captureProcessIO redirects stdout/stderr for auth tests.
func captureProcessIO(t *testing.T) func() (string, string) {
	t.Helper()

	stdoutPath := filepath.Join(t.TempDir(), "stdout")
	stderrPath := filepath.Join(t.TempDir(), "stderr")
	stdoutFile, err := os.Create(stdoutPath)
	assert.NoError(t, err)
	stderrFile, err := os.Create(stderrPath)
	assert.NoError(t, err)

	origStdout, origStderr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = stdoutFile, stderrFile
	t.Cleanup(func() {
		os.Stdout, os.Stderr = origStdout, origStderr
		assert.NoError(t, stdoutFile.Close())
		assert.NoError(t, stderrFile.Close())
	})

	return func() (string, string) {
		t.Helper()

		assert.NoError(t, stdoutFile.Sync())
		assert.NoError(t, stderrFile.Sync())
		_, err := stdoutFile.Seek(0, 0)
		assert.NoError(t, err)
		_, err = stderrFile.Seek(0, 0)
		assert.NoError(t, err)
		stdout, err := io.ReadAll(stdoutFile)
		assert.NoError(t, err)
		stderr, err := io.ReadAll(stderrFile)
		assert.NoError(t, err)
		return string(stdout), string(stderr)
	}
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
