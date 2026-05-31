package gmv

import "image"

//
// Image geometry for terminal math
//

type (
	point = image.Point
	size  = image.Point
	rect  = image.Rectangle
)

// pt returns a terminal coordinate.
func pt(x, y int) point {
	return image.Pt(x, y)
}

// sz returns a terminal size.
func sz(w, h int) size {
	return image.Pt(w, h)
}

// rectWithSize returns a rectangle from an origin and size.
func rectWithSize(origin point, s size) rect {
	return rect{Min: origin, Max: origin.Add(s)}
}

// area returns the number of pixels in the size.
func area(s size) int {
	return s.X * s.Y
}

// center returns the origin needed to center inner inside outer.
func center(outer, inner size) point {
	return pt(max(0, (outer.X-inner.X)/2), max(0, (outer.Y-inner.Y)/2))
}

// bottomRight returns the origin needed to place inner at outer's bottom right.
func bottomRight(outer, inner size) point {
	return pt(max(0, outer.X-inner.X), max(0, outer.Y-inner.Y))
}
