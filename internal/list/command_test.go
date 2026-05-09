package list

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/gurgeous/gshoot/internal/auth"
	"github.com/gurgeous/gshoot/internal/google"
	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v3"
)

func TestNewCommand(t *testing.T) {
	restore := stubDeps()
	defer restore()

	resolveAuth = func(opts auth.Options) (auth.Resolved, error) {
		if opts.Command != auth.CommandList {
			t.Fatalf("Resolve() command = %q, want list", opts.Command)
		}
		return auth.Resolved{Scopes: auth.ScopesForCommand(auth.CommandList)}, nil
	}
	newTokenSource = func(_ context.Context, _ auth.Resolved) (oauth2.TokenSource, error) {
		return oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "token"}), nil
	}
	newGoogle = func(context.Context, oauth2.TokenSource) (*google.Client, error) {
		return &google.Client{}, nil
	}
	listRecent = func(context.Context, *google.Client, int) ([]*drive.File, error) {
		return []*drive.File{
			{Name: "Alpha", ModifiedTime: "2026-05-07T12:00:00Z"},
			{Name: "Beta", ModifiedTime: "2026-05-07T11:00:00Z"},
		}, nil
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := NewCommand()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	for _, want := range []string{
		"2026-05-07T12:00:00Z  Alpha",
		"2026-05-07T11:00:00Z  Beta",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("stdout missing %q:\n%s", want, stdout.String())
		}
	}
	for _, want := range []string{
		"listing spreadsheets...",
		"2 recent spreadsheets",
	} {
		if !strings.Contains(stderr.String(), want) {
			t.Fatalf("stderr missing %q:\n%s", want, stderr.String())
		}
	}
}

func TestNewCommandAuthError(t *testing.T) {
	restore := stubDeps()
	defer restore()

	resolveAuth = func(auth.Options) (auth.Resolved, error) {
		return auth.Resolved{}, errors.New("no auth")
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := NewCommand()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want error")
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(err.Error(), "no auth") {
		t.Fatalf("error = %q, want auth error", err.Error())
	}
}

func stubDeps() func() {
	origResolve := resolveAuth
	origToken := newTokenSource
	origGoogle := newGoogle
	origListRecent := listRecent
	return func() {
		resolveAuth = origResolve
		newTokenSource = origToken
		newGoogle = origGoogle
		listRecent = origListRecent
	}
}
