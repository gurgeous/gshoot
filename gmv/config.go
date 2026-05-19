package gmv

// Config reads GMV playback knobs from environment variables. It keeps invalid
// values out of the renderer.

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/colorprofile"
	"github.com/gurgeous/gshoot/env"
)

// config is the resolved playback and rendering configuration.
type config struct {
	fps           float64
	profile       colorprofile.Profile
	width         int
	height        int
	alphaBlend    bool
	diffThreshold int
}

// newConfig combines defaults, caller options, and environment overrides.
func newConfig() config {
	cfg := config{
		fps:           25,
		profile:       colorprofile.Detect(os.Stdout, os.Environ()),
		alphaBlend:    true,
		diffThreshold: 10,
	}

	if value := env.GSHOOT_GMV_FPS(); value != "" {
		cfg.fps = envFloat(value, cfg.fps)
	}
	if value := env.GSHOOT_GMV_WIDTH(); value != "" {
		cfg.width = envInt(value, cfg.width)
	}
	if value := env.GSHOOT_GMV_HEIGHT(); value != "" {
		cfg.height = envInt(value, cfg.height)
	}
	if value := env.GSHOOT_GMV_ALPHA(); value != "" {
		cfg.alphaBlend = envBool(value, cfg.alphaBlend)
	}
	if value := env.GSHOOT_GMV_DIFF_THRESHOLD(); value != "" {
		cfg.diffThreshold = envInt(value, cfg.diffThreshold)
	}

	return cfg
}

// colorProfile returns the GMV render profile for cached color escapes.
func (cfg config) colorProfile() colorprofile.Profile {
	if cfg.profile == colorprofile.TrueColor {
		return colorprofile.TrueColor
	}
	return colorprofile.ANSI256
}

// frameDelay returns the configured frame duration.
func (cfg config) frameDelay() time.Duration {
	return time.Duration(float64(time.Second) / cfg.fps)
}

// envFloat reads a positive float override or keeps fallback.
func envFloat(value string, fallback float64) float64 {
	n, err := strconv.ParseFloat(value, 64)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

// envInt reads a non-negative integer override or keeps fallback.
func envInt(value string, fallback int) int {
	n, err := strconv.Atoi(value)
	if err != nil || n < 0 {
		return fallback
	}
	return n
}

// envBool reads a boolean override or keeps fallback.
func envBool(value string, fallback bool) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "t", "yes", "y", "on":
		return true
	case "0", "false", "f", "no", "n", "off":
		return false
	default:
		return fallback
	}
}
