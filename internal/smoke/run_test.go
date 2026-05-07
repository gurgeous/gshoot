package smoke

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunRequiresBinary(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run(nil, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("Run() code = %d, want 1", code)
	}

	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}

	if !strings.Contains(stderr.String(), "missing required --gshoot path") {
		t.Fatalf("stderr = %q, want missing path", stderr.String())
	}
}

func TestRunStub(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"--gshoot", "tmp/gshoot"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run() code = %d, want 0", code)
	}

	if !strings.Contains(stdout.String(), "gsmoke stub") {
		t.Fatalf("stdout = %q, want stub message", stdout.String())
	}

	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}
