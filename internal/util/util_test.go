package util

import "testing"

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
