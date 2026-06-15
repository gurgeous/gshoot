package auth

import (
	"errors"
	"strings"

	"golang.org/x/oauth2"
)

//
// Helpers for turning OAuth failures into user-facing auth guidance.
//

var ErrLoginExpired = errors.New("Your Google login has expired. You'll need to log in again.\nhint: run `gshoot auth login` to log in again")

// IsInvalidGrant reports whether err came from a revoked or expired refresh token.
func IsInvalidGrant(err error) bool {
	var retrieveErr *oauth2.RetrieveError
	if errors.As(err, &retrieveErr) {
		return retrieveErr.ErrorCode == "invalid_grant"
	}

	return strings.Contains(err.Error(), `oauth2: "invalid_grant"`)
}
