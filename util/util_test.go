package util

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatInt(t *testing.T) {
	assert.Equal(t, "0", FormatInt(0))
	assert.Equal(t, "999", FormatInt(999))
	assert.Equal(t, "1,000", FormatInt(1000))
	assert.Equal(t, "1,234,567", FormatInt(1234567))
}

func TestClamp(t *testing.T) {
	assert.Equal(t, 1, Clamp(0, 1, 100))
	assert.Equal(t, 50, Clamp(50, 1, 100))
	assert.Equal(t, 100, Clamp(101, 1, 100))
}

func TestRenderHyperlink(t *testing.T) {
	got := RenderHyperlink("https://example.com", "Alpha")
	want := OSC + "8;;https://example.com" + ST + "Alpha" + OSC + "8;;" + ST
	if got != want {
		t.Fatalf("RenderHyperlink() = %q, want %q", got, want)
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

func TestCSVReadRectangularizesRows(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.csv")
	assert.NoError(t, os.WriteFile(path, []byte("a,b\n1\n2,3,4\n"), 0o600))

	rows, err := CSVRead(path)

	assert.NoError(t, err)
	assert.Equal(t, [][]string{
		{"a", "b", ""},
		{"1", "", ""},
		{"2", "3", "4"},
	}, rows)
}

func TestDecimalPrecision(t *testing.T) {
	assert.Equal(t, 1, DecimalPrecision([]string{"1", "2"}))
	assert.Equal(t, 3, DecimalPrecision([]string{"1.2", "3.456"}))
	assert.Equal(t, 4, DecimalPrecision([]string{"1.23456"}))
}
