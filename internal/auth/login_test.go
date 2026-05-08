package auth

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestResolveMissingAuthGuidesLogin(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	_, err := Resolve(Options{
		Env:     NewEnv(map[string]string{"HOME": home}),
		Command: CommandList,
	})
	if err == nil {
		t.Fatal("Resolve() error = nil, want error")
	}

	var noAuth *NoAuthError
	if !errors.As(err, &noAuth) {
		t.Fatalf("Resolve() error = %T, want NoAuthError", err)
	}
	if !strings.Contains(err.Error(), "gshoot auth login") {
		t.Fatalf("Resolve() error = %q, want login guidance", err.Error())
	}
}

func TestLoginMissingClientSecretGuidance(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := Login(context.Background(), LoginOptions{
		Env:    NewEnv(map[string]string{"HOME": t.TempDir()}),
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
	t.Parallel()

	home := t.TempDir()
	clientSecret := writeFile(t, filepath.Join(t.TempDir(), "client_secret.json"), `{"installed":{"client_id":"cid","client_secret":"secret","auth_uri":"https://accounts.google.com/o/oauth2/auth","token_uri":"https://oauth2.googleapis.com/token","redirect_uris":["http://127.0.0.1"]}}`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := Login(context.Background(), LoginOptions{
		Env:              NewEnv(map[string]string{"HOME": home}),
		ClientSecretPath: clientSecret,
		Stdout:           &stdout,
		Stderr:           &stderr,
		RunFlow: func(context.Context, *oauth2.Config, io.Writer, io.Writer) (*oauth2.Token, error) {
			return &oauth2.Token{
				AccessToken:  "access",
				RefreshToken: "refresh",
				TokenType:    "Bearer",
				Expiry:       time.Date(2026, 5, 8, 0, 0, 0, 0, time.UTC),
			}, nil
		},
	})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	configDir := filepath.Join(home, ".config", "gshoot")
	clientData, err := os.ReadFile(filepath.Join(configDir, oauthClientFileName))
	if err != nil {
		t.Fatalf("ReadFile(client) error = %v", err)
	}
	if !strings.Contains(string(clientData), `"client_id":"cid"`) {
		t.Fatalf("oauth-client.json = %q, want imported client", string(clientData))
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
	t.Parallel()

	home := t.TempDir()
	writeFile(t, filepath.Join(home, ".config", "gshoot", oauthClientFileName), `{"installed":{"client_id":"cid","client_secret":"secret","auth_uri":"https://accounts.google.com/o/oauth2/auth","token_uri":"https://oauth2.googleapis.com/token","redirect_uris":["http://127.0.0.1"]}}`)

	err := Login(context.Background(), LoginOptions{
		Env:    NewEnv(map[string]string{"HOME": home}),
		Stdout: new(bytes.Buffer),
		Stderr: new(bytes.Buffer),
		RunFlow: func(context.Context, *oauth2.Config, io.Writer, io.Writer) (*oauth2.Token, error) {
			return nil, errors.New("access_denied: Access blocked")
		},
	})
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
}

func TestLoginRejectsNonOAuthClientSecret(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	clientSecret := writeFile(t, filepath.Join(t.TempDir(), "service-account.json"), `{"type":"service_account","client_email":"robot@example.com","private_key":"abc"}`)

	err := Login(context.Background(), LoginOptions{
		Env:              NewEnv(map[string]string{"HOME": home}),
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
	t.Parallel()

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
	t.Parallel()

	_, err := selectLoopbackRedirect([]string{"https://example.com/callback"})
	if err == nil {
		t.Fatal("selectLoopbackRedirect() error = nil, want error")
	}
}

func TestFriendlyLoginErrorInvalidClient(t *testing.T) {
	t.Parallel()

	base := errors.New("invalid_client")
	err := friendlyLoginError(base)
	if !strings.Contains(err.Error(), "re-download") {
		t.Fatalf("friendlyLoginError() = %q, want invalid-client hint", err.Error())
	}
	if !errors.Is(err, base) {
		t.Fatalf("friendlyLoginError() should wrap the original error")
	}
}

func TestInspectStatusReadyForLogin(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	writeFile(t, filepath.Join(home, ".config", "gshoot", oauthClientFileName), `{"installed":{"client_id":"cid","client_secret":"secret","auth_uri":"https://accounts.google.com/o/oauth2/auth","token_uri":"https://oauth2.googleapis.com/token","redirect_uris":["http://127.0.0.1"]}}`)

	status := InspectStatus(NewEnv(map[string]string{"HOME": home}))
	if !status.HasOAuthClient || status.LoggedIn {
		t.Fatalf("InspectStatus() = %#v, want ready-for-login state", status)
	}
}

func TestInspectStatusAuthenticatedFromCachedToken(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	writeFile(t, filepath.Join(home, ".config", "gshoot", oauthClientFileName), `{"installed":{"client_id":"cid","client_secret":"secret","auth_uri":"https://accounts.google.com/o/oauth2/auth","token_uri":"https://oauth2.googleapis.com/token","redirect_uris":["http://127.0.0.1"]}}`)
	writeFile(t, filepath.Join(home, ".config", "gshoot", oauthTokenFileName), `{"access_token":"a","refresh_token":"r","token_type":"Bearer","expiry":"3026-05-07T22:00:00Z"}`)

	status := InspectStatus(NewEnv(map[string]string{"HOME": home}))
	if !status.LoggedIn || status.ResolvedSource != SourceKindCachedOAuth {
		t.Fatalf("InspectStatus() = %#v, want cached-oauth logged-in state", status)
	}
}

func TestPrintStatusNoAuth(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	PrintStatus(&out, Status{ConfigDir: "/tmp/gshoot"})
	if !strings.Contains(out.String(), "auth login --client-secret") {
		t.Fatalf("PrintStatus() = %q, want login guidance", out.String())
	}
}

func TestLogout(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	writeFile(t, filepath.Join(home, ".config", "gshoot", oauthTokenFileName), `{"access_token":"a","token_type":"Bearer","expiry":"3026-05-07T22:00:00Z"}`)

	removed, err := Logout(NewEnv(map[string]string{"HOME": home}))
	if err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	if !removed {
		t.Fatal("Logout() removed = false, want true")
	}
	if fileExists(filepath.Join(home, ".config", "gshoot", oauthTokenFileName)) {
		t.Fatal("Logout() should remove the cached token")
	}
}

func TestLogoutMissingToken(t *testing.T) {
	t.Parallel()

	removed, err := Logout(NewEnv(map[string]string{"HOME": t.TempDir()}))
	if err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	if removed {
		t.Fatal("Logout() removed = true, want false")
	}
}
