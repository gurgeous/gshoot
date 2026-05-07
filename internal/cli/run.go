package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

// Run executes the gshoot CLI.
func Run(args []string, stdout, stderr io.Writer) int {
	cmd := newRootCmd(stdout, stderr)
	cmd.SetArgs(args)

	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}

	return 0
}

func newRootCmd(stdout, stderr io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "gshoot",
		Short:         "CSV to Google Sheets workflows",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.AddCommand(
		newStubCmd("up", "Upload CSV data to Google Sheets"),
		newStubCmd("down", "Download sheet data"),
		newStubCmd("list", "List recent spreadsheets"),
	)

	return cmd
}

func newStubCmd(use, short string) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		Run:   func(*cobra.Command, []string) {},
	}
}
