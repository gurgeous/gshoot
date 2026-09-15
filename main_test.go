package main

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/gurgeous/gshoot/gog"
	"github.com/stretchr/testify/assert"
)

func TestFatalTextHandlesNewlines(t *testing.T) {
	got := fatalText(errors.New("missing file\nhint: run `gshoot list`"))
	lines := strings.Split(got, "\n")

	assert.Len(t, lines, 2)
	assert.Contains(t, lines[0], "gshoot: missing file")
	assert.Contains(t, lines[1], "hint: run `gshoot list`")
	assert.NotContains(t, lines[1], "gshoot:")
}

func TestReportErrorGuidesLikelyGogAuthFailures(t *testing.T) {
	var out bytes.Buffer
	reportError(&out, fmt.Errorf("find spreadsheet: %w", &gog.AuthError{
		Message: "No auth for drive me@example.com",
	}))

	text := out.String()
	assert.Contains(t, text, "No auth for drive me@example.com")
	assert.Contains(t, text, "gog drive ls")
}
