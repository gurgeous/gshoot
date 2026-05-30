package auth

import (
	"context"
	"fmt"
	"os"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/gurgeous/gshoot/env"
	"github.com/gurgeous/gshoot/util"
	"github.com/gurgeous/gshoot/ux"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

//
// Login
//

var openBrowser = util.OpenBrowserURL

func (m *Manager) Login(ctx context.Context) error {
	//
	// browser login flow
	//

	// send the user off to google.com, get an oauth token using our client secret
	token, err := browserLoginFlow(ctx, m.client)
	if err != nil {
		return err
	}

	//
	// success!
	//

	if err := m.SaveOAuthToken(token); err != nil {
		return err
	}
	fmt.Println()
	fmt.Println(ux.Success.Render("gshoot: success! oauth token copied to " + m.TokenPath))
	fmt.Println("gshoot should work now, have fun!")

	return nil
}

//
// logout (delete token but leave client secrets)
//

func (m *Manager) Logout() {
	os.Remove(m.TokenPath)
	m.token = nil
}

// browserLoginFlow performs the browser round trip and code exchange.
func browserLoginFlow(ctx context.Context, client *OClient) (*oauth2.Token, error) {
	if env.NewConfig().Smoke {
		return &oauth2.Token{
			AccessToken:  "smoke-access-token",
			RefreshToken: "smoke-refresh-token",
			TokenType:    "Bearer",
			Expiry:       time.Now().Add(time.Hour),
		}, nil
	}

	//
	// create oauth2 config
	//

	config := oauth2.Config{
		ClientID:     client.ClientID,
		ClientSecret: client.ClientSecret,
		Endpoint:     google.Endpoint,
		RedirectURL:  client.LocalhostRedirect.String(),
		Scopes:       Scopes,
	}

	//
	// now start our loopback server and get the auth code url (which includes the loopback url)
	//

	loopback := NewLoopback(client.LocalhostRedirect)
	if err := loopback.Start(); err != nil {
		return nil, err
	}
	config.RedirectURL = loopback.RedirectURL
	authURL := config.AuthCodeURL(loopback.State, oauth2.AccessTypeOffline, oauth2.SetAuthURLParam("prompt", "consent"))

	//
	// tell the user what to do
	//

	intro := "Now you will need to click through the OAuth thing at Google. I will open this magic Google URL in your browser. If I can't open your browser, you can click or copy/paste to open it manually. Here is the URL:"
	fmt.Println(lipgloss.Wrap(intro, 72, " "))
	fmt.Println()
	fmt.Println(ux.Success.Render(authURL))
	fmt.Println(ux.Muted.Render("(only works if you can run a browser, see README for headless tips)"))
	fmt.Println()
	fmt.Println(ux.Brand.Render("gshoot is now waiting for you to finish OAuth so we can continue..."))
	openBrowser(authURL)

	//
	// now we wait for someone to hit our loopback url
	//

	code, err := loopback.Wait(ctx)
	if err != nil {
		return nil, err
	}

	//
	// exchange the callback code for an OAuth token
	//

	token, err := config.Exchange(ctx, code)
	if err != nil {
		return nil, err
	}

	//
	// success!
	//

	return token, nil
}
