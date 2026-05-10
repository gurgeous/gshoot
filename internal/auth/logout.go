package auth

import (
	"fmt"
	"os"
	"path/filepath"
)

// Logout clears the cached OAuth session while keeping the client config.
func Logout() (bool, error) {
	tokenPath := filepath.Join(ConfigDir(), oauthTokenFileName)
	err := os.Remove(tokenPath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, fmt.Errorf("remove cached oauth token: %w", err)
}
