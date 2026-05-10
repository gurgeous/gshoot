package sub

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewLogoutCommand(t *testing.T) {
	orig := runLogout
	runLogout = func() (bool, error) { return true, nil }
	t.Cleanup(func() {
		runLogout = orig
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := newLogoutCommand()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(stdout.String(), "Removed cached OAuth token") {
		t.Fatalf("stdout = %q, want logout message", stdout.String())
	}
}
