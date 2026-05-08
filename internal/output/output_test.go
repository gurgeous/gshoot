package output

import (
	"bytes"
	"testing"
)

func TestPrinterPlainOutputOnBuffers(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	p := New(&stdout, &stderr)

	p.Info("hello")
	p.Warn("careful")

	if stdout.String() != "hello\n" {
		t.Fatalf("stdout = %q, want plain output", stdout.String())
	}
	if stderr.String() != "careful\n" {
		t.Fatalf("stderr = %q, want plain output", stderr.String())
	}
	if got := p.SpinnerFrame(0, "working"); got == "" {
		t.Fatal("SpinnerFrame() = empty, want rendered frame")
	}
}
