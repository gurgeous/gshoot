package gmv

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCardStyleKeepsTextStyleAfterForegroundReset(t *testing.T) {
	card := newCard("\x1b[1;31mA\x1b[39mB")

	px, ok := card.pixelAt(pt(0, 0))
	assert.True(t, ok)
	assert.Equal(t, "\x1b[1;31m", px.Style)

	px, ok = card.pixelAt(pt(1, 0))
	assert.True(t, ok)
	assert.Equal(t, "\x1b[1m", px.Style)
}

func TestCardStyleDoesNotAccumulateOldForegrounds(t *testing.T) {
	card := newCard("\x1b[31mA\x1b[32mB")

	px, ok := card.pixelAt(pt(1, 0))
	assert.True(t, ok)
	assert.Equal(t, "\x1b[32m", px.Style)
}

func TestCardPreservesHorizontalPadding(t *testing.T) {
	card := newCard("  X  ")

	assert.Equal(t, 5, card.image.size().X)
	px, ok := card.pixelAt(pt(4, 0))
	assert.True(t, ok)
	assert.Equal(t, " ", px.Ch)
}

func TestCardPadsShortLines(t *testing.T) {
	card := newCard("XX\nY")

	px, ok := card.pixelAt(pt(1, 1))
	assert.True(t, ok)
	assert.Equal(t, " ", px.Ch)
}
