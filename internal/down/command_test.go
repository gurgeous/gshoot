package down

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/gurgeous/gshoot/internal/auth"
	"github.com/gurgeous/gshoot/internal/google"
	"golang.org/x/oauth2"
)

func TestNewCommandStdout(t *testing.T) {
	restore := stubCommandDeps()
	defer restore()

	resolveAuth = func(opts auth.Options) (auth.Resolved, error) {
		if opts.Command != auth.CommandDown {
			t.Fatalf("Resolve() command = %q, want down", opts.Command)
		}
		return auth.Resolved{Scopes: auth.ScopesForCommand(auth.CommandDown)}, nil
	}
	newTokenSource = func(_ context.Context, _ auth.Resolved) (oauth2.TokenSource, error) {
		return oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "token"}), nil
	}
	newGoogle = func(context.Context, oauth2.TokenSource) (*google.Client, error) {
		return &google.Client{}, nil
	}
	downloadSheet = func(_ context.Context, _ *google.Client, spreadsheetName, sheetName string) ([][]string, error) {
		if spreadsheetName != "Budget" || sheetName != "" {
			t.Fatalf("Download() args = (%q, %q)", spreadsheetName, sheetName)
		}
		return [][]string{{"name", "count"}, {"alpha", "1"}}, nil
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := NewCommand()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"Budget"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got, want := stdout.String(), "name,count\nalpha,1\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestNewCommandOutputFile(t *testing.T) {
	restore := stubCommandDeps()
	defer restore()

	resolveAuth = func(auth.Options) (auth.Resolved, error) {
		return auth.Resolved{Scopes: auth.ScopesForCommand(auth.CommandDown)}, nil
	}
	newTokenSource = func(_ context.Context, _ auth.Resolved) (oauth2.TokenSource, error) {
		return oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "token"}), nil
	}
	newGoogle = func(context.Context, oauth2.TokenSource) (*google.Client, error) {
		return &google.Client{}, nil
	}
	downloadSheet = func(_ context.Context, _ *google.Client, _, _ string) ([][]string, error) {
		return [][]string{{"name", "count"}, {"alpha", "1"}}, nil
	}

	path := filepath.Join(t.TempDir(), "out.csv")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := NewCommand()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"Budget", "--output", path})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if got, want := string(data), "name,count\nalpha,1\n"; got != want {
		t.Fatalf("file = %q, want %q", got, want)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func stubCommandDeps() func() {
	origResolve := resolveAuth
	origToken := newTokenSource
	origGoogle := newGoogle
	origDownload := downloadSheet
	return func() {
		resolveAuth = origResolve
		newTokenSource = origToken
		newGoogle = origGoogle
		downloadSheet = origDownload
	}
}
