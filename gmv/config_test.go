package gmv

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewConfigReadsAlphaEnv(t *testing.T) {
	t.Setenv("GSHOOT_GMV_ALPHA", "off")

	cfg := newConfig()

	assert.False(t, cfg.alphaBlend)
}

func TestNewConfigIgnoresInvalidEnv(t *testing.T) {
	t.Setenv("GSHOOT_GMV_FPS", "wat")
	t.Setenv("GSHOOT_GMV_WIDTH", "-1")
	t.Setenv("GSHOOT_GMV_HEIGHT", "-1")
	t.Setenv("GSHOOT_GMV_ALPHA", "maybe")
	t.Setenv("GSHOOT_GMV_DIFF_THRESHOLD", "-1")

	cfg := newConfig()

	assert.Equal(t, 25.0, cfg.fps)
	assert.Equal(t, 0, cfg.width)
	assert.Equal(t, 0, cfg.height)
	assert.True(t, cfg.alphaBlend)
	assert.Equal(t, 10, cfg.diffThreshold)
}
