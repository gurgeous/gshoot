package ux

import (
	"os"

	lipgloss "charm.land/lipgloss/v2"
	"github.com/gurgeous/gshoot/env"
)

var (
	Brand   lipgloss.Style
	Dim     lipgloss.Style
	Info    lipgloss.Style
	Success lipgloss.Style
	Warn    lipgloss.Style
	Error   lipgloss.Style
	Subtle  lipgloss.Style
)

// Default to dark. Later, we can look at the terminal to get a better answer.
func init() {
	setStyles(lipgloss.LightDark(true))
}

// Init sets styles from the terminal theme or GSHOOT_THEME.
func Init() {
	if env.GSHOOT_THEME() == "" {
		setStyles(lipgloss.LightDark(lipgloss.HasDarkBackground(os.Stdin, os.Stdout)))
		return
	}
	setStyles(lipgloss.LightDark(env.GSHOOT_THEME() != "light"))
}

// setStyles rebuilds global styles for one light/dark profile.
func setStyles(fn lipgloss.LightDarkFunc) {
	fg := func(light, dark string) lipgloss.Style {
		return lipgloss.NewStyle().Foreground(fn(lipgloss.Color(light), lipgloss.Color(dark)))
	}

	Brand = fg(tailwindColors.Blue.c600, tailwindColors.Blue.c400).Bold(true)
	Dim = fg(tailwindColors.Gray.c400, tailwindColors.Gray.c600)
	Info = fg(tailwindColors.Blue.c600, tailwindColors.Blue.c400).Bold(true)
	Success = fg(tailwindColors.Green.c700, tailwindColors.Green.c400).Bold(true)
	Warn = fg(tailwindColors.Amber.c700, tailwindColors.Amber.c400).Bold(true)
	Error = fg(tailwindColors.Red.c700, tailwindColors.Red.c400).Bold(true)
	Subtle = fg(tailwindColors.Slate.c600, tailwindColors.Slate.c400)
	dotsWithColor = renderDots()
}
