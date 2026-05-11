package auth

import (
	"context"
	"errors"
	"fmt"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
)

// NewTokenSource creates an oauth2 token source from resolved auth state.
func NewTokenSource(ctx context.Context, resolved Resolved, scopes []string) (oauth2.TokenSource, error) {
	switch resolved.Source.Kind {
	case SourceKindRawToken:
		return oauth2.StaticTokenSource(&oauth2.Token{
			AccessToken: resolved.Source.Token,
			TokenType:   "Bearer",
		}), nil
	case SourceKindCredentialsFile, SourceKindApplicationDefaultCredentials:
		if resolved.Source.CredentialFile == nil {
			return nil, errors.New("missing parsed credential file")
		}
		return tokenSourceFromCredentialFile(ctx, resolved.Source.CredentialFile, scopes)
	case SourceKindCachedOAuth:
		if resolved.Source.OAuthToken == nil {
			return nil, errors.New("missing parsed oauth token")
		}

		token := &oauth2.Token{
			AccessToken:  resolved.Source.OAuthToken.AccessToken,
			RefreshToken: resolved.Source.OAuthToken.RefreshToken,
			TokenType:    resolved.Source.OAuthToken.TokenType,
			Expiry:       resolved.Source.OAuthToken.Expiry,
		}

		client, err := LoadCredentialFile(resolved.OAuthClientPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				if !token.Valid() {
					return nil, errors.New("cached oauth token is expired and oauth-client.json is missing")
				}
				return oauth2.StaticTokenSource(token), nil
			}
			return nil, fmt.Errorf("load oauth client config: %w", err)
		}
		if client.Kind != CredentialKindOAuthClient || client.OAuthClient == nil {
			return nil, errors.New("oauth client config is not an oauth client")
		}

		config := &oauth2.Config{
			ClientID:     client.OAuthClient.ClientID,
			ClientSecret: client.OAuthClient.ClientSecret,
			Endpoint:     oauthEndpoint(client.OAuthClient.AuthURI, client.OAuthClient.TokenURI),
			Scopes:       scopes,
			RedirectURL:  firstRedirectURI(client.OAuthClient.RedirectURIs),
		}
		return config.TokenSource(ctx, token), nil
	default:
		return nil, fmt.Errorf("unsupported auth source: %q", resolved.Source.Kind)
	}
}

func tokenSourceFromCredentialFile(ctx context.Context, cred *CredentialFile, scopes []string) (oauth2.TokenSource, error) {
	switch cred.Kind {
	case CredentialKindAuthorizedUser:
		if cred.AuthorizedUser == nil {
			return nil, errors.New("missing authorized user credentials")
		}
		config := &oauth2.Config{
			ClientID:     cred.AuthorizedUser.ClientID,
			ClientSecret: cred.AuthorizedUser.ClientSecret,
			Endpoint:     oauthEndpoint("", cred.AuthorizedUser.TokenURI),
			Scopes:       scopes,
		}
		token := &oauth2.Token{
			RefreshToken: cred.AuthorizedUser.RefreshToken,
		}
		return config.TokenSource(ctx, token), nil
	case CredentialKindServiceAccount:
		if cred.ServiceAccount == nil {
			return nil, errors.New("missing service account credentials")
		}
		config := &jwt.Config{
			Email:      cred.ServiceAccount.ClientEmail,
			PrivateKey: []byte(cred.ServiceAccount.PrivateKey),
			TokenURL:   google.JWTTokenURL,
			Scopes:     scopes,
		}
		if cred.ServiceAccount.TokenURI != "" {
			config.TokenURL = cred.ServiceAccount.TokenURI
		}
		return config.TokenSource(ctx), nil
	case CredentialKindOAuthClient:
		return nil, errors.New("oauth client config requires browser login")
	default:
		return nil, fmt.Errorf("unsupported credential kind: %q", cred.Kind)
	}
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

func firstRedirectURI(redirectURIs []string) string {
	if len(redirectURIs) == 0 {
		return ""
	}
	return redirectURIs[0]
}
