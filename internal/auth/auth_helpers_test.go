package auth

import (
	"testing"

	"github.com/gurgeous/gshoot/internal/env"
	"github.com/gurgeous/gshoot/internal/testutil"
)

func withAuthEnv(t *testing.T, overrides map[string]string) {
	t.Helper()
	testutil.WithEnv(t, overrides, authEnvVars())
}

func authEnvVars() map[string]*string {
	return map[string]*string{
		"GOOGLE_APPLICATION_CREDENTIALS": &env.GOOGLE_APPLICATION_CREDENTIALS,
		"GSHOOT_CONFIG_DIR":              &env.GSHOOT_CONFIG_DIR,
		"GSHOOT_CREDENTIALS_FILE":        &env.GSHOOT_CREDENTIALS_FILE,
		"GSHOOT_THEME":                   &env.GSHOOT_THEME,
		"GSHOOT_TOKEN":                   &env.GSHOOT_TOKEN,
	}
}
