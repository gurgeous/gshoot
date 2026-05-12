package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/gurgeous/gshoot/auth"
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
	return auth.NewClient().Login(context.Background(), auth.LoginOptions{
		ClientSecretPath: c.ClientSecretPath,
		Stdout:           os.Stdout,
		Stderr:           os.Stderr,
	})
}

// AuthLogoutCmd clears the saved OAuth token.
type AuthLogoutCmd struct{}

// Run executes the auth logout command.
func (c *AuthLogoutCmd) Run() error {
	removed, err := auth.NewClient().Logout()
	if err != nil {
		return err
	}
	if removed {
		fmt.Fprintln(os.Stdout, "Removed cached OAuth token. OAuth client config was kept.")
		return nil
	}
	fmt.Fprintln(os.Stdout, "No cached OAuth token was present.")
	return nil
}

// AuthStatusCmd prints the current auth status.
type AuthStatusCmd struct{}

// Run executes the auth status command.
func (c *AuthStatusCmd) Run() error {
	return auth.NewClient().Status(os.Stdout)
}
