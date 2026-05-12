package commands

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/gurgeous/gshoot/util"
	"github.com/stretchr/testify/assert"
)

func TestAuthLoginCommand(t *testing.T) {
	err, _, _ := testCommandWithSetup(t, &AuthLoginCmd{}, nil, func(string) {})
	assert.Error(t, err)
	if err != nil {
		assert.Contains(t, err.Error(), "oauth-client.json")
	}
}

func TestAuthLogoutCommand(t *testing.T) {
	err, stdout, _ := testCommandWithSetup(t, &AuthLogoutCmd{}, nil, func(home string) {
		writeCommandAuthFiles(t, home)
	})

	assert.NoError(t, err)
	assert.Contains(t, stdout, "Removed cached OAuth token")
}

func TestAuthStatusCommandLoggedIn(t *testing.T) {
	err, stdout, _ := testCommandWithSetup(t, &AuthStatusCmd{}, nil, func(home string) {
		writeCommandAuthFiles(t, home)
	})

	assert.NoError(t, err)
	assert.Contains(t, stdout, "Status: logged in")
}

func TestAuthStatusCommandExpiredToken(t *testing.T) {
	err, stdout, _ := testCommandWithSetup(t, &AuthStatusCmd{}, nil, func(home string) {
		writeCommandAuthFiles(t, home, time.Now().Add(-time.Hour))
	})

	assert.NoError(t, err)
	assert.Contains(t, stdout, "Status: not logged in yet")
}

func TestAuthStatusCommandNoAuth(t *testing.T) {
	err, stdout, _ := testCommandWithSetup(t, &AuthStatusCmd{}, nil, func(string) {})

	assert.NoError(t, err)
	assert.Contains(t, stdout, "Status: no auth configured")
	assert.Contains(t, stdout, "auth login --client-secret")
}

func writeCommandAuthFiles(t *testing.T, home string, expiry ...time.Time) {
	t.Helper()

	configDir := filepath.Join(home, ".config", "gshoot")
	clientJSON := `{"installed":{"client_id":"cid","client_secret":"secret","auth_uri":"https://accounts.google.com/o/oauth2/auth","token_uri":"https://oauth2.googleapis.com/token","redirect_uris":["http://127.0.0.1/oauth2/callback"]}}`
	tokenExpiry := time.Now().Add(time.Hour)
	if len(expiry) > 0 {
		tokenExpiry = expiry[0]
	}
	tokenJSON := `{"access_token":"token","refresh_token":"refresh","token_type":"Bearer","expiry":"` + tokenExpiry.UTC().Format(time.RFC3339) + `"}`
	assert.NoError(t, util.WritePrivateFile(filepath.Join(configDir, "oauth-client.json"), []byte(clientJSON)))
	assert.NoError(t, util.WritePrivateFile(filepath.Join(configDir, "oauth-token.json"), []byte(tokenJSON)))
}
