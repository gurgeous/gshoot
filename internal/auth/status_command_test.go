package auth

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestNewStatusCommand(t *testing.T) {
	origStatus := status
	origPrint := printStatus
	status = func() Status {
		return Status{ConfigDir: "/tmp/gshoot", ReadyForLogin: true}
	}
	printStatus = func(w io.Writer, status Status) {
		_, _ = io.WriteString(w, "Status: not logged in yet\n")
	}
	t.Cleanup(func() {
		status = origStatus
		printStatus = origPrint
	})

	var stdout bytes.Buffer
	cmd := NewStatusCommand()
	cmd.SetOut(&stdout)
	cmd.SetArgs(nil)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(stdout.String(), "not logged in") {
		t.Fatalf("stdout = %q, want status output", stdout.String())
	}
}
