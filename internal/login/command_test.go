package login

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/gurgeous/gshoot/internal/auth"
)

func TestNewCommand(t *testing.T) {
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
	cmd := NewCommand()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--client-secret", "/tmp/client.json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestNewCommandError(t *testing.T) {
	orig := runLogin
	runLogin = func(context.Context, auth.LoginOptions) error {
		return errors.New("bad login")
	}
	t.Cleanup(func() {
		runLogin = orig
	})

	cmd := NewCommand()
	cmd.SetArgs(nil)

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want error")
	}
	if got, want := err.Error(), "bad login"; got != want {
		t.Fatalf("error = %q, want %q", got, want)
	}
}
