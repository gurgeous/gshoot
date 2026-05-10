package authcmd

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/gurgeous/gshoot/internal/auth"
)

func TestNewCommandStatus(t *testing.T) {
	origStatus := status
	origPrint := printStatus
	status = func() auth.Status {
		return auth.Status{ConfigDir: "/tmp/gshoot", ReadyForLogin: true}
	}
	printStatus = func(w io.Writer, status auth.Status) {
		_, _ = io.WriteString(w, "Status: not logged in yet\n")
	}
	t.Cleanup(func() {
		status = origStatus
		printStatus = origPrint
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := NewCommand()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"status"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(stdout.String(), "not logged in") {
		t.Fatalf("stdout = %q, want status output", stdout.String())
	}
}

func TestNewCommandIncludesSubcommands(t *testing.T) {
	cmd := NewCommand()
	for _, want := range []string{"login", "status", "logout"} {
		if _, _, err := cmd.Find([]string{want}); err != nil {
			t.Fatalf("Find(%q) error = %v", want, err)
		}
	}
}
