package ux

import (
	"image/color"
	"os"
	"regexp"
	"sort"
	"strings"

	lipgloss "charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/gurgeous/gshoot/util"
)

var (
	Brand   lipgloss.Style // blue
	Success lipgloss.Style // green
	Warn    lipgloss.Style // orange
	Error   lipgloss.Style // red
	Muted   lipgloss.Style // gray
	Fatal   lipgloss.Style // white on red
)

// Init sets up styles from config and terminal background.
func Init(theme string) {
	profile := colorprofile.Detect(os.Stdout, os.Environ())

	// if we are initing with a theme, use that. otherwise detect from termbg
	var fn lipgloss.LightDarkFunc
	if theme != "" {
		fn = lipgloss.LightDark(theme != "light")
	} else {
		fn = lipgloss.LightDark(lipgloss.HasDarkBackground(os.Stdin, os.Stdout))
	}

	// tiny helper for clearing up boilerplate
	fg := func(light, dark string) lipgloss.Style {
		return lipgloss.NewStyle().Foreground(downsampleColor(profile, fn(lipgloss.Color(light), lipgloss.Color(dark))))
	}

	// styles
	Brand = fg(Tailwind.Blue.C600, Tailwind.Blue.C400).Bold(true)
	Muted = fg(Tailwind.Gray.C400, Tailwind.Gray.C600)
	Success = fg(Tailwind.Green.C700, Tailwind.Green.C400).Bold(true)
	Warn = fg(Tailwind.Amber.C700, Tailwind.Amber.C400).Bold(true)
	Error = fg(Tailwind.Red.C700, Tailwind.Red.C400).Bold(true)
	Fatal = lipgloss.NewStyle().
		Foreground(downsampleColor(profile, lipgloss.Color("#fff"))).
		Background(downsampleColor(profile, lipgloss.Color(Tailwind.Red.C700))).
		Bold(true)
}

// downsampleColor converts colors to the detected terminal profile.
func downsampleColor(profile colorprofile.Profile, c color.Color) color.Color {
	switch profile {
	case colorprofile.TrueColor:
		return c
	case colorprofile.ANSI256, colorprofile.ANSI:
		converted := profile.Convert(c)
		if converted != nil {
			return converted
		}
	case colorprofile.Unknown, colorprofile.NoTTY, colorprofile.ASCII:
		return lipgloss.NoColor{}
	}
	return lipgloss.NoColor{}
}

//
// Restyle
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

//
// Markdown renders a tiny markdown subset
//

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
