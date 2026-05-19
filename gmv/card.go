package gmv

// Card turns lipgloss-rendered ANSI text into a transparent terminal image.
// It owns glyphs and foreground styles only; renderer supplies movie-derived backgrounds.

import (
	"strings"

	lipgloss "charm.land/lipgloss/v2"
	xansi "github.com/charmbracelet/x/ansi"
)

// card is rendered card text plus its terminal dimensions.
type card struct {
	image timage
}

// newCard stores ANSI card text and measures its terminal size.
func newCard(text string) card {
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")

	w := 0
	for _, line := range lines {
		w = max(w, lipgloss.Width(line))
	}
	h := len(lines)
	card := card{}
	card.image.resize(sz(w, h))
	for row, line := range lines {
		card.parseLine(row, line)
	}

	return card
}

// pixelAt returns a card pixel at local terminal coordinates.
func (c card) pixelAt(p point) (tpixel, bool) {
	if !c.image.contains(p) {
		return tpixel{}, false
	}
	px := c.image.at(p)
	return px, px.Ch != ""
}

//
// helpers
//

// parseLine decodes one ANSI-styled line into a card row.
func (c card) parseLine(row int, line string) {
	for col := range c.image.row(row) {
		c.image.set(pt(col, row), tpixel{Ch: " "})
	}

	var buf strings.Builder
	col := 0
	state := xansi.NormalState
	for ii := 0; ii < len(line) && col < c.image.size().X; {
		str, w, n, nxt := xansi.DecodeSequence(line[ii:], state, nil)
		state = nxt
		ii += n

		// ANSI codes are zero-width but still affect the next pixel.
		if w == 0 {
			buf.WriteString(str)
			continue
		}

		c.image.set(pt(col, row), tpixel{
			Style: buf.String(),
			Ch:    str,
		})
		col++
	}
}
