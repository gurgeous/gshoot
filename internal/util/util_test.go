package util

import (
	"bytes"
	"os"
	"path/filepath"
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
