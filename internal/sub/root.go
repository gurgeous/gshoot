package sub

import (
	"fmt"
	"io"

	"github.com/gurgeous/gshoot/internal/ux"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	version = "dev"
	rootCmd = &cobra.Command{
		Use:           "gshoot",
		Short:         fmt.Sprintf("Magically %s from Google Sheets.", ux.Brand.Render("import/export CSVs")),
		SilenceErrors: true,
		SilenceUsage:  true,
		CompletionOptions: cobra.CompletionOptions{
			HiddenDefaultCmd: true,
		},
	}
)

func init() {
	var showVersion bool

	rootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "print version number")
	rootCmd.SetHelpFunc(func(command *cobra.Command, _ []string) {
		writeHelp(command.OutOrStdout(), command)
	})
	rootCmd.RunE = func(cmd *cobra.Command, _ []string) error {
		if showVersion {
			fmt.Fprintf(cmd.OutOrStdout(), "gshoot %s\n", version)
			return nil
		}
		writeHelp(cmd.OutOrStdout(), cmd)
		return nil
	}
	rootCmd.AddCommand(newStubCmd("up", "Upload a local CSV file to a Google Sheet"))
}

// Run executes the gshoot CLI.
func Run(args []string, stdout, stderr io.Writer) int {
	ux.Init()

	resetCommand(rootCmd)
	rootCmd.SetArgs(args)
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(stderr)

	if err := rootCmd.Execute(); err != nil {
		writeError(stderr, err)
		return 1
	}

	return 0
}

func newStubCmd(use, short string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Run:   func(*cobra.Command, []string) {},
	}
	return cmd
}

func isRootCmd(cmd *cobra.Command) bool {
	return cmd != nil && !cmd.HasParent()
}

func noArgs(usage string) cobra.PositionalArgs {
	return func(_ *cobra.Command, args []string) error {
		if len(args) == 0 {
			return nil
		}
		return fmt.Errorf("expected `%s`", usage)
	}
}

func resetCommand(cmd *cobra.Command) {
	resetFlagSet(cmd.Flags())
	resetFlagSet(cmd.PersistentFlags())
	for _, sub := range cmd.Commands() {
		resetCommand(sub)
	}
}

func resetFlagSet(flags *pflag.FlagSet) {
	if flags == nil {
		return
	}
	flags.VisitAll(func(flag *pflag.Flag) {
		_ = flag.Value.Set(flag.DefValue)
		flag.Changed = false
	})
}
