package app

import (
	cenv "github.com/caarlos0/env/v11"

	"github.com/gurgeous/gshoot/ux"
)

// Env contains process-wide environment config.
var Env Config

// Config contains gshoot environment variables.
type Config struct {
	Smoke bool   `env:"GSHOOT_SMOKE"` // use deterministic smoke-test behavior
	Theme string `env:"GSHOOT_THEME"` // force light or dark UI theme
}

// Init initializes process-wide app state.
func Init() {
	var err error
	Env, err = cenv.ParseAs[Config]()
	if err != nil {
		Env = Config{}
	}
	ux.Init(Env.Theme)
}
