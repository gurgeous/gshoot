package down

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/gurgeous/gshoot/internal/env"
	"github.com/gurgeous/gshoot/internal/google"
	"github.com/gurgeous/gshoot/internal/testutil"
)

func TestNewCommandStdout(t *testing.T) {
	restore := stubDownload(t, func(spreadsheetName, sheetName string) ([][]string, error) {
		if spreadsheetName != "Budget" || sheetName != "" {
			t.Fatalf("Download() args = (%q, %q)", spreadsheetName, sheetName)
		}
		return [][]string{{"name", "count"}, {"alpha", "1"}}, nil
	})
	defer restore()
	testutil.WithEnv(t, map[string]string{"GSHOOT_TOKEN": "token", "HOME": t.TempDir()}, envVars())

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
	restore := stubDownload(t, func(_, _ string) ([][]string, error) {
		return [][]string{{"name", "count"}, {"alpha", "1"}}, nil
	})
	defer restore()
	testutil.WithEnv(t, map[string]string{"GSHOOT_TOKEN": "token", "HOME": t.TempDir()}, envVars())

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

func stubDownload(t *testing.T, fn func(spreadsheetName, sheetName string) ([][]string, error)) func() {
	t.Helper()

	origDownload := downloadSheet
	downloadSheet = func(_ context.Context, _ *google.Client, spreadsheetName, sheetName string) ([][]string, error) {
		return fn(spreadsheetName, sheetName)
	}
	return func() {
		downloadSheet = origDownload
	}
}

func envVars() map[string]*string {
	return map[string]*string{
		"GOOGLE_APPLICATION_CREDENTIALS": &env.GOOGLE_APPLICATION_CREDENTIALS,
		"GSHOOT_CONFIG_DIR":              &env.GSHOOT_CONFIG_DIR,
		"GSHOOT_CREDENTIALS_FILE":        &env.GSHOOT_CREDENTIALS_FILE,
		"GSHOOT_THEME":                   &env.GSHOOT_THEME,
		"GSHOOT_TOKEN":                   &env.GSHOOT_TOKEN,
	}
}
