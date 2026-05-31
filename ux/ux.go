package ux

import (
	"os"
	"regexp"
	"sort"
	"strings"

	lipgloss "charm.land/lipgloss/v2"
	"github.com/gurgeous/gshoot/env"
	"github.com/gurgeous/gshoot/util"
)

const AppName = "gshoot"

var (
	Brand   lipgloss.Style // blue
	Success lipgloss.Style // green
	Warn    lipgloss.Style // orange
	Error   lipgloss.Style // red
	Muted   lipgloss.Style // gray
	Fatal   lipgloss.Style // white on red
)

// Default to dark. Later, we can look at the terminal to get a better answer.
func init() {
	setStyles(lipgloss.LightDark(true))
}

// Init sets up styles from config and terminal background.
func Init(cfg env.Config) {
	if cfg.Theme != "" {
		setStyles(lipgloss.LightDark(cfg.Theme != "light"))
		return
	}

	setStyles(lipgloss.LightDark(lipgloss.HasDarkBackground(os.Stdin, os.Stdout)))
}

// setStyles rebuilds global styles for one light/dark profile.
func setStyles(fn lipgloss.LightDarkFunc) {
	fg := func(light, dark string) lipgloss.Style {
		return lipgloss.NewStyle().Foreground(fn(lipgloss.Color(light), lipgloss.Color(dark)))
	}

	// text styles
	Brand = fg(Tailwind.Blue.C600, Tailwind.Blue.C400).Bold(true)
	Muted = fg(Tailwind.Gray.C400, Tailwind.Gray.C600)
	Success = fg(Tailwind.Green.C700, Tailwind.Green.C400).Bold(true)
	Warn = fg(Tailwind.Amber.C700, Tailwind.Amber.C400).Bold(true)
	Error = fg(Tailwind.Red.C700, Tailwind.Red.C400).Bold(true)
	Fatal = lipgloss.NewStyle().Foreground(lipgloss.Color("white")).Background(lipgloss.Color(Tailwind.Red.C700)).Bold(true)

	// dots
	dotsWithColor = renderDots()
}

//
// styleText
//

type RestyleRule struct {
	Re    *regexp.Regexp
	Style lipgloss.Style
}

// Restyle applies ordered re-style rules to text.
func Restyle(str string, styles []RestyleRule) string {
	type match struct {
		start int
		end   int
		style lipgloss.Style
	}

	// find matches
	matches := []match{}
	for _, style := range styles {
		for _, m := range style.Re.FindAllStringSubmatchIndex(str, -1) {
			var start, end int
			if len(m) == 2 {
				start, end = m[0], m[1]
			} else {
				start, end = m[2], m[3]
			}
			matches = append(matches, match{start: start, end: end, style: style.Style})
		}
	}

	// sort
	sort.SliceStable(matches, func(i, j int) bool { return matches[i].start < matches[j].start })

	// apply matches, ignore overlaps
	pos := 0
	var buf strings.Builder
	for _, m := range matches {
		if m.start < pos {
			continue
		}
		buf.WriteString(str[pos:m.start])
		buf.WriteString(m.style.Render(str[m.start:m.end]))
		pos = m.end
	}
	if pos < len(str) {
		buf.WriteString(str[pos:])
	}
	return buf.String()
}

// Markdown renders the tiny markdown subset used in CLI prose.
func Markdown(str string) string {
	linkRe := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`) // markdown link
	boldRe := regexp.MustCompile(`\*\*([^*]+)\*\*`)         // markdown bold
	italicRe := regexp.MustCompile(`\*([^*]+)\*`)           // markdown italic

	out := linkRe.ReplaceAllStringFunc(str, func(match string) string {
		parts := linkRe.FindStringSubmatch(match)
		return util.RenderHyperlink(parts[2], Brand.Underline(true).Render(parts[1]))
	})
	out = boldRe.ReplaceAllStringFunc(out, func(match string) string {
		parts := boldRe.FindStringSubmatch(match)
		return Brand.Render(parts[1])
	})
	out = italicRe.ReplaceAllStringFunc(out, func(match string) string {
		parts := italicRe.FindStringSubmatch(match)
		return Warn.Render(parts[1])
	})

	return lipgloss.Wrap(out, 72, " ")
}
