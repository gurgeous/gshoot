package commands

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAuthLogin(t *testing.T) {
	err, _, _ := testCommandWithSetup(t, &AuthLoginCmd{}, nil, func(string) {})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "oauth-client.json")
}

func TestAuthLogout(t *testing.T) {
	err, stdout, _ := testCommandWithSetup(t, &AuthLogoutCmd{}, nil, func(home string) {
		writeAuthFiles(t, home)
	})
	assert.NoError(t, err)
	assert.Contains(t, stdout, "Removed cached OAuth token")
}

func TestAuthStatus(t *testing.T) {
	err, stdout, _ := testCommandWithSetup(t, &AuthStatusCmd{}, nil, func(home string) {
		writeAuthFiles(t, home)
	})
	assert.NoError(t, err)
	assert.Contains(t, stdout, "Status: logged in")

	// expired
	err, stdout, _ = testCommandWithSetup(t, &AuthStatusCmd{}, nil, func(home string) {
		writeAuthFiles(t, home, authFilesOptions{HasClient: true, HasToken: true, Expiry: time.Now().Add(-time.Hour)})
	})
	assert.NoError(t, err)
	assert.Contains(t, stdout, "Status: not logged in yet")

	// no auth
	err, stdout, _ = testCommandWithSetup(t, &AuthStatusCmd{}, nil, func(string) {})
	assert.NoError(t, err)
	assert.Contains(t, stdout, "Status: no auth configured")
	assert.Contains(t, stdout, "auth login --client-secret")
}
