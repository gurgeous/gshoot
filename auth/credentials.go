package auth

// auth/credentials.go defines the saved OAuth client JSON shape.

// OAuthClient is an installed/web OAuth client config.
type OAuthClient struct {
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	AuthURI      string   `json:"auth_uri"`
	TokenURI     string   `json:"token_uri"`
	RedirectURIs []string `json:"redirect_uris"`
}
