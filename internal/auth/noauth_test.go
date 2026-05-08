package auth

import (
	"bytes"
	"strings"
	"testing"
)

func TestWriteNoAuthErrorWithoutOAuthClient(t *testing.T) {
	var stderr bytes.Buffer
	err := &NoAuthError{
		Command:   CommandList,
		ConfigDir: t.TempDir(),
	}

	WriteNoAuthError(&stderr, err)

	got := stderr.String()
	for _, want := range []string{
		"You will need to authenticate first.",
		"setting up auth with Google Sheets is",
		"Don't blame gshoot.",
		"Try this first:",
		"gshoot auth status",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("output missing %q:\n%q", want, got)
		}
	}
}
