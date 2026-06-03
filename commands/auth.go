package commands

import (
	"context"
	"fmt"
	"os"

	"charm.land/lipgloss/v2"
	"github.com/gurgeous/gshoot/app"
	"github.com/gurgeous/gshoot/auth"
	"github.com/gurgeous/gshoot/ux"
)

//
// Auth xxx commands
//

type (
	AuthCmd struct {
		Login  AuthLoginCmd  `cmd:"" help:"Login via OAuth. (start here!)"`
		Logout AuthLogoutCmd `cmd:"" help:"Logout of OAuth."`
		Status AuthStatusCmd `cmd:"" help:"Show auth status."`
	}

	// subcommands
	AuthLoginCmd struct {
		ClientSecretPath string `name:"client-secret" type:"path" help:"Path to a Google Desktop app OAuth client JSON."`
	}
	AuthLogoutCmd struct {
		Purge bool `help:"Delete saved client secrets too."`
	}
	AuthStatusCmd struct{}
)

//
// login
//

func (c *AuthLoginCmd) Run() error {
	manager, err := auth.NewManager()
	if err != nil {
		return err
	}

	// --client-secret
	if c.ClientSecretPath != "" {
		if err = manager.SaveOClient(c.ClientSecretPath); err != nil {
			return err
		}
		fmt.Println(ux.Success.Render("gshoot: copied to " + manager.ClientPath))
		fmt.Println()
	}

	// can't proceed with login without client secrets
	if !manager.HasClientSecrets() {
		return ShowAuthStatus()
	}

	return manager.Login(context.Background(), app.Env.Smoke, os.Stdout)
}

//
// logout
//

func (c *AuthLogoutCmd) Run() error {
	client, err := auth.NewManager()
	if err != nil {
		return err
	}
	client.Logout(c.Purge)

	msg := "gshoot: You are now " + ux.Warn.Render("logged out") + ","
	msg += " here is your updated status."
	_, _ = fmt.Println(lipgloss.Wrap(msg, 72, " "))
	fmt.Println()
	ShowAuthStatus()
	return nil
}

//
// status
//

func (c *AuthStatusCmd) Run() error {
	return ShowAuthStatus()
}
