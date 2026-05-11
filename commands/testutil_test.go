package commands

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/gurgeous/gshoot/env"
	"github.com/gurgeous/gshoot/testutil"
)

//
// mock google apis inside TestMain
//

// tests can mess with this
var (
	googleAPIHandler http.HandlerFunc = invalid
	invalid          http.HandlerFunc = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "google api test handler not installed", http.StatusInternalServerError)
	})
)

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

func withAPI(t testutil.TestingT, handler http.HandlerFunc) {
	t.Helper()
	t.Cleanup(func() { googleAPIHandler = invalid })
	googleAPIHandler = handler
}

type roundTripper func(*http.Request) (*http.Response, error)

func (fn roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

//
// wee helper that calls Main and captures output
//

func testMain(args ...string) (int, string, string) {
	var stdout, stderr bytes.Buffer
	code := Main(args, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

//
// env mocks
//

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

//
// tempdir
//

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
