package gmv

// Compose is the renderer's image assembly step.
// It samples the movie into the next terminal image, then overlays card and stats images before diffing.

// drawFrame samples the movie background into the next frame.
func (r *renderer) drawFrame(fr int) {
	r.next.resize(r.draw.Size())

	origin := r.movie.frameOrigin(fr)
	for row := range r.draw.Dy() {
		for col := range r.draw.Dx() {
			p := pt(col, row)
			pal := r.movie.sample(origin, r.draw, p)
			r.next.set(p, tpixel{Ch: " ", Color: r.palette[pal]})
		}
	}
}

//
// compositing operators
//

// blender returns card compositing rules for the current movie frame.
func (r *renderer) blender(fr int) pixelBlend {
	origin := r.movie.frameOrigin(fr)
	return func(p point, _ tpixel, src tpixel) tpixel {
		color := r.cardBG
		if r.alphaBlend {
			pal := r.movie.sample(origin, r.draw, p)
			color = r.dimPalette[pal]
		}
		return tpixel{Ch: src.Ch, Color: color, Style: src.Style}
	}
}

// sourceOver copies source pixels over destination pixels.
func sourceOver(_ point, _ tpixel, src tpixel) tpixel {
	return src
}
