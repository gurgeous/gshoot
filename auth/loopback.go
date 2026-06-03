package auth

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/gurgeous/gshoot/util"
)

//
// Temporary localhost OAuth callback server
//

const (
	oauthReadHeaderTimeout = 5 * time.Second
	oauthShutdownTimeout   = 5 * time.Second
)

type Loopback struct {
	RedirectURL string // redirect URL sent to Google after binding an ephemeral port
	CallbackURL string // local callback URL shown to the user
	State       string // expected OAuth `state` (random hex)

	// internal
	redirect *url.URL     // configured loopback redirect URL from the OAuth client
	server   *http.Server // temporary HTTP server accepting the callback
	codeCh   chan string  // successful authorization code from the callback
	errCh    chan error   // callback or server error
}

// NewLoopback builds a loopback callback server for OAuth login.
func NewLoopback(redirect *url.URL) *Loopback {
	return &Loopback{
		redirect: redirect,
		State:    util.RandomHex(16),
	}
}

// Start binds the local callback server and records the redirect URLs.
func (l *Loopback) Start() error {
	redirectURL := *l.redirect
	redirectURL.Scheme = "http"

	host := redirectURL.Hostname()
	if host == "" {
		host = "127.0.0.1"
	}

	listener, err := net.Listen("tcp", net.JoinHostPort(host, "0"))
	if err != nil {
		return fmt.Errorf("listen for OAuth callback: %w", err)
	}

	// Replace the configured loopback port with the ephemeral port we actually bound.
	redirectURL.Host = listener.Addr().String()
	callbackURL := redirectURL.String()
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)
	mux := http.NewServeMux()
	server := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: oauthReadHeaderTimeout,
	}
	l.RedirectURL = callbackURL
	l.CallbackURL = callbackURL
	l.server = server
	l.codeCh = codeCh
	l.errCh = errCh

	mux.HandleFunc(redirectURL.Path, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if got := q.Get("state"); got != l.State {
			http.Error(w, "state mismatch", http.StatusBadRequest)
			l.errCh <- errors.New("OAuth callback state mismatch")
			return
		}
		if gotErr := q.Get("error"); gotErr != "" {
			http.Error(w, "login failed", http.StatusBadRequest)
			l.errCh <- fmt.Errorf("OAuth callback error: %s", gotErr)
			return
		}
		code := q.Get("code")
		if code == "" {
			http.Error(w, "missing code", http.StatusBadRequest)
			l.errCh <- errors.New("OAuth callback missing code")
			return
		}
		fmt.Fprintln(w, "gshoot: login complete, you can close this tab.") // HTTP response
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		l.codeCh <- code
	})

	go func() {
		if serveErr := server.Serve(listener); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			l.errCh <- serveErr
		}
	}()

	return nil
}

// Wait returns the first successful code, callback error, or caller cancellation.
func (l *Loopback) Wait(ctx context.Context) (string, error) {
	defer l.shutdown()
	select {
	case code := <-l.codeCh:
		return code, nil
	case err := <-l.errCh:
		return "", err
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

func (l *Loopback) shutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), oauthShutdownTimeout)
	defer cancel()
	_ = l.server.Shutdown(ctx)
}
