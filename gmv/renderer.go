package gmv

// Renderer converts composited terminal images into ANSI output.
// It keeps previous and next images, writes keyframes after invalidation, and otherwise emits dirty spans to reduce bytes.

import (
	"bytes"
	"image/color"

	xansi "github.com/charmbracelet/x/ansi"
)

// renderer caches palettes and previous terminal state.
type renderer struct {
	movie         *movie         // decoded source animation
	palette       []paletteColor // terminal escapes for source colors
	dimPalette    []paletteColor // dimmed colors for card backgrounds
	cardBG        paletteColor   // opaque card and stats background
	maxSize       size           // max drawn frame dimensions
	alphaBlend    bool           // dim movie pixels under the card
	diffThreshold int            // threshold for ignoring tiny color changes
	statsStyle    string         // ANSI style for demo stats text
	term          size           // current terminal size

	next    timage // image being prepared for output
	prev    timage // last image believed to be on screen
	draw    rect   // terminal rectangle occupied by next
	painted rect   // terminal rectangle occupied by prev
	valid   bool   // whether prev matches terminal state
}

// newRenderer precomputes escape strings for movie palettes.
func newRenderer(movie *movie, cfg config, term size) *renderer {
	profile := cfg.colorProfile()
	palette := make([]paletteColor, len(movie.palette))
	dimPalette := make([]paletteColor, len(movie.dimPalette))
	for i, c := range movie.palette {
		palette[i] = newPaletteColor(profile, c)
	}
	for i, c := range movie.dimPalette {
		dimPalette[i] = newPaletteColor(profile, c)
	}

	renderer := &renderer{
		movie:         movie,
		palette:       palette,
		dimPalette:    dimPalette,
		cardBG:        newPaletteColor(profile, cardBackgroundColor),
		maxSize:       sz(cfg.width, cfg.height),
		alphaBlend:    cfg.alphaBlend,
		diffThreshold: cfg.diffThreshold,
		statsStyle:    foregroundEscape(profile, statsColor),
		term:          term,
	}
	renderer.draw = renderer.layoutRect()
	return renderer
}

// resize records a terminal resize and invalidates cached output.
func (r *renderer) resize(term size) bool {
	if r.term == term {
		return false
	}
	r.term = term
	r.reset()
	return true
}

// reset forces the next frame to redraw fully.
func (r *renderer) reset() {
	r.valid = false
}

// render emits a full or dirty frame into out.
func (r *renderer) render(out *bytes.Buffer, fr int, card card, stats statsTracker) {
	out.Reset()

	// draw frame
	r.draw = r.layoutRect()
	r.drawFrame(fr)
	// draw card
	r.next.overlay(card.image, center(r.next.size(), card.image.size()), r.blender(fr))
	// draw stats
	statsImage := stats.image(r.statsStyle, r.cardBG)
	r.next.overlay(statsImage, bottomRight(r.next.size(), statsImage.size()), sourceOver)

	// usually we can get away with rendering changed pixels, but sometimes we need to do a full keyframe
	keyframe := !r.valid || r.painted != r.draw || r.prev.size() != r.next.size()
	if keyframe {
		r.renderKeyFrame(out)
		r.prev.copyFrom(r.next)
		r.painted = r.draw
		r.valid = true
		return
	}

	r.renderDirty(out)
}

// renderKeyFrame writes every pixel in the draw rectangle.
func (r *renderer) renderKeyFrame(out *bytes.Buffer) {
	out.Grow(area(r.draw.Size()) * 48)
	out.WriteString(xansi.SetModeSynchronizedOutput)

	writer := newPixelWriter(out)
	for row := range r.draw.Dy() {
		out.WriteString(xansi.CursorPosition(r.draw.Min.X+1, r.draw.Min.Y+row+1))
		for _, px := range r.next.row(row) {
			writer.write(px)
		}
	}

	out.WriteString(xansi.ResetStyle)
	out.WriteString(xansi.ResetModeSynchronizedOutput)
}

// renderDirty writes only changed pixel spans.
func (r *renderer) renderDirty(out *bytes.Buffer) {
	writer := newPixelWriter(out)
	wrote := false

	for row := range r.draw.Dy() {
		for col := 0; col < r.draw.Dx(); {
			p := pt(col, row)
			if !r.pixelChanged(r.prev.at(p), r.next.at(p)) {
				col++
				continue
			}

			if !wrote {
				out.WriteString(xansi.SetModeSynchronizedOutput)
				wrote = true
			}
			out.WriteString(xansi.CursorPosition(r.draw.Min.X+col+1, r.draw.Min.Y+row+1))
			for col < r.draw.Dx() {
				p = pt(col, row)
				next := r.next.at(p)
				if !r.pixelChanged(r.prev.at(p), next) {
					break
				}
				writer.write(next)
				r.prev.set(p, next)
				col++
			}
		}
	}

	if !wrote {
		return
	}
	out.WriteString(xansi.ResetStyle)
	out.WriteString(xansi.ResetModeSynchronizedOutput)
}

// pixelChanged decides whether a pixel needs repainting.
func (r *renderer) pixelChanged(prev, next tpixel) bool {
	if prev.Ch != next.Ch || prev.Style != next.Style {
		return true
	}
	if prev.Color.Escape == next.Color.Escape {
		return false
	}
	if r.diffThreshold <= 0 {
		return true
	}
	return colorDistanceSquared(prev.Color.Color, next.Color.Color) >= r.diffThreshold*r.diffThreshold
}

// colorDistanceSquared returns a weighted RGB distance.
func colorDistanceSquared(a, b color.RGBA) int {
	dr := int(a.R) - int(b.R)
	dg := int(a.G) - int(b.G)
	db := int(a.B) - int(b.B)
	return (299*dr*dr + 587*dg*dg + 114*db*db) / 1000
}

// layoutRect fills the terminal within configured size caps.
func (r *renderer) layoutRect() rect {
	s := r.term
	if r.maxSize.X > 0 {
		s.X = min(s.X, r.maxSize.X)
	}
	if r.maxSize.Y > 0 {
		s.Y = min(s.Y, r.maxSize.Y)
	}
	if s.X < 1 {
		s.X = 1
	}
	if s.Y < 1 {
		s.Y = 1
	}

	return rectWithSize(center(r.term, s), s)
}

// pixelWriter emits pixels while suppressing redundant style escapes.
type pixelWriter struct {
	out   *bytes.Buffer
	bg    string
	style string
}

// newPixelWriter starts a terminal pixel writer for one style run.
func newPixelWriter(out *bytes.Buffer) pixelWriter {
	return pixelWriter{out: out}
}

// write writes one terminal pixel with minimal style changes.
func (writer *pixelWriter) write(px tpixel) {
	if px.Style != writer.style {
		if writer.style != "" {
			writer.out.WriteString(xansi.ResetStyle)
			writer.bg = ""
		}
		if px.Style != "" {
			writer.out.WriteString(px.Style)
			writer.bg = ""
		}
		writer.style = px.Style
	}

	if px.Color.Escape != writer.bg {
		writer.out.WriteString(px.Color.Escape)
		writer.bg = px.Color.Escape
	}

	writer.out.WriteString(px.Ch)
}
