package env

import cenv "github.com/caarlos0/env/v11"

// Config contains environment variables used by gshoot.
// Charm also looks at many env vars, like NO_COLOR, TERM, etc. Those are not listed here.
type Config struct {
	Theme string `env:"GSHOOT_THEME"` // force light or dark UI theme

	// these are for messing around with the movie player
	GMVFPS           float64 `env:"GSHOOT_GMV_FPS"`            // override movie playback FPS
	GMVWidth         int     `env:"GSHOOT_GMV_WIDTH"`          // cap movie render width
	GMVHeight        int     `env:"GSHOOT_GMV_HEIGHT"`         // cap movie render height
	GMVNoAlpha       bool    `env:"GSHOOT_GMV_NO_ALPHA"`       // disable alpha blending under cards
	GMVDiffThreshold int     `env:"GSHOOT_GMV_DIFF_THRESHOLD"` // tune frame diff tolerance
}

// NewConfig reads gshoot's environment config.
func NewConfig() Config {
	cfg, err := cenv.ParseAs[Config]()
	if err != nil {
		return Config{}
	}
	return cfg
}
