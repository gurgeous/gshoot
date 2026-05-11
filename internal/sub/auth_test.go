package sub

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gurgeous/gshoot/internal/auth"
	"github.com/gurgeous/gshoot/internal/testutil"
	"github.com/gurgeous/gshoot/internal/util"
)

func TestLoginCommand(t *testing.T) {
	orig := runLogin
	runLogin = func(_ context.Context, opts auth.LoginOptions) error {
		if opts.ClientSecretPath != "/tmp/client.json" {
			t.Fatalf("Login() client secret = %q, want /tmp/client.json", opts.ClientSecretPath)
		}
		if opts.Stdout == nil || opts.Stderr == nil {
			t.Fatal("Login() missing stdio")
		}
		return nil
	}
	t.Cleanup(func() {
		runLogin = orig
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := newLoginCommand()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--client-secret", "/tmp/client.json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestLogoutCommand(t *testing.T) {
	orig := runLogout
	runLogout = func() (bool, error) { return true, nil }
	t.Cleanup(func() {
		runLogout = orig
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := newLogoutCommand()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(stdout.String(), "Removed cached OAuth token") {
		t.Fatalf("stdout = %q, want logout message", stdout.String())
	}
}

func TestStatusCommand(t *testing.T) {
	origResolve := resolveAuth
	resolveAuth = func() (auth.Resolved, error) {
		return auth.Resolved{}, errors.New("no auth")
	}
	t.Cleanup(func() {
		resolveAuth = origResolve
	})

	home := t.TempDir()
	testutil.WithEnv(t, map[string]string{"HOME": home}, nil)
	clientPath := filepath.Join(home, ".config", "gshoot", "oauth-client.json")
	if err := util.WritePrivateFile(clientPath, []byte(`{"installed":{"client_id":"cid"}}`)); err != nil {
		t.Fatalf("WritePrivateFile() error = %v", err)
	}

	var stdout bytes.Buffer
	cmd := newStatusCommand()
	cmd.SetOut(&stdout)
	cmd.SetArgs(nil)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(stdout.String(), "not logged in") {
		t.Fatalf("stdout = %q, want status output", stdout.String())
	}
}

func TestWriteStatusAuthenticated(t *testing.T) {
	origResolve := resolveAuth
	resolveAuth = func() (auth.Resolved, error) {
		return auth.Resolved{
			Source: auth.Source{
				Kind: auth.SourceKindCachedOAuth,
				Path: "/tmp/token.json",
			},
		}, nil
	}
	t.Cleanup(func() {
		resolveAuth = origResolve
	})

	testutil.WithEnv(t, map[string]string{"HOME": t.TempDir()}, nil)
	var out bytes.Buffer
	writeStatus(&out)
	if !strings.Contains(out.String(), "authenticated via cached_oauth") {
		t.Fatalf("writeStatus() = %q, want authenticated output", out.String())
	}
}

func TestWriteStatusNoAuth(t *testing.T) {
	origResolve := resolveAuth
	resolveAuth = func() (auth.Resolved, error) {
		return auth.Resolved{}, errors.New("no auth")
	}
	t.Cleanup(func() {
		resolveAuth = origResolve
	})

	testutil.WithEnv(t, map[string]string{"HOME": t.TempDir()}, nil)
	var out bytes.Buffer
	writeStatus(&out)
	if !strings.Contains(out.String(), "auth login --client-secret") {
		t.Fatalf("writeStatus() = %q, want login guidance", out.String())
	}
}
