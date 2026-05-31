package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/gurgeous/gshoot/app"
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
		fmt.Fprintln(os.Stdout, ux.Success.Render("gshoot: copied to "+manager.ClientPath))
		fmt.Fprintln(os.Stdout)
	}

	// can't proceed with login without client secrets
	if !manager.HasClientSecrets() {
		ShowAuthStatus(manager)
		return nil
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
	client.Logout()
	return nil
}

//
// status
//

func (c *AuthStatusCmd) Run() error {
	manager, err := auth.NewManager()
	if err != nil {
		return err
	}
	ShowAuthStatus(manager)
	return nil
}
