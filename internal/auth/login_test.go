package auth

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gurgeous/gshoot/internal/util"
)

func TestResolveMissingAuthGuidesLogin(t *testing.T) {
	home := t.TempDir()
	withTestEnv(t, map[string]string{"HOME": home})
	_, err := Resolve(Options{Command: CommandList})
	if err == nil {
		t.Fatal("Resolve() error = nil, want error")
	}

	var noAuth *NoAuthError
	if !errors.As(err, &noAuth) {
		t.Fatalf("Resolve() error = %T, want NoAuthError", err)
	}
	if got, want := noAuth.Error(), "gshoot: list [no auth found]\n"; got != want {
		t.Fatalf("Resolve() error = %q, want %q", got, want)
	}
}

func TestLoginMissingClientSecretGuidance(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Login(context.Background(), LoginOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err == nil {
		t.Fatal("Login() error = nil, want error")
	}

	msg := err.Error()
	for _, want := range []string{
		"oauth-client.json",
		"Desktop app",
		"Test users",
		"gshoot auth login --client-secret",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("Login() error = %q, want %q", msg, want)
		}
	}
}

func TestLoginImportsClientAndSavesToken(t *testing.T) {
	home := t.TempDir()
	var tokenEndpointHit bool
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenEndpointHit = true
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm() error = %v", err)
		}
		if got, want := r.Form.Get("code"), "test-code"; got != want {
			t.Fatalf("token exchange code = %q, want %q", got, want)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"access","refresh_token":"refresh","token_type":"Bearer","expires_in":3600}`))
	}))
	defer tokenServer.Close()

	clientSecret := writeFile(t, filepath.Join(t.TempDir(), "client_secret.json"), `{"installed":{"client_id":"cid","client_secret":"secret","auth_uri":"https://accounts.google.com/o/oauth2/auth","token_uri":"`+tokenServer.URL+`","redirect_uris":["http://127.0.0.1/oauth2/callback"]}}`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	authURLCh := stubOpenBrowser(t)
	withTestEnv(t, map[string]string{"HOME": home})
	errCh := make(chan error, 1)
	go func() {
		errCh <- Login(context.Background(), LoginOptions{
			ClientSecretPath: clientSecret,
			Stdout:           &stdout,
			Stderr:           &stderr,
		})
	}()

	sendOAuthCallback(t, <-authURLCh, "test-code")

	if err := <-errCh; err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	configDir := ConfigDir()
	clientData, err := os.ReadFile(filepath.Join(configDir, oauthClientFileName))
	if err != nil {
		t.Fatalf("ReadFile(client) error = %v", err)
	}
	if !strings.Contains(string(clientData), `"client_id":"cid"`) {
		t.Fatalf("oauth-client.json = %q, want imported client", string(clientData))
	}
	if !tokenEndpointHit {
		t.Fatal("token endpoint was not called")
	}

	token, err := LoadOAuthToken(filepath.Join(configDir, oauthTokenFileName))
	if err != nil {
		t.Fatalf("LoadOAuthToken() error = %v", err)
	}
	if token.AccessToken != "access" || token.RefreshToken != "refresh" {
		t.Fatalf("saved token = %#v, want oauth token", token)
	}
	if !strings.Contains(stdout.String(), "Login complete") {
		t.Fatalf("stdout = %q, want success message", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestLoginFlowErrorAddsGoogleGuidance(t *testing.T) {
	home := t.TempDir()
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"access_denied","error_description":"Access blocked"}`))
	}))
	defer tokenServer.Close()

	withTestEnv(t, map[string]string{"HOME": home})
	writeFile(t, filepath.Join(ConfigDir(), oauthClientFileName), `{"installed":{"client_id":"cid","client_secret":"secret","auth_uri":"https://accounts.google.com/o/oauth2/auth","token_uri":"`+tokenServer.URL+`","redirect_uris":["http://127.0.0.1/oauth2/callback"]}}`)
	authURLCh := stubOpenBrowser(t)
	errCh := make(chan error, 1)
	go func() {
		errCh <- Login(context.Background(), LoginOptions{
			Stdout: new(bytes.Buffer),
			Stderr: new(bytes.Buffer),
		})
	}()

	sendOAuthCallback(t, <-authURLCh, "denied-code")

	err := <-errCh
	if err == nil {
		t.Fatal("Login() error = nil, want error")
	}

	msg := err.Error()
	for _, want := range []string{
		"Access blocked",
		"Test users",
		"OAuth consent screen",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("Login() error = %q, want %q", msg, want)
		}
	}
	if util.FileExists(filepath.Join(ConfigDir(), oauthTokenFileName)) {
		t.Fatal("Login() should not save a token on exchange failure")
	}
}

func TestLoginRejectsNonOAuthClientSecret(t *testing.T) {
	home := t.TempDir()
	clientSecret := writeFile(t, filepath.Join(t.TempDir(), "service-account.json"), `{"type":"service_account","client_email":"robot@example.com","private_key":"abc"}`)

	withTestEnv(t, map[string]string{"HOME": home})
	err := Login(context.Background(), LoginOptions{
		ClientSecretPath: clientSecret,
		Stdout:           new(bytes.Buffer),
		Stderr:           new(bytes.Buffer),
	})
	if err == nil {
		t.Fatal("Login() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "Desktop app OAuth client") {
		t.Fatalf("Login() error = %q, want oauth client error", err.Error())
	}
}

func TestSelectLoopbackRedirect(t *testing.T) {
	got, err := selectLoopbackRedirect([]string{
		"https://example.com/callback",
		"http://127.0.0.1/oauth2/callback",
	})
	if err != nil {
		t.Fatalf("selectLoopbackRedirect() error = %v", err)
	}
	if got.String() != "http://127.0.0.1/oauth2/callback" {
		t.Fatalf("selectLoopbackRedirect() = %q, want loopback redirect", got.String())
	}
}

func TestSelectLoopbackRedirectMissing(t *testing.T) {
	_, err := selectLoopbackRedirect([]string{"https://example.com/callback"})
	if err == nil {
		t.Fatal("selectLoopbackRedirect() error = nil, want error")
	}
}

func TestFriendlyLoginErrorInvalidClient(t *testing.T) {
	base := errors.New("invalid_client")
	err := friendlyLoginError(base)
	if !strings.Contains(err.Error(), "re-download") {
		t.Fatalf("friendlyLoginError() = %q, want invalid-client hint", err.Error())
	}
	if !errors.Is(err, base) {
		t.Fatalf("friendlyLoginError() should wrap the original error")
	}
}

func TestOAuthConfigForLoginDefaultsToGoogleEndpoints(t *testing.T) {
	config, err := oauthConfigForLogin(&OAuthClient{
		ClientID:     "cid",
		ClientSecret: "secret",
		RedirectURIs: []string{"http://127.0.0.1/oauth2/callback"},
	})
	if err != nil {
		t.Fatalf("oauthConfigForLogin() error = %v", err)
	}
	if got, want := config.Endpoint.AuthURL, "https://accounts.google.com/o/oauth2/auth"; got != want {
		t.Fatalf("config.Endpoint.AuthURL = %q, want %q", got, want)
	}
	if got, want := config.Endpoint.TokenURL, "https://oauth2.googleapis.com/token"; got != want {
		t.Fatalf("config.Endpoint.TokenURL = %q, want %q", got, want)
	}
}

func TestInspectStatusReadyForLogin(t *testing.T) {
	home := t.TempDir()
	withTestEnv(t, map[string]string{"HOME": home})
	writeFile(t, filepath.Join(ConfigDir(), oauthClientFileName), `{"installed":{"client_id":"cid","client_secret":"secret","auth_uri":"https://accounts.google.com/o/oauth2/auth","token_uri":"https://oauth2.googleapis.com/token","redirect_uris":["http://127.0.0.1"]}}`)
	status := InspectStatus()
	if !status.HasOAuthClient || status.LoggedIn {
		t.Fatalf("InspectStatus() = %#v, want ready-for-login state", status)
	}
}

func TestInspectStatusAuthenticatedFromCachedToken(t *testing.T) {
	home := t.TempDir()
	withTestEnv(t, map[string]string{"HOME": home})
	writeFile(t, filepath.Join(ConfigDir(), oauthClientFileName), `{"installed":{"client_id":"cid","client_secret":"secret","auth_uri":"https://accounts.google.com/o/oauth2/auth","token_uri":"https://oauth2.googleapis.com/token","redirect_uris":["http://127.0.0.1"]}}`)
	writeFile(t, filepath.Join(ConfigDir(), oauthTokenFileName), `{"access_token":"a","refresh_token":"r","token_type":"Bearer","expiry":"3026-05-07T22:00:00Z"}`)
	status := InspectStatus()
	if !status.LoggedIn || status.ResolvedSource != SourceKindCachedOAuth {
		t.Fatalf("InspectStatus() = %#v, want cached-oauth logged-in state", status)
	}
}

func TestPrintStatusNoAuth(t *testing.T) {
	var out bytes.Buffer
	PrintStatus(&out, Status{ConfigDir: "/tmp/gshoot"})
	if !strings.Contains(out.String(), "auth login --client-secret") {
		t.Fatalf("PrintStatus() = %q, want login guidance", out.String())
	}
}

func TestLogout(t *testing.T) {
	home := t.TempDir()
	withTestEnv(t, map[string]string{"HOME": home})
	writeFile(t, filepath.Join(ConfigDir(), oauthTokenFileName), `{"access_token":"a","token_type":"Bearer","expiry":"3026-05-07T22:00:00Z"}`)
	removed, err := Logout()
	if err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	if !removed {
		t.Fatal("Logout() removed = false, want true")
	}
	if util.FileExists(filepath.Join(ConfigDir(), oauthTokenFileName)) {
		t.Fatal("Logout() should remove the cached token")
	}
}

func TestLogoutMissingToken(t *testing.T) {
	withTestEnv(t, map[string]string{"HOME": t.TempDir()})
	removed, err := Logout()
	if err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	if removed {
		t.Fatal("Logout() removed = true, want false")
	}
}

func stubOpenBrowser(t *testing.T) <-chan string {
	t.Helper()

	// This swaps a package-global test seam, so callers must not run in parallel.
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

func sendOAuthCallback(t *testing.T, authURL, code string) {
	t.Helper()

	u, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("Parse(authURL) error = %v", err)
	}

	callbackURL := u.Query().Get("redirect_uri")
	if callbackURL == "" {
		t.Fatal("auth URL missing redirect_uri")
	}
	state := u.Query().Get("state")
	if state == "" {
		t.Fatal("auth URL missing state")
	}

	callback, err := url.Parse(callbackURL)
	if err != nil {
		t.Fatalf("Parse(callbackURL) error = %v", err)
	}
	query := callback.Query()
	query.Set("code", code)
	query.Set("state", state)
	callback.RawQuery = query.Encode()

	resp, err := http.Get(callback.String())
	if err != nil {
		t.Fatalf("GET(callback) error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("callback status = %d, want 200", resp.StatusCode)
	}
}
