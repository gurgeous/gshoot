package googletest

import (
	"net/http"
	"net/url"

	"github.com/gurgeous/gshoot/internal/testutil"
)

// WithGoogleAPI rewrites outgoing Google API requests to serverURL for a test.
func WithGoogleAPI(t testutil.TestingT, serverURL string) {
	t.Helper()

	target, err := url.Parse(serverURL)
	if err != nil {
		t.Fatalf("Parse(serverURL) error = %v", err)
	}

	orig := http.DefaultTransport
	http.DefaultTransport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		cloned := req.Clone(req.Context())
		cloned.URL.Scheme = target.Scheme
		cloned.URL.Host = target.Host
		cloned.Host = target.Host
		return orig.RoundTrip(cloned)
	})
	t.Cleanup(func() {
		http.DefaultTransport = orig
	})
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
