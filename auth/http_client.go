package auth

import (
	"context"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

//
// An authenticated OAuth HTTP client to talk to Google. Wraps oauth2.NewClient,
// saves fresh oauth tokens to disk on refesh
//

func (m *Manager) HTTPClient(ctx context.Context) (*http.Client, error) {
	if m.token == nil || m.client == nil {
		panic("no token or client?")
	}

	config := &oauth2.Config{
		ClientID:     m.client.ClientID,
		ClientSecret: m.client.ClientSecret,
		Endpoint:     google.Endpoint,
		Scopes:       Scopes,
		RedirectURL:  m.client.LocalhostRedirect.String(),
	}

	// get token source, then wrap it with our version that saves refreshes
	tokenSource := config.TokenSource(ctx, m.token)
	tokenSource = &saveTokenSource{
		manager:  m,
		src:      tokenSource,
		previous: m.token,
	}

	// now create the http client with our oauth2 stuff inside
	return oauth2.NewClient(ctx, tokenSource), nil
}

//
// special TokenSource wrapper, it saves updated OAuthToken to disk on refresh
//

type saveTokenSource struct {
	manager  *Manager
	src      oauth2.TokenSource
	previous *oauth2.Token
}

func (s *saveTokenSource) Token() (*oauth2.Token, error) {
	// note: this can trigger a refresh
	nxt, err := s.src.Token()
	if err != nil {
		return nil, err
	}

	// use existing refresh token, google doesn't necessarily send a new one
	if nxt.RefreshToken == "" {
		nxt.RefreshToken = s.previous.RefreshToken
	}

	// save if necessary
	if nxt.AccessToken != s.previous.AccessToken || !nxt.Expiry.Equal(s.previous.Expiry) {
		if err := s.manager.SaveOAuthToken(nxt); err != nil {
			return nil, err
		}
	}
	s.previous = nxt

	return nxt, nil
}
