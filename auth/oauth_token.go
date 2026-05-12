package auth

import "time"

// auth/oauth_token.go defines the saved OAuth token JSON shape.

// OAuthToken is cached OAuth token state.
type OAuthToken struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	Expiry       time.Time `json:"expiry"`
}
