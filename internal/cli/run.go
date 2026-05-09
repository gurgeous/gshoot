package cli

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/gurgeous/gshoot/internal/auth"
	"github.com/gurgeous/gshoot/internal/down"
	"github.com/gurgeous/gshoot/internal/list"
	"github.com/gurgeous/gshoot/internal/ux"
	"github.com/spf13/cobra"
)

var (
	version         = "dev"
	resolveAuth     = auth.Resolve
	newTokenSource  = auth.NewTokenSource
	loginAuth       = auth.Login
	logoutAuth      = auth.Logout
	statusAuth      = auth.InspectStatus
	printAuthStatus = auth.PrintStatus
)

// Run executes the gshoot CLI.
func Run(args []string, stdout, stderr io.Writer) int {
	ux.Init()

	cmd := newRootCmd()
	cmd.SetArgs(args)
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)

	if err := cmd.Execute(); err != nil {
		writeError(stderr, err)
		return 1
	}

	return 0
}

//
// root
//

func newRootCmd() *cobra.Command {
	var showVersion bool

	cmd := &cobra.Command{
		Use:           "gshoot",
		Short:         fmt.Sprintf("Magically %s from Google Sheets.", ux.Brand.Render("import/export CSVs")),
		SilenceErrors: true,
		SilenceUsage:  true,
		CompletionOptions: cobra.CompletionOptions{
			HiddenDefaultCmd: true,
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			if showVersion {
				fmt.Fprintf(cmd.OutOrStdout(), "gshoot %s\n", version)
				return nil
			}
			writeHelp(cmd.OutOrStdout(), cmd)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&showVersion, "version", "v", false, "print version number")
	cmd.SetHelpFunc(func(command *cobra.Command, _ []string) {
		writeHelp(command.OutOrStdout(), command)
	})
	cmd.AddCommand(
		newAuthCmd(),
		newStubCmd("up", "Upload a local CSV file to a Google Sheet"),
		down.NewCommand(),
		list.NewCommand(),
	)

	return cmd
}

//
// commands
//

func newAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Login (or logout) from Google Sheets",
	}
	cmd.AddCommand(
		newAuthLoginCmd(),
		newAuthStatusCmd(),
		newAuthLogoutCmd(),
	)
	return cmd
}

func newAuthLoginCmd() *cobra.Command {
	var clientSecretPath string

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Run browser OAuth login",
		Example: strings.Join([]string{
			"gshoot auth login",
			"  gshoot auth login --client-secret ~/Downloads/client_secret.json",
		}, "\n"),
		Args: noArgs("gshoot auth login"),
		RunE: func(cmd *cobra.Command, _ []string) error {
			return loginAuth(context.Background(), auth.LoginOptions{
				ClientSecretPath: clientSecretPath,
				Stdout:           cmd.OutOrStdout(),
				Stderr:           cmd.ErrOrStderr(),
			})
		},
	}
	cmd.Flags().StringVar(&clientSecretPath, "client-secret", "", "path to a downloaded Google Desktop app OAuth client JSON")
	return cmd
}

func newAuthStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show auth status",
		Args:  noArgs("gshoot auth status"),
		RunE: func(cmd *cobra.Command, _ []string) error {
			printAuthStatus(cmd.OutOrStdout(), statusAuth())
			return nil
		},
	}
	return cmd
}

func newAuthLogoutCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Clear cached OAuth token",
		Args:  noArgs("gshoot auth logout"),
		RunE: func(cmd *cobra.Command, _ []string) error {
			removed, err := logoutAuth()
			if err != nil {
				return err
			}
			if removed {
				fmt.Fprintln(cmd.OutOrStdout(), "Removed cached OAuth token. OAuth client config was kept.")
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "No cached OAuth token was present.")
			}
			return nil
		},
	}
	return cmd
}

func newStubCmd(use, short string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Run:   func(*cobra.Command, []string) {},
	}
	return cmd
}

func noArgs(usage string) cobra.PositionalArgs {
	return func(_ *cobra.Command, args []string) error {
		if len(args) == 0 {
			return nil
		}
		return fmt.Errorf("expected `%s`", usage)
	}
}

func isRootCmd(cmd *cobra.Command) bool {
	return cmd != nil && !cmd.HasParent()
}
