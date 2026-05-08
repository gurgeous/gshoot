package smoke

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunRequiresBinary(t *testing.T) {
	t.Parallel()

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

func TestRunStub(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"--gshoot", "tmp/gshoot"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run() code = %d, want 0", code)
	}

	if !strings.Contains(stdout.String(), "gsmoke stub") {
		t.Fatalf("stdout = %q, want stub message", stdout.String())
	}

	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunStubWithCommand(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"--", "go", "run", "./cmd/gshoot"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run() code = %d, want 0", code)
	}

	if !strings.Contains(stdout.String(), "gsmoke stub: using go run ./cmd/gshoot") {
		t.Fatalf("stdout = %q, want command message", stdout.String())
	}

	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunInfersSiblingBinary(t *testing.T) {
	t.Parallel()

	exeDir := t.TempDir()
	exePath := writeExecutable(t, filepath.Join(exeDir, "smoke"))
	gshootPath := writeExecutable(t, filepath.Join(exeDir, "gshoot"))

	restore := swapExecutablePath(t, exePath)
	defer restore()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run(nil, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run() code = %d, want 0", code)
	}

	if !strings.Contains(stdout.String(), "gsmoke stub: using "+gshootPath) {
		t.Fatalf("stdout = %q, want inferred sibling path", stdout.String())
	}

	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func writeExecutable(t *testing.T, path string) string {
	t.Helper()

	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}

func swapExecutablePath(t *testing.T, path string) func() {
	t.Helper()

	original := executablePath
	executablePath = func() (string, error) { return path, nil }
	return func() { executablePath = original }
}
