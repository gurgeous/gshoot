package ux

import (
	"testing"

	lipgloss "charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/stretchr/testify/assert"
)

func TestDownsampleConvertsColorProfiles(t *testing.T) {
	c := lipgloss.Color("#60a5fa")

	assert.Equal(t, c, downsample(colorprofile.TrueColor, c))
	assert.IsType(t, lipgloss.ANSIColor(0), downsample(colorprofile.ANSI256, c))
	assert.Equal(t, lipgloss.NoColor{}, downsample(colorprofile.NoTTY, c))
	assert.Equal(t, lipgloss.NoColor{}, downsample(colorprofile.ASCII, c))
}

func TestBrandRenderMatchesColorProfile(t *testing.T) {
	c := lipgloss.Color("#60a5fa")

	truecolor := textStyle(colorprofile.TrueColor, c, true).Render("brand")
	ansi256 := textStyle(colorprofile.ANSI256, c, true).Render("brand")
	noColor := textStyle(colorprofile.NoTTY, c, true).Render("brand")

	assert.Contains(t, truecolor, "38;2;96;165;250")
	assert.NotContains(t, truecolor, "\x1b[38;5;")
	assert.Contains(t, ansi256, "38;5;")
	assert.NotContains(t, ansi256, "\x1b[38;2;")
	assert.NotContains(t, noColor, "\x1b[38;")
	assert.Contains(t, noColor, "\x1b[1m")
}
