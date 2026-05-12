package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/gurgeous/gshoot/auth"
	"github.com/gurgeous/gshoot/ux"
)

// commands/auth.go wires the auth subcommands to Client methods.

// AuthCmd groups the auth-related subcommands.
type AuthCmd struct {
	Login  AuthLoginCmd  `cmd:"" help:"Run browser OAuth login."`
	Logout AuthLogoutCmd `cmd:"" help:"Clear the cached OAuth token."`
	Status AuthStatusCmd `cmd:"" help:"Show auth status."`
}

// AuthLoginCmd runs interactive browser login.
type AuthLoginCmd struct {
	ClientSecretPath string `name:"client-secret" type:"path" help:"Path to a Google Desktop app OAuth client JSON."`
}

// Run executes the auth login command.
func (c *AuthLoginCmd) Run() error {
	client := auth.NewClient()
	if c.ClientSecretPath != "" {
		if err := client.ImportOClient(c.ClientSecretPath); err != nil {
			return err
		}
		fmt.Println(ux.Success.Render("Imported client config to " + client.ClientPath()))
	}

	return auth.NewClient().Login(context.Background(), auth.LoginOptions{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	})
}

// AuthLogoutCmd clears the saved OAuth token.
type AuthLogoutCmd struct{}

// Run executes the auth logout command.
func (c *AuthLogoutCmd) Run() error {
	auth.NewClient().Logout()
	return nil
}

// AuthStatusCmd prints the current auth status.
type AuthStatusCmd struct{}

// Run executes the auth status command.
func (c *AuthStatusCmd) Run() error {
	return auth.NewClient().Status(os.Stdout)
}
