package commands

import (
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/gurgeous/gshoot/app"
	"github.com/stretchr/testify/assert"
)

type runnable interface {
	Run() error
}

func testCommand(t *testing.T, command runnable, responses ...string) (error, string, string, string) {
	t.Helper()
	origDir, err := os.Getwd()
	assert.NoError(t, err)
	tmp := t.TempDir()
	assert.NoError(t, os.Chdir(tmp))
	t.Cleanup(func() { assert.NoError(t, os.Chdir(origDir)) })

	stdout, err := os.Create(filepath.Join(tmp, "stdout"))
	assert.NoError(t, err)
	stderr, err := os.Create(filepath.Join(tmp, "stderr"))
	assert.NoError(t, err)
	origStdout, origStderr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = stdout, stderr
	t.Cleanup(func() {
		os.Stdout, os.Stderr = origStdout, origStderr
		_ = stdout.Close()
		_ = stderr.Close()
	})

	bin := filepath.Join(tmp, "bin")
	assert.NoError(t, os.Mkdir(bin, 0o700))
	script := `#!/bin/sh
dir="$FAKE_GOG_DIR"
n=0
test -f "$dir/count" && n=$(cat "$dir/count")
n=$((n + 1))
echo "$n" > "$dir/count"
printf '%s\n' "$*" >> "$dir/log"
cat > "$dir/stdin.$n"
test -f "$dir/response.$n" && cat "$dir/response.$n"
`
	assert.NoError(t, os.WriteFile(filepath.Join(bin, "gog"), []byte(script), 0o700))
	assert.NoError(t, os.WriteFile(filepath.Join(tmp, "log"), nil, 0o600))
	for i, response := range responses {
		assert.NoError(t, os.WriteFile(filepath.Join(tmp, "response."+strconv.Itoa(i+1)), []byte(response), 0o600))
	}
	t.Setenv("FAKE_GOG_DIR", tmp)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	app.Init()

	runErr := command.Run()
	assert.NoError(t, stdout.Sync())
	assert.NoError(t, stderr.Sync())
	return runErr, readFile(t, stdout.Name()), readFile(t, stderr.Name()), readFile(t, filepath.Join(tmp, "log"))
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	file, err := os.Open(path)
	assert.NoError(t, err)
	defer file.Close()
	data, err := io.ReadAll(file)
	assert.NoError(t, err)
	return strings.TrimSpace(string(data))
}

func writeCSV(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "input.csv")
	assert.NoError(t, os.WriteFile(path, []byte(body), 0o600))
	return path
}
