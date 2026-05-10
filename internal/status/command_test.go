package status

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/gurgeous/gshoot/internal/auth"
)

func TestNewStatusCommand(t *testing.T) {
	origStatus := runStatus
	origPrint := writeStatus
	runStatus = func() auth.Status {
		return auth.Status{ConfigDir: "/tmp/gshoot", ReadyForLogin: true}
	}
	writeStatus = func(w io.Writer, status auth.Status) {
		_, _ = io.WriteString(w, "Status: not logged in yet\n")
	}
	t.Cleanup(func() {
		runStatus = origStatus
		writeStatus = origPrint
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
