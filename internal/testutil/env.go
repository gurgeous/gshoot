package testutil

import (
	"os"

	"github.com/adrg/xdg"
)

type TestingT interface {
	Cleanup(func())
	Fatalf(string, ...any)
	Helper()
}

// WithEnv applies process env overrides and mirrored env package vars for a test.
func WithEnv(t TestingT, overrides map[string]string, vars map[string]*string) {
	t.Helper()

	names := make(map[string]struct{}, len(vars)+len(overrides))
	for name := range vars {
		names[name] = struct{}{}
	}
	for name := range overrides {
		names[name] = struct{}{}
	}

	oldVars := make(map[string]string, len(vars))
	for name, ptr := range vars {
		oldVars[name] = *ptr
		*ptr = ""
	}

	oldEnv := make(map[string]string, len(names))
	oldSet := make(map[string]bool, len(names))
	for name := range names {
		oldEnv[name], oldSet[name] = os.LookupEnv(name)
		if err := os.Unsetenv(name); err != nil {
			t.Fatalf("unset env %s: %v", name, err)
		}
	}

	for name, value := range overrides {
		if ptr, ok := vars[name]; ok {
			*ptr = value
		}
		if err := os.Setenv(name, value); err != nil {
			t.Fatalf("set env %s: %v", name, err)
		}
	}
	xdg.Reload()

	t.Cleanup(func() {
		for name, value := range oldVars {
			*vars[name] = value
		}
		for name := range names {
			if oldSet[name] {
				if err := os.Setenv(name, oldEnv[name]); err != nil {
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
