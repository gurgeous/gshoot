package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/gurgeous/gshoot/internal/auth"
)

func TestRunAuthLogin(t *testing.T) {
	restore := stubAuthLogin(t)
	defer restore()

	loginAuth = func(_ context.Context, opts auth.LoginOptions) error {
		if opts.ClientSecretPath != "/tmp/client.json" {
			t.Fatalf("Login() client secret = %q, want /tmp/client.json", opts.ClientSecretPath)
		}
		if opts.Stdout == nil || opts.Stderr == nil {
			t.Fatal("Login() missing stdio")
		}
		return nil
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"auth", "login", "--client-secret", "/tmp/client.json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run() code = %d, want 0", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunListNoAuthShowsFriendlyHint(t *testing.T) {
	restore := stubAuthLogin(t)
	defer restore()

	resolveAuth = func(auth.Options) (auth.Resolved, error) {
		return auth.Resolved{}, &auth.NoAuthError{
			Command:   auth.CommandList,
			ConfigDir: "/tmp/gshoot",
		}
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"list"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("Run() code = %d, want 1", code)
	}
	got := stderr.String()
	for _, want := range []string{
		"You will need to authenticate first.",
		"setting up auth with Google Sheets is",
		"Don't blame gshoot.",
		"Try this first:",
		"gshoot auth status",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("stderr missing %q:\n%q", want, got)
		}
	}
}

func TestRunAuthLoginError(t *testing.T) {
	restore := stubAuthLogin(t)
	defer restore()

	loginAuth = func(context.Context, auth.LoginOptions) error {
		return errors.New("bad login")
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"auth", "login"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("Run() code = %d, want 1", code)
	}
	if got, want := stderr.String(), "gshoot: bad login\n"+helpHint+"\n"; got != want {
		t.Fatalf("stderr = %q, want %q", got, want)
	}
}

func TestRunAuthStatus(t *testing.T) {
	restore := stubAuthLogin(t)
	defer restore()

	statusAuth = func() auth.Status {
		return auth.Status{ConfigDir: "/tmp/gshoot", ReadyForLogin: true}
	}
	printAuthStatus = func(w io.Writer, status auth.Status) {
		_, _ = io.WriteString(w, "Status: not logged in yet\n")
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"auth", "status"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run() code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "not logged in") {
		t.Fatalf("stdout = %q, want status output", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunAuthLogout(t *testing.T) {
	restore := stubAuthLogin(t)
	defer restore()

	logoutAuth = func() (bool, error) { return true, nil }

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"auth", "logout"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run() code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "Removed cached OAuth token") {
		t.Fatalf("stdout = %q, want logout message", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func stubAuthLogin(t *testing.T) func() {
	t.Helper()

	origLogin := loginAuth
	origResolve := resolveAuth
	origLogout := logoutAuth
	origStatus := statusAuth
	origPrint := printAuthStatus
	return func() {
		loginAuth = origLogin
		resolveAuth = origResolve
		logoutAuth = origLogout
		statusAuth = origStatus
		printAuthStatus = origPrint
	}
}
