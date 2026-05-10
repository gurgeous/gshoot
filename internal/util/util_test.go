package util

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func TestIndent(t *testing.T) {
	got := Indent("a\nb", "  ")
	want := "  a\n  b"
	if got != want {
		t.Fatalf("Indent() = %q, want %q", got, want)
	}
}

func TestPadRight(t *testing.T) {
	tests := []struct {
		s      string
		length int
		want   string
	}{
		{s: "a", length: 3, want: "a  "},
		{s: "abc", length: 3, want: "abc "},
		{s: "abcd", length: 3, want: "abcd "},
	}

	for _, tt := range tests {
		if got := PadRight(tt.s, tt.length); got != tt.want {
			t.Fatalf("PadRight() = %q, want %q", got, tt.want)
		}
	}
}

func TestFileExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "exists.txt")
	_ = os.WriteFile(path, []byte("x"), 0o600)
	if !FileExists(path) {
		t.Fatalf("FileExists(%q) = false, want true", path)
	}
	if FileExists(filepath.Join(dir, "missing.txt")) {
		t.Fatal("FileExists(missing) = true, want false")
	}
}

func TestWritePrivateFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "secret.txt")
	if err := WritePrivateFile(path, []byte("top secret\n")); err != nil {
		t.Fatalf("WritePrivateFile() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if got, want := string(data), "top secret\n"; got != want {
		t.Fatalf("file contents = %q, want %q", got, want)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if got, want := info.Mode().Perm(), os.FileMode(0o600); got != want {
		t.Fatalf("file mode = %#o, want %#o", got, want)
	}
}

func TestRandomHex(t *testing.T) {
	got, err := RandomHex(16)
	if err != nil {
		t.Fatalf("RandomHex() error = %v", err)
	}
	if len(got) != 32 {
		t.Fatalf("len(RandomHex()) = %d, want 32", len(got))
	}
	if !regexp.MustCompile(`^[0-9a-f]+$`).MatchString(got) {
		t.Fatalf("RandomHex() = %q, want lowercase hex", got)
	}
}

func TestBrowserCommandArgs(t *testing.T) {
	tests := []struct {
		goos     string
		wantName string
		wantArgs []string
	}{
		{goos: "darwin", wantName: "open", wantArgs: []string{"https://example.com"}},
		{goos: "windows", wantName: "rundll32", wantArgs: []string{"url.dll,FileProtocolHandler", "https://example.com"}},
		{goos: "linux", wantName: "xdg-open", wantArgs: []string{"https://example.com"}},
	}

	for _, tt := range tests {
		gotName, gotArgs := browserCommandArgs(tt.goos, "https://example.com")
		if gotName != tt.wantName {
			t.Fatalf("browserCommandArgs(%q) name = %q, want %q", tt.goos, gotName, tt.wantName)
		}
		if len(gotArgs) != len(tt.wantArgs) {
			t.Fatalf("browserCommandArgs(%q) args = %#v, want %#v", tt.goos, gotArgs, tt.wantArgs)
		}
		for i := range gotArgs {
			if gotArgs[i] != tt.wantArgs[i] {
				t.Fatalf("browserCommandArgs(%q) args = %#v, want %#v", tt.goos, gotArgs, tt.wantArgs)
			}
		}
	}
}

func TestHyperlink(t *testing.T) {
	var out bytes.Buffer
	if got := Hyperlink(&out, "https://example.com", "Alpha"); got != "Alpha" {
		t.Fatalf("Hyperlink() = %q, want plain label", got)
	}
}

func TestIsTTY(t *testing.T) {
	var out bytes.Buffer
	if IsTty(&out) {
		t.Fatal("IsTTY() = true, want false")
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		s      string
		length int
		want   string
	}{
		{s: "abc", length: 5, want: "abc"},
		{s: "abcdef", length: 5, want: "abcd…"},
		{s: "abcdef", length: 1, want: "…"},
		{s: "abcdef", length: 0, want: ""},
		{s: "猫猫猫", length: 3, want: "猫…"},
	}

	for _, tt := range tests {
		if got := Truncate(tt.s, tt.length); got != tt.want {
			t.Fatalf("Truncate() = %q, want %q", got, tt.want)
		}
	}
}
