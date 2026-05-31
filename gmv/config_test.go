package gmv

import (
	"io"
	"testing"

	"github.com/gurgeous/gshoot/env"
	"github.com/stretchr/testify/assert"
)

func TestNewConfigReadsEnv(t *testing.T) {
	t.Setenv("GSHOOT_GMV_FPS", "12.5")
	t.Setenv("GSHOOT_GMV_WIDTH", "80")
	t.Setenv("GSHOOT_GMV_HEIGHT", "24")
	t.Setenv("GSHOOT_GMV_NO_ALPHA", "true")
	t.Setenv("GSHOOT_GMV_DIFF_THRESHOLD", "4")

	cfg := newConfig(env.NewConfig(), io.Discard)

	assert.Equal(t, 12.5, cfg.fps)
	assert.Equal(t, 80, cfg.width)
	assert.Equal(t, 24, cfg.height)
	assert.False(t, cfg.alphaBlend)
	assert.Equal(t, 4, cfg.diffThreshold)
}

func TestNewConfigIgnoresInvalidEnv(t *testing.T) {
	t.Setenv("GSHOOT_GMV_FPS", "wat")
	t.Setenv("GSHOOT_GMV_WIDTH", "80")

	cfg := newConfig(env.NewConfig(), io.Discard)

	assert.Equal(t, 25.0, cfg.fps)
	assert.Equal(t, 0, cfg.width)
	assert.Equal(t, 0, cfg.height)
	assert.True(t, cfg.alphaBlend)
	assert.Equal(t, 10, cfg.diffThreshold)
}
