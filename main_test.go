package main

import (
	"bytes"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/gurgeous/gshoot/ux"
	"github.com/stretchr/testify/assert"
)

func TestHelpHidesDemoCommand(t *testing.T) {
	var cli CLI
	var out bytes.Buffer
	parser, err := kong.New(
		&cli,
		kong.Name("gshoot"),
		kong.Help(ux.HelpPrinter),
		kong.ConfigureHelp(kong.HelpOptions{Compact: true}),
		kong.Writers(&out, &out),
		kong.Exit(func(int) {}),
	)
	assert.NoError(t, err)
	ctx, err := kong.Trace(parser, []string{})
	assert.NoError(t, err)

	err = ux.HelpPrinter(kong.HelpOptions{
		Compact:        true,
		ValueFormatter: kong.DefaultHelpValueFormatter,
	}, ctx)

	assert.NoError(t, err)
	assert.Contains(t, out.String(), "auth")
	assert.NotContains(t, out.String(), "demo")
}
