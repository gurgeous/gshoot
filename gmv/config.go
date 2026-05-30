package gmv

// Config is the small environment surface for GMV playback.
// It resolves fps, color profile, draw caps, alpha blending, and diff tolerance before rendering starts.

import (
	"os"
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
	envCfg := env.NewConfig()
	cfg := config{
		fps:           25,
		profile:       colorprofile.Detect(os.Stdout, os.Environ()),
		alphaBlend:    true,
		diffThreshold: 10,
	}

	if envCfg.GMVFPS > 0 {
		cfg.fps = envCfg.GMVFPS
	}
	if envCfg.GMVWidth > 0 {
		cfg.width = envCfg.GMVWidth
	}
	if envCfg.GMVHeight > 0 {
		cfg.height = envCfg.GMVHeight
	}
	if envCfg.GMVNoAlpha {
		cfg.alphaBlend = false
	}
	if envCfg.GMVDiffThreshold > 0 {
		cfg.diffThreshold = envCfg.GMVDiffThreshold
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
