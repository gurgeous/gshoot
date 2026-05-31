package commands

import (
	"context"

	"github.com/gurgeous/gshoot/auth"
	"github.com/gurgeous/gshoot/ux"
)

//
// commands
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
	AuthLogoutCmd struct{}
	AuthStatusCmd struct{}
)

//
// login
//

func (c *AuthLoginCmd) Run(a *App) error {
	manager, err := auth.NewManager()
	if err != nil {
		return err
	}

	// --client-secret
	if c.ClientSecretPath != "" {
		if err = manager.SaveOClient(c.ClientSecretPath); err != nil {
			return err
		}
		a.Println(ux.Success.Render("gshoot: copied to " + manager.ClientPath))
		a.Println()
	}

	// can't proceed with login without client secrets
	if !manager.HasClientSecrets() {
		manager.ShowStatus()
		return nil
	}

	return manager.Login(context.Background(), a.Smoke)
}

//
// logout
//

func (c *AuthLogoutCmd) Run(*App) error {
	client, err := auth.NewManager()
	if err != nil {
		return err
	}
	client.Logout()
	return nil
}

//
// status
//

func (c *AuthStatusCmd) Run(_ *App) error {
	manager, err := auth.NewManager()
	if err != nil {
		return err
	}
	manager.ShowStatus()
	return nil
}
