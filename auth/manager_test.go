package auth

import (
	"os"
	"testing"
	"time"

	"github.com/gurgeous/gshoot/util"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
)

func TestSetupStateGuidesAuthWizard(t *testing.T) {
	client := withAuthHome(t)

	assert.False(t, client.HasClientSecrets())
	assert.False(t, client.LoggedIn())

	assert.NoError(t, os.MkdirAll(util.ConfigDir(), 0o700))
	client, err := NewManager()
	assert.NoError(t, err)
	assert.False(t, client.HasClientSecrets())
	assert.False(t, client.LoggedIn())

	assert.NoError(t, util.WritePrivateFile(client.ClientPath, []byte(`{"type":"service_account"}`)))
	_, err = NewManager()
	assert.Error(t, err)
	if err != nil {
		assert.Contains(t, err.Error(), "missing `installed:`")
	}

	writeClient(t, `{"installed":{"client_id":"cid","client_secret":"secret","redirect_uris":["http://127.0.0.1/oauth2/callback"]}}`)
	client, err = NewManager()
	assert.NoError(t, err)
	assert.True(t, client.HasClientSecrets())
	assert.False(t, client.LoggedIn())

	writeAuthToken(t, oauth2.Token{
		AccessToken:  "expired",
		RefreshToken: "refresh",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(-time.Hour),
	})
	client, err = NewManager()
	assert.NoError(t, err)
	assert.True(t, client.HasClientSecrets())
	assert.True(t, client.LoggedIn())

	writeAuthToken(t, futureToken())
	client, err = NewManager()
	assert.NoError(t, err)
	assert.True(t, client.HasClientSecrets())
	assert.True(t, client.LoggedIn())
}

func TestSetupStateRejectsBrokenTokenFiles(t *testing.T) {
	client := withAuthHome(t)
	writeClient(t, `{"installed":{"client_id":"cid","client_secret":"secret","redirect_uris":["http://127.0.0.1/oauth2/callback"]}}`)

	for _, tt := range []struct {
		body string
		want string
	}{
		{body: `{"refresh_token":"refresh","expiry":"2026-05-07T22:00:00Z"}`, want: "missing `access_token`"},
		{body: `{"access_token":"access","expiry":"2026-05-07T22:00:00Z"}`, want: "missing `refresh_token`"},
		{body: `{"access_token":"access","refresh_token":"refresh"}`, want: "missing `expiry`"},
	} {
		assert.NoError(t, util.WritePrivateFile(client.TokenPath, []byte(tt.body)))

		_, err := NewManager()

		assert.Error(t, err)
		if err != nil {
			assert.Contains(t, err.Error(), tt.want)
		}
	}
}
