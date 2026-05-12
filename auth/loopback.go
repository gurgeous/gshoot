package auth

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"
)

// auth/loopback.go runs the temporary localhost callback server for browser login.

const oauthReadHeaderTimeout = 5 * time.Second

// startLoopback starts a one-shot localhost callback server for OAuth login.
func startLoopback(redirectRaw, state string) (string, string, func(context.Context) (string, error), error) {
	redirectURL, err := url.Parse(redirectRaw)
	if err != nil {
		return "", "", nil, fmt.Errorf("parse redirect url: %w", err)
	}

	host := redirectURL.Hostname()
	if host == "" {
		host = "127.0.0.1"
	}

	listener, err := net.Listen("tcp", net.JoinHostPort(host, "0"))
	if err != nil {
		return "", "", nil, fmt.Errorf("listen for oauth callback: %w", err)
	}

	// Replace the configured loopback port with the ephemeral port we actually bound.
	redirectURL.Host = listener.Addr().String()
	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)
	server := &http.Server{ReadHeaderTimeout: oauthReadHeaderTimeout}
	mux := http.NewServeMux()
	mux.HandleFunc(redirectURL.Path, func(w http.ResponseWriter, r *http.Request) {
		// Shutdown must happen after the response is written, and not on this handler's stack.
		defer func() { go func() { _ = server.Shutdown(context.Background()) }() }()
		q := r.URL.Query()
		if got := q.Get("state"); got != state {
			http.Error(w, "state mismatch", http.StatusBadRequest)
			errCh <- errors.New("oauth callback state mismatch")
			return
		}
		if gotErr := q.Get("error"); gotErr != "" {
			http.Error(w, "login failed", http.StatusBadRequest)
			errCh <- fmt.Errorf("oauth callback error: %s", gotErr)
			return
		}
		code := q.Get("code")
		if code == "" {
			http.Error(w, "missing code", http.StatusBadRequest)
			errCh <- errors.New("oauth callback missing code")
			return
		}
		fmt.Fprintln(w, "gshoot login complete. You can close this tab.")
		codeCh <- code
	})
	server.Handler = mux

	go func() {
		if serveErr := server.Serve(listener); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			errCh <- serveErr
		}
	}()

	// Wait for the first successful code, callback error, or caller cancellation.
	wait := func(ctx context.Context) (string, error) {
		defer listener.Close()
		select {
		case code := <-codeCh:
			return code, nil
		case err := <-errCh:
			return "", err
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}

	return redirectURL.String(), "http://" + listener.Addr().String() + redirectURL.Path, wait, nil
}
