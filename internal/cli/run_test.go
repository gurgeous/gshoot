package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunRootHelp(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"--help"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run() code = %d, want 0", code)
	}

	output := stdout.String()
	for _, want := range []string{"gshoot", "up", "down", "list"} {
		if !strings.Contains(output, want) {
			t.Fatalf("help output missing %q:\n%s", want, output)
		}
	}

	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunUnknownCommand(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"wat"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("Run() code = %d, want 1", code)
	}

	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}

	if !strings.Contains(stderr.String(), "unknown command") {
		t.Fatalf("stderr = %q, want unknown command", stderr.String())
	}
}

func TestRunSubcommandHelp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "up", args: []string{"up", "--help"}, want: "Upload CSV data"},
		{name: "down", args: []string{"down", "--help"}, want: "Download sheet data"},
		{name: "list", args: []string{"list", "--help"}, want: "List recent spreadsheets"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var stdout bytes.Buffer
			var stderr bytes.Buffer

			code := Run(tt.args, &stdout, &stderr)
			if code != 0 {
				t.Fatalf("Run() code = %d, want 0", code)
			}

			if !strings.Contains(stdout.String(), tt.want) {
				t.Fatalf("stdout = %q, want %q", stdout.String(), tt.want)
			}

			if stderr.Len() != 0 {
				t.Fatalf("stderr = %q, want empty", stderr.String())
			}
		})
	}
}
