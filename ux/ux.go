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

//
// Shared color styles, tiny markdown, and regex restyling.
//

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
	// if we are initing with a theme, use that. otherwise detect from termbg
	var lightDark lipgloss.LightDarkFunc
	if theme != "" {
		lightDark = lipgloss.LightDark(theme != "light")
	} else {
		lightDark = lipgloss.LightDark(lipgloss.HasDarkBackground(os.Stdin, os.Stdout))
	}
	fn := func(light, dark string) color.Color {
		return lightDark(lipgloss.Color(light), lipgloss.Color(dark))
	}

	// is the terminal 256? 16M? No color?
	profile := colorprofile.Detect(os.Stdout, os.Environ())

	// calculate our styles, taking into account term profile and theme
	Brand = textStyle(profile, fn(Tailwind.Blue.C600, Tailwind.Blue.C400), true)
	Muted = textStyle(profile, fn(Tailwind.Gray.C400, Tailwind.Gray.C600), false)
	Success = textStyle(profile, fn(Tailwind.Green.C700, Tailwind.Green.C400), true)
	Warn = textStyle(profile, fn(Tailwind.Amber.C700, Tailwind.Amber.C400), true)
	Error = textStyle(profile, fn(Tailwind.Red.C700, Tailwind.Red.C400), true)
	Fatal = textStyle(profile, lipgloss.Color("#fff"), true).
		Background(downsample(profile, lipgloss.Color(Tailwind.Red.C700))).
		Bold(true)
}

// textStyle builds a fg text style for the active terminal profile.
func textStyle(profile colorprofile.Profile, c color.Color, bold bool) lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(downsample(profile, c)).
		Bold(bold)
}

// downsample converts colors to the detected terminal profile.
func downsample(profile colorprofile.Profile, c color.Color) color.Color {
	switch profile {
	case colorprofile.TrueColor:
		return c
	case colorprofile.ANSI256:
		return profile.Convert(c)
	default:
		return lipgloss.NoColor{}
	}
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
