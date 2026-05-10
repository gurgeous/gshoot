package status

import (
	"bytes"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gurgeous/gshoot/internal/auth"
	"github.com/gurgeous/gshoot/internal/testutil"
	"github.com/gurgeous/gshoot/internal/util"
)

func TestNewStatusCommand(t *testing.T) {
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
	cmd := NewStatusCommand()
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
