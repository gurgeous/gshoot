package ux

import (
	"bytes"
	"regexp"

	lipgloss "charm.land/lipgloss/v2"
	"github.com/alecthomas/kong"
)

// HelpPrinter renders Kong help with gshoot's ANSI color rules.
func HelpPrinter(options kong.HelpOptions, ctx *kong.Context) error {
	// render vanilla kong help into tmp buf
	var buf bytes.Buffer
	orig := ctx.Stdout
	ctx.Stdout = &buf
	err := kong.DefaultHelpPrinter(options, ctx)
	ctx.Stdout = orig
	if err != nil {
		return err
	}

	// now add color
	styles := []RestyleRule{
		{Re: regexp.MustCompile(`(?m)^[A-Z][A-Za-z ]*:`), Style: Success},              // `Usage:`
		{Re: regexp.MustCompile(`(?m)^  ([a-z]+(?: [a-z]+)?)\s{2,}.*$`), Style: Brand}, // `  auth login ...`
		{Re: regexp.MustCompile(regexp.QuoteMeta("gshoot")), Style: Brand},             // gshoot
		{Re: regexp.MustCompile(`(?:^|\s)(-{1,2}[A-Za-z0-9=-]+)`), Style: Warn},        // --xxxx=
	}
	help := Restyle(buf.String(), styles)
	_, _ = lipgloss.Fprint(ctx.Stdout, help)
	return nil
}
