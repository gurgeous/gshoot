package sub

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/gurgeous/gshoot/internal/google"
)

func TestDownCommandStdout(t *testing.T) {
	restore := stubDownload(t, func(spreadsheetName, sheetName string) ([][]string, error) {
		if spreadsheetName != "Budget" || sheetName != "" {
			t.Fatalf("Download() args = (%q, %q)", spreadsheetName, sheetName)
		}
		return [][]string{{"name", "count"}, {"alpha", "1"}}, nil
	})
	defer restore()
	withRawTokenAuth(t)

	code, stdout, stderr := testMain("down", "Budget")
	if code != 0 {
		t.Fatalf("Main() code = %d, want 0", code)
	}
	if got, want := stdout, "name,count\nalpha,1\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
}

func TestDownCommandOutputFile(t *testing.T) {
	restore := stubDownload(t, func(_, _ string) ([][]string, error) {
		return [][]string{{"name", "count"}, {"alpha", "1"}}, nil
	})
	defer restore()
	withRawTokenAuth(t)

	path := filepath.Join(t.TempDir(), "out.csv")
	code, stdout, stderr := testMain("down", "Budget", "--output", path)
	if code != 0 {
		t.Fatalf("Main() code = %d, want 0", code)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if got, want := string(data), "name,count\nalpha,1\n"; got != want {
		t.Fatalf("file = %q, want %q", got, want)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
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
