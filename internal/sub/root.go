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
	tagline = fmt.Sprintf("Magically %s from Google Sheets.", ux.Brand.Render("import/export CSVs"))
	rootCmd = &cobra.Command{
		Use:           "gshoot",
		Short:         tagline,
		SilenceErrors: true,
		SilenceUsage:  true,
		CompletionOptions: cobra.CompletionOptions{
			HiddenDefaultCmd: true,
		},
		RunE: RootHandler,
	}

	// args
	showVersion = false
)

func init() {
	rootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "print version number")
	rootCmd.SetHelpFunc(func(command *cobra.Command, _ []string) { WriteHelp(command) })
}

func RootHandler(cmd *cobra.Command, _ []string) error {
	if showVersion {
		fmt.Fprintf(cmd.OutOrStdout(), "gshoot %s\n", version)
	} else {
		WriteHelp(cmd)
	}
	return nil
}

//
// main
//

func Main(args []string, stdout, stderr io.Writer) int {
	ux.Init()

	rootCmd.SetArgs(args)
	rootCmd.SetErr(stderr)
	rootCmd.SetOut(stdout)
	resetCommand(rootCmd)

	if err := rootCmd.Execute(); err != nil {
		writeError(stderr, err)
		return 1
	}

	return 0
}

//
// helpers
//

func noArgs(usage string) cobra.PositionalArgs {
	return func(_ *cobra.Command, args []string) error {
		if len(args) == 0 {
			return nil
		}
		return fmt.Errorf("expected `%s`", usage)
	}
}

func resetCommand(cmd *cobra.Command) {
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		_ = flag.Value.Set(flag.DefValue)
		flag.Changed = false
	})
	cmd.PersistentFlags().VisitAll(func(flag *pflag.Flag) {
		_ = flag.Value.Set(flag.DefValue)
		flag.Changed = false
	})
	for _, sub := range cmd.Commands() {
		resetCommand(sub)
	}
}
