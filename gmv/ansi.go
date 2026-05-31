package gmv

import (
	"bytes"
	"image/color"

	lipgloss "charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
)

//
// ANSI centralizes terminal color conversion for GMV rendering.
//

const (
	FG    = "\x1b[38;2;" // truecolor foreground SGR sequence.
	BG    = "\x1b[48;2;" // truecolor background SGR sequence.
	FG256 = "\x1b[38;5;" // indexed-color foreground SGR sequence.
	BG256 = "\x1b[48;5;" // indexed-color background SGR sequence.
)

var (
	// cardBackgroundColor is the opaque card fill for non-alpha mode.
	cardBackgroundColor = color.RGBA{R: 31, G: 34, B: 38, A: 255}
	// statsColor is the foreground color for demo stats.
	statsColor = color.RGBA{R: 229, G: 231, B: 235, A: 255}
)

//
// writing colors to buffers
//

// ansiTruecolor writes a truecolor foreground or background SGR sequence.
func ansiTruecolor(out *bytes.Buffer, fg bool, c color.RGBA) {
	if fg {
		out.WriteString(FG)
	} else {
		out.WriteString(BG)
	}
	writeUint8(out, c.R)
	out.WriteByte(';')
	writeUint8(out, c.G)
	out.WriteByte(';')
	writeUint8(out, c.B)
	out.WriteByte('m')
}

// ansi256 writes an indexed-color foreground or background SGR sequence.
func ansi256(out *bytes.Buffer, fg bool, c lipgloss.ANSIColor) {
	if fg {
		out.WriteString(FG256)
	} else {
		out.WriteString(BG256)
	}
	writeUint8(out, uint8(c))
	out.WriteByte('m')
}

// writeUint8 writes a decimal uint8 without allocating.
func writeUint8(out *bytes.Buffer, n uint8) {
	if n >= 10 {
		if n >= 100 {
			out.WriteByte('0' + n/100)
			n %= 100
		}
		out.WriteByte('0' + n/10)
		n %= 10
	}
	out.WriteByte('0' + n)
}

//
// palette
//

// paletteColor stores both RGB and its background escape.
type paletteColor struct {
	Color  color.RGBA
	Escape string
}

// newPaletteColor caches color data for one terminal palette entry.
func newPaletteColor(profile colorprofile.Profile, c color.RGBA) paletteColor {
	display := displayColor(profile, c)
	return paletteColor{
		Color:  rgba(display),
		Escape: backgroundEscape(profile, display),
	}
}

// foregroundEscape returns a foreground escape for the profile.
func foregroundEscape(profile colorprofile.Profile, c color.Color) string {
	return colorEscape(profile, c, true)
}

// backgroundEscape returns a background escape for the profile.
func backgroundEscape(profile colorprofile.Profile, c color.Color) string {
	return colorEscape(profile, c, false)
}

// colorEscape returns a foreground or background escape for the profile.
func colorEscape(profile colorprofile.Profile, c color.Color, fg bool) string {
	var out bytes.Buffer
	if profile == colorprofile.TrueColor {
		ansiTruecolor(&out, fg, rgba(c))
	} else if indexed, ok := c.(lipgloss.ANSIColor); ok {
		ansi256(&out, fg, indexed)
	} else if indexed, ok := displayColor(profile, rgba(c)).(lipgloss.ANSIColor); ok {
		ansi256(&out, fg, indexed)
	}
	return out.String()
}

// displayColor returns the color as emitted for the configured profile.
func displayColor(profile colorprofile.Profile, c color.RGBA) color.Color {
	if profile == colorprofile.TrueColor {
		return c
	}
	return colorprofile.ANSI256.Convert(c)
}

// downsample rewrites embedded truecolor SGR sequences for the profile.
func downsample(s string, profile colorprofile.Profile) string {
	if profile == colorprofile.TrueColor || s == "" {
		return s
	}

	var out bytes.Buffer
	writer := colorprofile.Writer{
		Forward: &out,
		Profile: profile,
	}
	_, _ = writer.Write([]byte(s))
	return out.String()
}

// dim darkens an RGB color for card compositing.
func dim(c color.RGBA, amount float64) color.RGBA {
	return color.RGBA{
		R: uint8(float64(c.R) * amount),
		G: uint8(float64(c.G) * amount),
		B: uint8(float64(c.B) * amount),
		A: c.A,
	}
}

// rgba converts any Go color into 8-bit RGBA.
func rgba(c color.Color) color.RGBA {
	r, g, b, a := c.RGBA()
	return color.RGBA{
		R: uint8(r >> 8),
		G: uint8(g >> 8),
		B: uint8(b >> 8),
		A: uint8(a >> 8),
	}
}
