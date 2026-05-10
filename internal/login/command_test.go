package login

import (
	"bytes"
	"context"
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
	cmd := NewLoginCommand()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--client-secret", "/tmp/client.json"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}
