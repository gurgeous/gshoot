package smoke

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gurgeous/gshoot/internal/auth"
	"golang.org/x/oauth2"
)

func TestRunDownSmoke(t *testing.T) {
	restore := stubSmokeDeps(t)
	defer restore()

	tempDir := t.TempDir()
	var ran [][]string

	resolveAuth = func(opts auth.Options) (auth.Resolved, error) {
		if opts.Command != auth.CommandUp {
			t.Fatalf("Resolve() command = %q, want up", opts.Command)
		}
		return auth.Resolved{Scopes: auth.ScopesForCommand(auth.CommandUp)}, nil
	}
	newTokenSource = func(context.Context, auth.Resolved) (oauth2.TokenSource, error) {
		return oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "token"}), nil
	}
	newSmokeClient = func(context.Context, oauth2.TokenSource) (Client, error) {
		return &fakeResetClient{}, nil
	}
	runCommand = func(name string, args ...string) error {
		ran = append(ran, append([]string{name}, args...))
		return os.WriteFile(filepath.Join(tempDir, "gsmoke-down.csv"), []byte(expectedDownCSV), 0o644)
	}
	smokeTmpDir = func() string { return tempDir }

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"--gshoot", "/tmp/gshoot"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run() code = %d, want 0", code)
	}

	if len(ran) != 1 {
		t.Fatalf("runCommand() calls = %d, want 1", len(ran))
	}
	want := []string{"/tmp/gshoot", "down", "gsmoke", "down-basic", "--output", filepath.Join(tempDir, "gsmoke-down.csv")}
	if strings.Join(ran[0], "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("runCommand() = %#v, want %#v", ran[0], want)
	}

	if !strings.Contains(stdout.String(), "gsmoke") || !strings.Contains(stdout.String(), "down-basic") {
		t.Fatalf("stdout = %q, want fixture summary", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

type fakeResetClient struct{}

func (f *fakeResetClient) ResetDownFixture(context.Context, string, string, [][]string) (string, error) {
	return "spreadsheet-id", nil
}
