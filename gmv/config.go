package gmv

import (
	"os"
	"time"

	"github.com/charmbracelet/colorprofile"
)

//
// MV playback settings
//

type config struct {
	fps           float64
	profile       colorprofile.Profile
	width         int
	height        int
	alphaBlend    bool
	diffThreshold int
}

// newConfig returns default playback settings.
func newConfig() config {
	return config{
		fps:           25,
		profile:       colorprofile.Detect(os.Stdout, os.Environ()),
		alphaBlend:    true,
		diffThreshold: 10,
	}
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
