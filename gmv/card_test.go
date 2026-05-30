package gmv

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCardStyleKeepsTextStyleAfterForegroundReset(t *testing.T) {
	card := newCard("\x1b[1;31mA\x1b[39mB")

	px := card.image.at(pt(0, 0))
	assert.Equal(t, "\x1b[1;31m", px.Style)

	px = card.image.at(pt(1, 0))
	assert.Equal(t, "\x1b[1m", px.Style)
}

func TestCardStyleDoesNotAccumulateOldForegrounds(t *testing.T) {
	card := newCard("\x1b[31mA\x1b[32mB")

	px := card.image.at(pt(1, 0))
	assert.Equal(t, "\x1b[32m", px.Style)
}

func TestCardStyleIgnoresUnsupportedStyles(t *testing.T) {
	card := newCard("\x1b[3;4;48;2;1;2;3;31mA")

	px := card.image.at(pt(0, 0))
	assert.Equal(t, "\x1b[31m", px.Style)
}

func TestCardPreservesHorizontalPadding(t *testing.T) {
	card := newCard("  X  ")

	assert.Equal(t, 5, card.image.size().X)
	px := card.image.at(pt(4, 0))
	assert.Equal(t, " ", px.Ch)
}

func TestCardPadsShortLines(t *testing.T) {
	card := newCard("XX\nY")

	px := card.image.at(pt(1, 1))
	assert.Equal(t, " ", px.Ch)
}
