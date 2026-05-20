package util

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
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

func TestCSVString(t *testing.T) {
	got := CSVString([][]string{{"name", "count"}, {"alpha", "1"}})

	assert.Equal(t, "name,count\nalpha,1\n", got)
}

func TestStringSliceHelpers(t *testing.T) {
	values := []string{"alpha", "beta", "gamma"}

	assert.Equal(t, 1, IndexOfString(values, "beta"))
	assert.Equal(t, -1, IndexOfString(values, "missing"))
	assert.True(t, ContainsString(values, "gamma"))
	assert.False(t, ContainsString(values, "missing"))
	assert.True(t, AnyContains(values, "mm"))
	assert.False(t, AnyContains(values, "zz"))
}

func TestAllMatch(t *testing.T) {
	re := regexp.MustCompile(`^\d+$`)

	assert.True(t, AllMatch([]string{"1", "22"}, re))
	assert.False(t, AllMatch([]string{"1", "no"}, re))
}

func TestDecimalPrecision(t *testing.T) {
	assert.Equal(t, 1, DecimalPrecision([]string{"1", "2"}))
	assert.Equal(t, 3, DecimalPrecision([]string{"1.2", "3.456"}))
	assert.Equal(t, 4, DecimalPrecision([]string{"1.23456"}))
}
