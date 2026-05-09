package auth

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestNewCommandLogin(t *testing.T) {
	restore := stubCommand(t)
	defer restore()

	login = func(_ context.Context, opts LoginOptions) error {
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
	cmd := NewCommand()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"login", "--client-secret", "/tmp/client.json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestNewCommandLoginError(t *testing.T) {
	restore := stubCommand(t)
	defer restore()

	login = func(context.Context, LoginOptions) error {
		return errors.New("bad login")
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := NewCommand()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"login"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want error")
	}
	if got, want := err.Error(), "bad login"; got != want {
		t.Fatalf("error = %q, want %q", got, want)
	}
}

func TestNewCommandStatus(t *testing.T) {
	restore := stubCommand(t)
	defer restore()

	status = func() Status {
		return Status{ConfigDir: "/tmp/gshoot", ReadyForLogin: true}
	}
	printStatus = func(w io.Writer, status Status) {
		_, _ = io.WriteString(w, "Status: not logged in yet\n")
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := NewCommand()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"status"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(stdout.String(), "not logged in") {
		t.Fatalf("stdout = %q, want status output", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestNewCommandLogout(t *testing.T) {
	restore := stubCommand(t)
	defer restore()

	logout = func() (bool, error) { return true, nil }

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := NewCommand()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"logout"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(stdout.String(), "Removed cached OAuth token") {
		t.Fatalf("stdout = %q, want logout message", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func stubCommand(t *testing.T) func() {
	t.Helper()

	origLogin := login
	origLogout := logout
	origStatus := status
	origPrint := printStatus
	return func() {
		login = origLogin
		logout = origLogout
		status = origStatus
		printStatus = origPrint
	}
}
