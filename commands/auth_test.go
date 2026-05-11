package commands

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gurgeous/gshoot/auth"
	"github.com/gurgeous/gshoot/testutil"
	"github.com/gurgeous/gshoot/util"
	"github.com/stretchr/testify/assert"
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

	code, _, _ := testMain("auth", "login", "--client-secret", "/tmp/client.json")
	assert.Equal(t, 0, code)
}

func TestLogoutCommand(t *testing.T) {
	orig := runLogout
	runLogout = func() (bool, error) { return true, nil }
	t.Cleanup(func() {
		runLogout = orig
	})

	code, stdout, _ := testMain("auth", "logout")
	assert.Equal(t, 0, code)
	assert.Contains(t, stdout, "Removed cached OAuth token")
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

	code, stdout, _ := testMain("auth", "status")
	assert.Equal(t, 0, code)
	assert.Contains(t, stdout, "not logged in")
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
	var out strings.Builder
	writeStatus(&out)
	assert.Contains(t, out.String(), "authenticated via cached_oauth")
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
	var out strings.Builder
	writeStatus(&out)
	assert.Contains(t, out.String(), "auth login --client-secret")
}
