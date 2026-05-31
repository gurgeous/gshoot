package ux

import (
	"bytes"
	"regexp"
	"strings"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/assert"
)

type testCLI struct {
	Verbose bool `short:"v" help:"Print more output."`
}

func TestHelpPrinterWritesKongHelp(t *testing.T) {
	var cli testCLI
	var out bytes.Buffer
	parser, err := kong.New(
		&cli,
		kong.Name("gshoot"),
		kong.Help(HelpPrinter),
		kong.Writers(&out, &out),
		kong.Exit(func(int) {}),
	)
	assert.NoError(t, err)
	ctx, err := kong.Trace(parser, []string{})
	assert.NoError(t, err)

	err = HelpPrinter(kong.HelpOptions{
		Compact:        true,
		ValueFormatter: kong.DefaultHelpValueFormatter,
	}, ctx)

	assert.NoError(t, err)
	assert.Contains(t, out.String(), "Usage:")
	assert.Contains(t, out.String(), "gshoot")
	assert.Contains(t, out.String(), "[flags]")
	assert.Contains(t, out.String(), "--verbose")
}

func TestColorizeColorsSectionsCommandsAndFlags(t *testing.T) {
	help := strings.Join([]string{
		"Usage: gshoot up <spreadsheet> <csv> [flags]",
		"",
		"Arguments:",
		"  <spreadsheet>    Spreadsheet name.",
		"",
		"Flags:",
		"  -h, --help            Show context-sensitive help.",
		"      --sheet=STRING    Destination sheet name.",
		"",
		"Commands:",
		"  auth login     Run browser OAuth login.",
	}, "\n") + "\n"

	colored := Restyle(help, helpRules())

	assert.Contains(t, colored, Success.Render("Usage:"))
	assert.Contains(t, colored, Success.Render("Arguments:"))
	assert.Contains(t, colored, Success.Render("Flags:"))
	assert.Contains(t, colored, Brand.Render("gshoot")+" up")
	assert.Contains(t, colored, Brand.Render("auth login"))
	assert.Contains(t, colored, Warn.Render("--help"))
	assert.Contains(t, colored, Warn.Render("--sheet=STRING"))
	assert.Contains(t, colored, "<spreadsheet>")
	assert.Contains(t, colored, "[flags]")
	assert.Contains(t, colored, "Destination sheet name.")
	assert.Contains(t, colored, "context-sensitive")
	assert.NotContains(t, colored, "context"+Warn.Render("-sensitive"))
}

func TestColorHelpHandlesEmptyInput(t *testing.T) {
	assert.Equal(t, "", Restyle("", helpRules()))
}

func TestColorHelpHandlesUsageWithoutCommand(t *testing.T) {
	colored := Restyle("Usage: [--flag]\n", helpRules())

	assert.Contains(t, colored, Success.Render("Usage:"))
	assert.Contains(t, colored, "[--flag]")
}

func TestColorHelpLeavesCommandLikeProseAlone(t *testing.T) {
	colored := Restyle("auth login runs browser OAuth login.\n", helpRules())

	assert.Equal(t, "auth login runs browser OAuth login.\n", colored)
}

func helpRules() []RestyleRule {
	return []RestyleRule{
		{Re: regexp.MustCompile(`(?m)^[A-Z][A-Za-z ]*:`), Style: Success},
		{Re: regexp.MustCompile(`(?m)^  ([a-z]+(?: [a-z]+)?)\s{2,}.*$`), Style: Brand},
		{Re: regexp.MustCompile(regexp.QuoteMeta("gshoot")), Style: Brand},
		{Re: regexp.MustCompile(`(?:^|\s)(-{1,2}[A-Za-z0-9=-]+)`), Style: Warn},
	}
}

func TestStyleTextAppliesWholeAndGroupedMatches(t *testing.T) {
	styled := Restyle("alpha --flag beta", []RestyleRule{
		{Re: regexp.MustCompile(`alpha`), Style: Success},
		{Re: regexp.MustCompile(`\s(--flag)`), Style: Warn},
	})

	assert.Equal(t, Success.Render("alpha")+" "+Warn.Render("--flag")+" beta", styled)
}

func TestMarkdownStylesBoldAndLinks(t *testing.T) {
	rendered := Markdown("Read **the docs**, *carefully*, or [GitHub](https://example.com).")

	assert.Contains(t, rendered, Brand.Render("the docs"))
	assert.Contains(t, rendered, Warn.Render("carefully"))
	assert.Contains(t, rendered, "\x1b]8;;https://example.com\x1b\\")
	assert.Contains(t, rendered, Brand.Underline(true).Render("GitHub"))
	assert.Contains(t, rendered, "\x1b]8;;\x1b\\")
	assert.NotContains(t, rendered, "**")
	assert.NotContains(t, rendered, "*carefully*")
	assert.NotContains(t, rendered, "[GitHub]")
}
