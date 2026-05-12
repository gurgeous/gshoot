package auth

import (
	"context"
	"errors"
	"fmt"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// auth/token_source.go turns saved browser auth files into token sources.

// TokenSource creates an oauth2 token source from saved auth state.
func (c *Manager) TokenSource(ctx context.Context, scopes []string) (oauth2.TokenSource, error) {
	tokenState, err := c.LoadOAuthToken()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("gshoot [no auth found]\nchecked files: %s, %s", c.ClientPath(), c.TokenPath())
		}
		return nil, fmt.Errorf("load cached oauth token: %w", err)
	}

	token := &oauth2.Token{
		AccessToken:  tokenState.AccessToken,
		RefreshToken: tokenState.RefreshToken,
		TokenType:    tokenState.TokenType,
		Expiry:       tokenState.Expiry,
	}

	client, err := c.LoadOClient()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if !token.Valid() {
				return nil, errors.New("cached oauth token is expired and oauth-client.json is missing")
			}
			return oauth2.StaticTokenSource(token), nil
		}
		return nil, fmt.Errorf("load oauth client config: %w", err)
	}
	config := &oauth2.Config{
		ClientID:     client.ClientID,
		ClientSecret: client.ClientSecret,
		Endpoint:     oauthEndpoint(client.AuthURI, client.TokenURI),
		Scopes:       scopes,
		RedirectURL: func() string {
			if len(client.RedirectURIs) == 0 {
				return ""
			}
			return client.RedirectURIs[0]
		}(),
	}
	return config.TokenSource(ctx, token), nil
}

// oauthEndpoint applies optional endpoint overrides on top of Google's defaults.
func oauthEndpoint(authURL, tokenURL string) oauth2.Endpoint {
	endpoint := google.Endpoint
	if authURL != "" {
		endpoint.AuthURL = authURL
	}
	if tokenURL != "" {
		endpoint.TokenURL = tokenURL
	}
	return endpoint
}
