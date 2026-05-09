package auth

import (
	"os"
	"reflect"
	"testing"

	"github.com/adrg/xdg"
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
	}

	old := make(map[string]string, len(vars))
	oldSet := make(map[string]bool, len(vars))
	for name, ptr := range vars {
		old[name] = *ptr
		_, oldSet[name] = os.LookupEnv(name)
	}

	for name, value := range overrides {
		ptr, ok := vars[name]
		if !ok {
			t.Fatalf("unknown test env var %q", name)
		}
		reflect.ValueOf(ptr).Elem().SetString(value)
		if err := os.Setenv(name, value); err != nil {
			t.Fatalf("setenv %s: %v", name, err)
		}
	}
	xdg.Reload()

	t.Cleanup(func() {
		for name, value := range old {
			reflect.ValueOf(vars[name]).Elem().SetString(value)
			if oldSet[name] {
				if err := os.Setenv(name, value); err != nil {
					t.Fatalf("restore env %s: %v", name, err)
				}
				continue
			}
			if err := os.Unsetenv(name); err != nil {
				t.Fatalf("unset env %s: %v", name, err)
			}
		}
		xdg.Reload()
	})
}
