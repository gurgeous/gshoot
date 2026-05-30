package env

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewConfigReadsEnv(t *testing.T) {
	t.Setenv("GSHOOT_THEME", "light")
	t.Setenv("GSHOOT_GMV_FPS", "12.5")
	t.Setenv("GSHOOT_GMV_WIDTH", "80")
	t.Setenv("GSHOOT_GMV_HEIGHT", "24")
	t.Setenv("GSHOOT_GMV_NO_ALPHA", "true")
	t.Setenv("GSHOOT_GMV_DIFF_THRESHOLD", "4")

	cfg := NewConfig()

	assert.Equal(t, "light", cfg.Theme)
	assert.Equal(t, 12.5, cfg.GMVFPS)
	assert.Equal(t, 80, cfg.GMVWidth)
	assert.Equal(t, 24, cfg.GMVHeight)
	assert.True(t, cfg.GMVNoAlpha)
	assert.Equal(t, 4, cfg.GMVDiffThreshold)
}

func TestNewConfigReturnsZeroConfigOnInvalidEnv(t *testing.T) {
	t.Setenv("GSHOOT_THEME", "light")
	t.Setenv("GSHOOT_GMV_FPS", "wat")

	assert.Equal(t, Config{}, NewConfig())
}
