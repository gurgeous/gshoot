package gmv

import (
	"bytes"
	"image"
	"image/color"
	"testing"

	"github.com/charmbracelet/colorprofile"
	"github.com/stretchr/testify/assert"
)

func TestRenderCompositesCardWithDimBackground(t *testing.T) {
	normal := color.RGBA{R: 100, G: 80, B: 60, A: 255}
	dim := color.RGBA{R: 35, G: 28, B: 21, A: 255}
	movie := &movie{
		Size:       sz(3, 1),
		Frames:     1,
		pix:        []uint8{0, 0, 0},
		stride:     3,
		bounds:     image.Rect(0, 0, 3, 1),
		palette:    []color.RGBA{normal},
		dimPalette: []color.RGBA{dim},
	}
	card := newCard("X")
	renderer := newRenderer(movie, config{profile: colorprofile.TrueColor, alphaBlend: true}, sz(3, 1))

	var out bytes.Buffer
	renderer.render(&out, 0, card, statsTracker{})

	assert.Contains(t, out.String(), backgroundEscape(colorprofile.TrueColor, dim)+"X")
}

func TestRenderCanSkipCardDimming(t *testing.T) {
	normal := color.RGBA{R: 100, G: 80, B: 60, A: 255}
	dim := color.RGBA{R: 35, G: 28, B: 21, A: 255}
	movie := &movie{
		Size:       sz(3, 1),
		Frames:     1,
		pix:        []uint8{0, 0, 0},
		stride:     3,
		bounds:     image.Rect(0, 0, 3, 1),
		palette:    []color.RGBA{normal},
		dimPalette: []color.RGBA{dim},
	}
	card := newCard("X")
	renderer := newRenderer(movie, config{profile: colorprofile.TrueColor, alphaBlend: false}, sz(3, 1))

	var out bytes.Buffer
	renderer.render(&out, 0, card, statsTracker{})

	assert.NotContains(t, out.String(), backgroundEscape(colorprofile.TrueColor, dim))
	assert.Contains(t, out.String(), renderer.cardBG.Escape+"X")
}

func TestRenderOnlyWritesChangedPixels(t *testing.T) {
	red := color.RGBA{R: 100, A: 255}
	blue := color.RGBA{B: 100, A: 255}
	movie := &movie{
		Size:       sz(2, 1),
		Frames:     2,
		pix:        []uint8{0, 0, 0, 1},
		stride:     4,
		bounds:     image.Rect(0, 0, 4, 1),
		palette:    []color.RGBA{red, blue},
		dimPalette: []color.RGBA{red, blue},
	}
	card := newCard("")
	renderer := newRenderer(movie, config{profile: colorprofile.TrueColor}, sz(2, 1))

	var out bytes.Buffer
	renderer.render(&out, 0, card, statsTracker{})
	out.Reset()

	renderer.render(&out, 1, card, statsTracker{})

	assert.Contains(t, out.String(), "\x1b[1;2H"+backgroundEscape(colorprofile.TrueColor, blue))
	assert.NotContains(t, out.String(), "\x1b[1;1H")
}

func TestRenderDowngradesColorsToANSI256(t *testing.T) {
	orange := color.RGBA{R: 255, G: 133, B: 55, A: 255}
	movie := &movie{
		Size:       sz(1, 1),
		Frames:     1,
		pix:        []uint8{0},
		stride:     1,
		bounds:     image.Rect(0, 0, 1, 1),
		palette:    []color.RGBA{orange},
		dimPalette: []color.RGBA{orange},
	}

	renderer := newRenderer(movie, config{profile: colorprofile.ANSI}, sz(1, 1))

	assert.Contains(t, renderer.palette[0].Escape, "\x1b[48;5;")
	assert.NotContains(t, renderer.palette[0].Escape, "48;2")
	assert.NotEqual(t, orange, renderer.palette[0].Color)
}

func TestPlaybackDowngradesCardANSIToANSI256(t *testing.T) {
	cardText := downsample("\x1b[38;2;255;133;55mX", config{profile: colorprofile.ANSI}.colorProfile())
	card := newCard(cardText)
	px, ok := card.pixelAt(pt(0, 0))

	assert.True(t, ok)
	assert.Contains(t, px.Style, "\x1b[38;5;")
	assert.NotContains(t, px.Style, "38;2")
}

func TestRenderSkipsSmallColorChanges(t *testing.T) {
	dark := color.RGBA{R: 10, G: 10, B: 10, A: 255}
	close := color.RGBA{R: 12, G: 12, B: 12, A: 255}
	movie := &movie{
		Size:       sz(1, 1),
		Frames:     2,
		pix:        []uint8{0, 1},
		stride:     2,
		bounds:     image.Rect(0, 0, 2, 1),
		palette:    []color.RGBA{dark, close},
		dimPalette: []color.RGBA{dark, close},
	}
	card := newCard("")
	renderer := newRenderer(movie, config{profile: colorprofile.TrueColor, diffThreshold: 10}, sz(1, 1))

	var out bytes.Buffer
	renderer.render(&out, 0, card, statsTracker{})
	out.Reset()

	renderer.render(&out, 1, card, statsTracker{})

	assert.Empty(t, out.String())
}

func TestPixelWriterRestoresBackgroundAfterStyleReset(t *testing.T) {
	bg := paletteColor{Escape: "bg"}
	var out bytes.Buffer
	writer := newPixelWriter(&out)

	writer.write(tpixel{Ch: "X", Color: bg, Style: "\x1b[0m"})

	assert.Equal(t, "bg\x1b[0mbgX", out.String())
}

func TestStatsRenderInLowerRight(t *testing.T) {
	black := color.RGBA{A: 255}
	movie := &movie{
		Size:       sz(8, 3),
		Frames:     1,
		pix:        make([]uint8, 24),
		stride:     8,
		bounds:     image.Rect(0, 0, 8, 3),
		palette:    []color.RGBA{black},
		dimPalette: []color.RGBA{black},
	}
	renderer := newRenderer(movie, config{profile: colorprofile.TrueColor}, sz(8, 3))

	renderer.drawFrame(0)
	statsImage := statsTracker{line: "18fps"}.image(renderer.statsStyle, renderer.cardBG)
	renderer.drawImage(statsImage, bottomRight(renderer.next.size(), statsImage.size()), sourceOver)
	row := renderer.next.row(renderer.draw.Dy() - 1)

	for col, char := range " 18fps " {
		assert.Equal(t, string(char), row[1+col].Ch)
	}
	assert.Equal(t, " ", renderer.next.at(pt(0, 0)).Ch)
}
