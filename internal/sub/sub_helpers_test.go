package sub

import (
	"bytes"

	"github.com/gurgeous/gshoot/internal/env"
	"github.com/gurgeous/gshoot/internal/testutil"
)

func withRawTokenAuth(t testutil.TestingT) {
	t.Helper()
	testutil.WithEnv(t, map[string]string{
		"GSHOOT_TOKEN": "token",
		"HOME":         tTempDir(t),
	}, envVars())
}

func envVars() map[string]*string {
	return map[string]*string{
		"GOOGLE_APPLICATION_CREDENTIALS": &env.GOOGLE_APPLICATION_CREDENTIALS,
		"GSHOOT_CONFIG_DIR":              &env.GSHOOT_CONFIG_DIR,
		"GSHOOT_CREDENTIALS_FILE":        &env.GSHOOT_CREDENTIALS_FILE,
		"GSHOOT_THEME":                   &env.GSHOOT_THEME,
		"GSHOOT_TOKEN":                   &env.GSHOOT_TOKEN,
	}
}

type tempDirT interface {
	testutil.TestingT
	TempDir() string
}

func tTempDir(t testutil.TestingT) string {
	tt, ok := t.(tempDirT)
	if !ok {
		t.Fatalf("test helper needs TempDir")
	}
	return tt.TempDir()
}

func testMain(args ...string) (int, string, string) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Main(args, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}
