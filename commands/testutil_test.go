package commands

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
)

//
// TestMain
//

// tests can mess with this
var (
	googleAPIHandler http.HandlerFunc = invalid
	invalid          http.HandlerFunc = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "google api test handler not installed", http.StatusInternalServerError)
	})
)

type roundTripper func(*http.Request) (*http.Response, error)

func (fn roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestMain(m *testing.M) {
	// create a fake server that points at googleAPIHandler
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		googleAPIHandler.ServeHTTP(w, r)
	}))
	defer server.Close()

	orig := http.DefaultTransport
	defer func() { http.DefaultTransport = orig }()
	http.DefaultTransport = roundTripper(func(req *http.Request) (*http.Response, error) {
		if !strings.Contains(req.URL.Host, "googleapis.com") {
			return orig.RoundTrip(req)
		}
		target, _ := url.Parse(server.URL)
		cloned := req.Clone(req.Context())
		cloned.URL.Scheme = target.Scheme
		cloned.URL.Host = target.Host
		cloned.Host = target.Host
		return orig.RoundTrip(cloned)
	})

	os.Exit(m.Run())
}

//
// test a kong command (run in tmp dir, capture stdout, etc)
//

// kong commands look like this
type runnable interface {
	Run() error
}

func testCommand(t *testing.T, cmd runnable, handler http.HandlerFunc) (error, string, string) {
	t.Helper()

	// use temp dir and temp files for stdout/stderr
	origDir, _ := os.Getwd()
	tmp := t.TempDir()
	os.Chdir(tmp)
	t.Cleanup(func() { os.Chdir(origDir) })

	// stub stdout/stderr
	origStdout, origStderr := os.Stdout, os.Stderr
	stdoutFile, _ := os.Create("test-stdout")
	stderrFile, _ := os.Create("test-stderr")
	os.Stdout, os.Stderr = stdoutFile, stderrFile
	t.Cleanup(func() { os.Stdout, os.Stderr = origStdout, origStderr })
	t.Cleanup(func() { stdoutFile.Close() })
	t.Cleanup(func() { stderrFile.Close() })

	// fake env
	t.Setenv("GSHOOT_TOKEN", "bogus_token")
	t.Setenv("HOME", tmp)

	// stub google api
	googleAPIHandler = handler
	defer func() { googleAPIHandler = invalid }()

	// run
	runErr := cmd.Run()

	//

	// drain stdout/stderr
	stdoutFile.Seek(0, 0)
	stderrFile.Seek(0, 0)
	stdoutBytes, _ := io.ReadAll(stdoutFile)
	stderrBytes, _ := io.ReadAll(stderrFile)

	return runErr, string(stdoutBytes), string(stderrBytes)
}

// //
// // env mocks
// //

// func withRawTokenAuth(t *testing.T) {
// 	t.Helper()
// 	testutil.WithEnv(t, map[string]string{
// 		"GSHOOT_TOKEN": "token",
// 		"HOME":         tTempDir(t),
// 	}, envVars())
// }

// func envVars() map[string]*string {
// 	return map[string]*string{
// 		"GOOGLE_APPLICATION_CREDENTIALS": &env.GOOGLE_APPLICATION_CREDENTIALS,
// 		"GSHOOT_CONFIG_DIR":              &env.GSHOOT_CONFIG_DIR,
// 		"GSHOOT_CREDENTIALS_FILE":        &env.GSHOOT_CREDENTIALS_FILE,
// 		"GSHOOT_THEME":                   &env.GSHOOT_THEME,
// 		"GSHOOT_TOKEN":                   &env.GSHOOT_TOKEN,
// 	}
// }
