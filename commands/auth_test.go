package commands

import (
	"testing"
	"time"

	"github.com/gurgeous/gshoot/auth"
	"github.com/stretchr/testify/assert"
)

func TestAuthLogin(t *testing.T) {
	err, stdout, _ := testCommandWithSetup(t, &AuthLoginCmd{}, nil, func(string) {})
	assert.NoError(t, err)
	assert.Contains(t, stdout, "auth status")
	assert.Contains(t, stdout, "client secrets")
}

func TestAuthLogout(t *testing.T) {
	err, _, _ := testCommandWithSetup(t, &AuthLogoutCmd{}, nil, func(home string) {
		writeAuthFiles(t, home)
	})
	assert.NoError(t, err)
}

func TestAuthStatus(t *testing.T) {
	err, stdout, _ := testCommandWithSetup(t, &AuthStatusCmd{}, nil, func(home string) {
		writeAuthFiles(t, home)
	})
	assert.NoError(t, err)
	assert.Contains(t, stdout, "Client secrets file:")
	assert.Contains(t, stdout, "Token file:")

	// expired
	err, stdout, _ = testCommandWithSetup(t, &AuthStatusCmd{}, nil, func(home string) {
		writeAuthFiles(t, home, authFilesOptions{HasClient: true, HasToken: true, Expiry: time.Now().Add(-time.Hour)})
	})
	assert.NoError(t, err)
	assert.Contains(t, stdout, "Client secrets file:")
	assert.Contains(t, stdout, "Token file:")

	// no auth
	err, stdout, _ = testCommandWithSetup(t, &AuthStatusCmd{}, nil, func(string) {})
	assert.NoError(t, err)
	assert.Contains(t, stdout, "auth status")
	assert.Contains(t, stdout, "client secrets")
	assert.Contains(t, stdout, "Github README")
	assert.Contains(t, auth.AuthReadmeURL, "github.com/gurgeous/gshoot#authentication")
	assert.NotContains(t, stdout, "<b>")
	assert.NotContains(t, stdout, "</b>")
}
