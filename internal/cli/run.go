package cli

import (
	"fmt"
	"io"

	"github.com/gurgeous/gshoot/internal/auth"
	"github.com/gurgeous/gshoot/internal/down"
	"github.com/gurgeous/gshoot/internal/list"
	"github.com/gurgeous/gshoot/internal/ux"
	"github.com/spf13/cobra"
)

var (
	version        = "dev"
	resolveAuth    = auth.Resolve
	newTokenSource = auth.NewTokenSource
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
		auth.NewCommand(),
		newStubCmd("up", "Upload a local CSV file to a Google Sheet"),
		down.NewCommand(),
		list.NewListCommand(),
	)

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
