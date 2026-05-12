package auth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/gurgeous/gshoot/util"
)

// NewTokenSource creates an oauth2 token source from resolved auth state.
func NewTokenSource(ctx context.Context, scopes []string) (oauth2.TokenSource, error) {
	configDir := util.ConfigDir()
	oauthClientPath := filepath.Join(configDir, oauthClientFileName)
	oauthTokenPath := filepath.Join(configDir, oauthTokenFileName)

	tokenState, err := LoadOAuthToken(oauthTokenPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("gshoot [no auth found]\nchecked files: %s, %s", oauthClientPath, oauthTokenPath)
		}
		return nil, fmt.Errorf("load cached oauth token: %w", err)
	}

	token := &oauth2.Token{
		AccessToken:  tokenState.AccessToken,
		RefreshToken: tokenState.RefreshToken,
		TokenType:    tokenState.TokenType,
		Expiry:       tokenState.Expiry,
	}

	client, err := LoadOAuthClient(oauthClientPath)
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
