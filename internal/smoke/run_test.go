package smoke

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gurgeous/gshoot/internal/auth"
	"golang.org/x/oauth2"
)

func TestRunRequiresBinary(t *testing.T) {
	restore := stubSmokeDeps(t)
	defer restore()

	executablePath = func() (string, error) { return filepath.Join(t.TempDir(), "missing-smoke"), nil }
	resolveAuth = func(auth.Options) (auth.Resolved, error) {
		t.Fatal("Resolve() should not be called")
		return auth.Resolved{}, nil
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run(nil, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("Run() code = %d, want 1", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "missing gshoot command") {
		t.Fatalf("stderr = %q, want missing command", stderr.String())
	}
}

func TestRunInfersSiblingBinary(t *testing.T) {
	restore := stubSmokeDeps(t)
	defer restore()

	exeDir := t.TempDir()
	tmpDir := t.TempDir()
	exePath := writeExecutable(t, filepath.Join(exeDir, "smoke"))
	gshootPath := writeExecutable(t, filepath.Join(exeDir, "gshoot"))

	executablePath = func() (string, error) { return exePath, nil }
	resolveAuth = func(auth.Options) (auth.Resolved, error) {
		return auth.Resolved{Scopes: auth.ScopesForCommand(auth.CommandUp)}, nil
	}
	newTokenSource = func(context.Context, auth.Resolved) (oauth2.TokenSource, error) {
		return oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "token"}), nil
	}
	newSmokeClient = func(context.Context, oauth2.TokenSource) (Client, error) {
		return &fakeResetClient{}, nil
	}
	var ran []string
	runCommand = func(name string, args ...string) error {
		ran = append([]string{name}, args...)
		return os.WriteFile(filepath.Join(smokeTmpDir(), "gsmoke-down.csv"), []byte(expectedDownCSV), 0o644)
	}
	smokeTmpDir = func() string { return tmpDir }

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run(nil, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run() code = %d, want 0", code)
	}
	want := []string{gshootPath, "down", "gsmoke", "down-basic", "--output", filepath.Join(tmpDir, "gsmoke-down.csv")}
	if strings.Join(ran, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("runCommand() = %#v, want %#v", ran, want)
	}
	if !strings.Contains(stdout.String(), "running "+gshootPath+" down gsmoke down-basic") {
		t.Fatalf("stdout = %q, want inferred command", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunPropagatesAuthError(t *testing.T) {
	restore := stubSmokeDeps(t)
	defer restore()

	resolveAuth = func(auth.Options) (auth.Resolved, error) {
		return auth.Resolved{}, errors.New("no auth")
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"--gshoot", "/tmp/gshoot"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("Run() code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "no auth") {
		t.Fatalf("stderr = %q, want auth error", stderr.String())
	}
}

func writeExecutable(t *testing.T, path string) string {
	t.Helper()

	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}

func stubSmokeDeps(t interface {
	Helper()
},
) func() {
	t.Helper()

	origResolve := resolveAuth
	origToken := newTokenSource
	origClient := newSmokeClient
	origRun := runCommand
	origExecutablePath := executablePath
	origTmpDir := smokeTmpDir
	origReadFile := readFile
	origMkdirAll := mkdirAll
	return func() {
		resolveAuth = origResolve
		newTokenSource = origToken
		newSmokeClient = origClient
		runCommand = origRun
		executablePath = origExecutablePath
		smokeTmpDir = origTmpDir
		readFile = origReadFile
		mkdirAll = origMkdirAll
	}
}
