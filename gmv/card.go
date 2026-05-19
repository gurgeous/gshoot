package gmv

// Card turns lipgloss-rendered ANSI text into a transparent terminal image.
// It owns glyphs and foreground styles only; renderer supplies movie-derived backgrounds.
// Card text is curated single-width text; wide glyphs are intentionally unsupported.

import (
	"strconv"
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

	parser := xansi.NewParser()
	style := sgrStyle{}
	col := 0
	state := xansi.NormalState
	for ii := 0; ii < len(line) && col < c.image.size().X; {
		str, w, n, nxt := xansi.DecodeSequence(line[ii:], state, parser)
		state = nxt
		ii += n

		// SGR codes are zero-width but still affect the next pixel.
		if w == 0 {
			if xansi.Cmd(parser.Command()).Final() == 'm' {
				style.apply(parser.Params())
			}
			continue
		}

		c.image.set(pt(col, row), tpixel{
			Style: style.String(),
			Ch:    str,
		})
		col++
	}
}

// sgrStyle is the effective foreground text style at one point in a card line.
type sgrStyle struct {
	intensity      string
	italic         string
	underline      string
	blink          string
	reverse        string
	conceal        string
	strike         string
	foreground     string
	underlineColor string
}

// apply updates the effective style from one SGR parameter list.
func (style *sgrStyle) apply(params xansi.Params) {
	if len(params) == 0 {
		style.reset()
		return
	}

	for ii := 0; ii < len(params); ii++ {
		attr, _ := sgrParam(params, ii)
		switch {
		case attr == 0:
			style.reset()
		case attr == 1 || attr == 2:
			style.intensity = strconv.Itoa(attr)
		case attr == 22:
			style.intensity = ""
		case attr == 3:
			style.italic = "3"
		case attr == 23:
			style.italic = ""
		case attr == 4 || attr == 21:
			style.underline = strconv.Itoa(attr)
		case attr == 24:
			style.underline = ""
		case attr == 5 || attr == 6:
			style.blink = strconv.Itoa(attr)
		case attr == 25:
			style.blink = ""
		case attr == 7:
			style.reverse = "7"
		case attr == 27:
			style.reverse = ""
		case attr == 8:
			style.conceal = "8"
		case attr == 28:
			style.conceal = ""
		case attr == 9:
			style.strike = "9"
		case attr == 29:
			style.strike = ""
		case (attr >= 30 && attr <= 37) || (attr >= 90 && attr <= 97):
			style.foreground = strconv.Itoa(attr)
		case attr == 38:
			style.foreground, ii = sgrExtended(params, ii, attr)
		case attr == 39:
			style.foreground = ""
		case attr == 48:
			_, ii = sgrExtended(params, ii, attr)
		case attr == 58:
			style.underlineColor, ii = sgrExtended(params, ii, attr)
		case attr == 59:
			style.underlineColor = ""
		}
	}
}

// reset clears all effective style state.
func (style *sgrStyle) reset() {
	*style = sgrStyle{}
}

// String returns the minimal SGR sequence for the effective style.
func (style sgrStyle) String() string {
	parts := []string{
		style.intensity,
		style.italic,
		style.underline,
		style.blink,
		style.reverse,
		style.conceal,
		style.strike,
		style.foreground,
		style.underlineColor,
	}

	var out strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		if out.Len() == 0 {
			out.WriteString("\x1b[")
		} else {
			out.WriteByte(';')
		}
		out.WriteString(part)
	}
	if out.Len() == 0 {
		return ""
	}
	out.WriteByte('m')
	return out.String()
}

// sgrParam returns one unpacked SGR parameter.
func sgrParam(params xansi.Params, ii int) (int, bool) {
	param, _, ok := params.Param(ii, 0)
	return param, ok
}

// sgrExtended returns a compact extended color SGR parameter.
func sgrExtended(params xansi.Params, ii int, attr int) (string, int) {
	mode, ok := sgrParam(params, ii+1)
	if !ok {
		return "", ii
	}

	switch mode {
	case 5:
		color, ok := sgrParam(params, ii+2)
		if !ok {
			return "", ii
		}
		return strconv.Itoa(attr) + ";5;" + strconv.Itoa(color), ii + 2
	case 2:
		r, okR := sgrParam(params, ii+2)
		g, okG := sgrParam(params, ii+3)
		b, okB := sgrParam(params, ii+4)
		if !okR || !okG || !okB {
			return "", ii
		}
		return strconv.Itoa(attr) + ";2;" + strconv.Itoa(r) + ";" + strconv.Itoa(g) + ";" + strconv.Itoa(b), ii + 4
	default:
		return "", ii
	}
}
