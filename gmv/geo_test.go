package gmv

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPointAddSub(t *testing.T) {
	p := pt(5, 7)
	q := pt(2, 3)

	assert.Equal(t, pt(7, 10), p.Add(q))
	assert.Equal(t, pt(3, 4), p.Sub(q))
}

func TestSizeArea(t *testing.T) {
	assert.Equal(t, 12, area(sz(3, 4)))
}

func TestSizeCenter(t *testing.T) {
	assert.Equal(t, pt(4, 3), center(sz(10, 8), sz(2, 2)))
	assert.Equal(t, pt(0, 0), center(sz(2, 2), sz(10, 8)))
}

func TestSizeBottomRight(t *testing.T) {
	assert.Equal(t, pt(8, 6), bottomRight(sz(10, 8), sz(2, 2)))
	assert.Equal(t, pt(0, 0), bottomRight(sz(2, 2), sz(10, 8)))
}

func TestRectLocal(t *testing.T) {
	rect := rectWithSize(pt(10, 20), sz(8, 6))

	assert.Equal(t, pt(2, 3), local(rect, pt(12, 23)))
}
