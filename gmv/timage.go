package gmv

// Timage is GMV's terminal image primitive.
// It is a rectangular grid of one-column pixels; empty Ch means transparent, while " " is an opaque blank.

// tpixel is one terminal cell in an image.
type tpixel struct {
	Ch    string
	Color paletteColor
	Style string
}

// timage is a two-dimensional terminal image.
type timage struct {
	pixels [][]tpixel
}

// resize makes the image rectangular with the requested dimensions.
func (img *timage) resize(s size) {
	if img.size() == s {
		return
	}
	img.pixels = make([][]tpixel, s.Y)
	for row := range img.pixels {
		img.pixels[row] = make([]tpixel, s.X)
	}
}

// size returns the image dimensions.
func (img timage) size() size {
	if len(img.pixels) == 0 {
		return sz(0, 0)
	}
	return sz(len(img.pixels[0]), len(img.pixels))
}

// contains reports whether p is inside the image.
func (img timage) contains(p point) bool {
	s := img.size()
	return p.X >= 0 && p.Y >= 0 && p.X < s.X && p.Y < s.Y
}

// at returns the pixel at p.
func (img timage) at(p point) tpixel {
	return img.pixels[p.Y][p.X]
}

// set writes px at p.
func (img *timage) set(p point, px tpixel) {
	img.pixels[p.Y][p.X] = px
}

// row returns one image row.
func (img timage) row(y int) []tpixel {
	return img.pixels[y]
}

// copyFrom copies src into img.
func (img *timage) copyFrom(src timage) {
	img.resize(src.size())
	for y, row := range src.pixels {
		copy(img.pixels[y], row)
	}
}

//
// blending
//

// pixelBlend composites a source pixel onto a destination pixel.
type pixelBlend func(point, tpixel, tpixel) tpixel

// overlay composites non-transparent source pixels onto img.
func (img *timage) overlay(src timage, origin point, blend pixelBlend) {
	for y, row := range src.pixels {
		for x, srcPx := range row {
			if srcPx.Ch == "" {
				continue
			}

			p := origin.Add(pt(x, y))
			if !img.contains(p) {
				continue
			}

			img.set(p, blend(p, img.at(p), srcPx))
		}
	}
}
