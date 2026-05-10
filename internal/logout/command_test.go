package logout

import (
	"bytes"
	"testing"
)

func TestNewCommand(t *testing.T) {
	orig := runLogout
	runLogout = func() (bool, error) { return true, nil }
	t.Cleanup(func() {
		runLogout = orig
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := NewCommand()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if stdout.Len() == 0 {
		t.Fatal("stdout = empty, want logout message")
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}
