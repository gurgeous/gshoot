package commands

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gurgeous/gshoot/app"
	"github.com/gurgeous/gshoot/auth"
	"github.com/gurgeous/gshoot/util"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
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
	Run(*app.App) error
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
	origDir, err := os.Getwd()
	assert.NoError(t, err)
	tmp := t.TempDir()
	assert.NoError(t, os.Chdir(tmp))
	t.Cleanup(func() { assert.NoError(t, os.Chdir(origDir)) })

	// capture stdout/stderr
	stdoutFile, err := os.Create("test-stdout")
	assert.NoError(t, err)
	stderrFile, err := os.Create("test-stderr")
	assert.NoError(t, err)
	t.Cleanup(func() { stdoutFile.Close() })
	t.Cleanup(func() { stderrFile.Close() })

	// fake browser auth under HOME
	t.Setenv("HOME", tmp)
	if len(setup) > 0 && setup[0] != nil {
		setup[0](tmp)
	} else {
		writeAuthFiles(t, tmp)
	}
	origStdout, origStderr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = stdoutFile, stderrFile
	a := app.New()
	t.Cleanup(func() {
		os.Stdout, os.Stderr = origStdout, origStderr
	})

	// stub google api
	googleAPIHandler = handler
	defer func() { googleAPIHandler = invalid }()

	// run
	runErr := cmd.Run(a)

	//

	// drain stdout/stderr
	_, err = stdoutFile.Seek(0, 0)
	assert.NoError(t, err)
	_, err = stderrFile.Seek(0, 0)
	assert.NoError(t, err)
	stdoutBytes, err := io.ReadAll(stdoutFile)
	assert.NoError(t, err)
	stderrBytes, err := io.ReadAll(stderrFile)
	assert.NoError(t, err)

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

	manager, err := auth.NewManager()
	if err != nil {
		t.Fatalf("load auth manager: %v", err)
	}
	if cfg.HasClient {
		clientJSON := `{"installed":{"client_id":"cid","client_secret":"secret","redirect_uris":["http://127.0.0.1/oauth2/callback"]}}`
		if err := util.WritePrivateFile(manager.ClientPath, []byte(clientJSON)); err != nil {
			t.Fatalf("write oauth-client.json: %v", err)
		}
	}
	if cfg.HasToken {
		if err := manager.SaveOAuthToken(&oauth2.Token{
			AccessToken:  "token",
			RefreshToken: "refresh",
			TokenType:    "Bearer",
			Expiry:       cfg.Expiry,
		}); err != nil {
			t.Fatalf("write oauth token: %v", err)
		}
	}
}
