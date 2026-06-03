package main

import (
	"errors"
	"strings"
	"testing"

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
