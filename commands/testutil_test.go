package commands

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/adrg/xdg"
	"github.com/gurgeous/gshoot/util"
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

type authFilesOptions struct {
	HasClient bool
	HasToken  bool
	Expiry    time.Time
}

func testCommand(t *testing.T, cmd runnable, handler http.HandlerFunc) (error, string, string) {
	return testCommandWithSetup(t, cmd, handler)
}

func testCommandWithSetup(t *testing.T, cmd runnable, handler http.HandlerFunc, setup ...func(string)) (error, string, string) {
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

	// fake browser auth under HOME
	t.Cleanup(xdg.Reload)
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, ".config"))
	xdg.Reload()
	if len(setup) > 0 && setup[0] != nil {
		setup[0](tmp)
	} else {
		writeAuthFiles(t, tmp)
	}

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

func writeAuthFiles(t *testing.T, _ string, opts ...authFilesOptions) {
	t.Helper()

	cfg := authFilesOptions{
		HasClient: true,
		HasToken:  true,
		Expiry:    time.Now().Add(time.Hour),
	}
	if len(opts) > 0 {
		cfg = opts[0]
		if cfg.Expiry.IsZero() {
			cfg.Expiry = time.Now().Add(time.Hour)
		}
	}

	configDir := util.ConfigDir()
	if cfg.HasClient {
		clientJSON := `{"installed":{"client_id":"cid","client_secret":"secret","auth_uri":"https://accounts.google.com/o/oauth2/auth","token_uri":"https://oauth2.googleapis.com/token","redirect_uris":["http://127.0.0.1/oauth2/callback"]}}`
		if err := util.WritePrivateFile(filepath.Join(configDir, "oauth-client.json"), []byte(clientJSON)); err != nil {
			t.Fatalf("write oauth-client.json: %v", err)
		}
	}
	if cfg.HasToken {
		tokenJSON := fmt.Sprintf(
			`{"access_token":"token","refresh_token":"refresh","token_type":"Bearer","expiry":"%s"}`,
			cfg.Expiry.UTC().Format(time.RFC3339),
		)
		if err := util.WritePrivateFile(filepath.Join(configDir, "oauth-token.json"), []byte(tokenJSON)); err != nil {
			t.Fatalf("write oauth-token.json: %v", err)
		}
	}
}
