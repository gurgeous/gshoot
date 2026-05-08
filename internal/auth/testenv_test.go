package auth

import (
	"reflect"
	"testing"

	"github.com/gurgeous/gshoot/internal/env"
)

func withTestEnv(t *testing.T, overrides map[string]string) {
	t.Helper()

	vars := map[string]*string{
		"GOOGLE_APPLICATION_CREDENTIALS": &env.GOOGLE_APPLICATION_CREDENTIALS,
		"GSHOOT_CONFIG_DIR":              &env.GSHOOT_CONFIG_DIR,
		"GSHOOT_CREDENTIALS_FILE":        &env.GSHOOT_CREDENTIALS_FILE,
		"GSHOOT_THEME":                   &env.GSHOOT_THEME,
		"GSHOOT_TOKEN":                   &env.GSHOOT_TOKEN,
		"HOME":                           &env.HOME,
		"XDG_CONFIG_HOME":                &env.XDG_CONFIG_HOME,
	}

	old := make(map[string]string, len(vars))
	for name, ptr := range vars {
		old[name] = *ptr
	}

	for name, value := range overrides {
		ptr, ok := vars[name]
		if !ok {
			t.Fatalf("unknown test env var %q", name)
		}
		reflect.ValueOf(ptr).Elem().SetString(value)
	}

	t.Cleanup(func() {
		for name, value := range old {
			reflect.ValueOf(vars[name]).Elem().SetString(value)
		}
	})
}
