package auth

import (
	"bytes"
	"testing"
)

func TestWriteNoAuthErrorWithoutOAuthClient(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer
	err := &NoAuthError{
		Command:   CommandList,
		ConfigDir: t.TempDir(),
	}

	WriteNoAuthError(&stderr, err)

	if got, want := stderr.String(), "You will need to authenticate first.\n\nI apologize in advance, setting up auth with Google Sheets is\nannoyingly difficult for some reason. Don't blame gshoot.\n\nTry this first:\n\ngshoot auth status\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestWriteNoAuthErrorWithOAuthClient(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer
	err := &NoAuthError{
		Command:   CommandDown,
		ConfigDir: t.TempDir(),
	}

	WriteNoAuthError(&stderr, err)

	if got, want := stderr.String(), "You will need to authenticate first.\n\nI apologize in advance, setting up auth with Google Sheets is\nannoyingly difficult for some reason. Don't blame gshoot.\n\nTry this first:\n\ngshoot auth status\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}
