package util

import (
	"bytes"
	"testing"
)

func TestIndentBlock(t *testing.T) {
	t.Parallel()

	got := IndentBlock("a\nb")
	want := "  a\n  b"
	if got != want {
		t.Fatalf("IndentBlock() = %q, want %q", got, want)
	}
}

func TestRPad(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		text    string
		padding int
		want    string
	}{
		{name: "pads shorter", text: "a", padding: 3, want: "a  "},
		{name: "adds one space when equal", text: "abc", padding: 3, want: "abc "},
		{name: "adds one space when longer", text: "abcd", padding: 3, want: "abcd "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := RPad(tt.text, tt.padding); got != tt.want {
				t.Fatalf("RPad() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		text  string
		width int
		want  string
	}{
		{name: "keeps shorter", text: "abc", width: 5, want: "abc"},
		{name: "truncates ascii", text: "abcdef", width: 5, want: "abcd…"},
		{name: "single cell", text: "abcdef", width: 1, want: "…"},
		{name: "zero width", text: "abcdef", width: 0, want: ""},
		{name: "wide rune", text: "猫猫猫", width: 3, want: "猫…"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := Truncate(tt.text, tt.width); got != tt.want {
				t.Fatalf("Truncate() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHyperlink(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	if got := Hyperlink(&out, "https://example.com", "Alpha"); got != "Alpha" {
		t.Fatalf("Hyperlink() = %q, want plain label", got)
	}
}

func TestIsTTY(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	if IsTty(&out) {
		t.Fatal("IsTTY() = true, want false")
	}
}
