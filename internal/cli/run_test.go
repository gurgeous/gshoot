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
	for _, want := range []string{
		"Usage: gshoot <command> [flags]",
		"Flags:",
		"  -h, --help     help for gshoot",
		"  -v, --version  print version number",
		"Commands:",
		"  auth           Login (or logout) from Google Sheets",
		"  up             Upload a local CSV file to a Google Sheet",
		"  down           Download a Google Sheet as CSV",
		"  list           List your Google Sheets",
		`Run "gshoot <command> --help" for more information on a command.`,
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("help output missing %q:\n%s", want, output)
		}
	}

	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunVersion(t *testing.T) {
	t.Parallel()

	origVersion := version
	version = "1.2.3"
	defer func() { version = origVersion }()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"--version"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run() code = %d, want 0", code)
	}
	if got, want := stdout.String(), "gshoot 1.2.3\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
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

	got := stderr.String()
	for _, want := range []string{
		`gshoot: unknown command "wat"`,
		helpHint,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("stderr = %q, want %q", got, want)
		}
	}
}

func TestRunDownMissingArgs(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"down"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("Run() code = %d, want 1", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if got := stderr.String(); got != "gshoot: expected `gshoot down <spreadsheet> [sheet]`\n"+helpHint+"\n" {
		t.Fatalf("stderr = %q", got)
	}
}

func TestRunSubcommandHelp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		want []string
	}{
		{name: "auth", args: []string{"auth", "--help"}, want: []string{"Login (or logout) from Google Sheets", "USAGE", "COMMANDS"}},
		{name: "auth status", args: []string{"auth", "status", "--help"}, want: []string{"Show auth status", "USAGE"}},
		{name: "auth logout", args: []string{"auth", "logout", "--help"}, want: []string{"Clear cached OAuth token", "USAGE"}},
		{name: "up", args: []string{"up", "--help"}, want: []string{"Upload a local CSV file to a Google Sheet", "USAGE"}},
		{name: "down", args: []string{"down", "--help"}, want: []string{"Download a Google Sheet as CSV", "USAGE", "FLAGS", "EXAMPLES"}},
		{name: "list", args: []string{"list", "--help"}, want: []string{"List your Google Sheets", "USAGE"}},
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

			for _, want := range tt.want {
				if !strings.Contains(stdout.String(), want) {
					t.Fatalf("stdout = %q, want %q", stdout.String(), want)
				}
			}

			if stderr.Len() != 0 {
				t.Fatalf("stderr = %q, want empty", stderr.String())
			}
		})
	}
}
