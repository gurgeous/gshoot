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
	rootCmd.SetHelpFunc(func(command *cobra.Command, _ []string) {
		writeHelp(command.OutOrStdout(), command)
	})
}

func RootHandler(cmd *cobra.Command, _ []string) error {
	if showVersion {
		fmt.Fprintf(cmd.OutOrStdout(), "gshoot %s\n", version)
		return nil
	}
	writeHelp(cmd.OutOrStdout(), cmd)
	return nil
}

//
// main
//

func Main(args []string, stdout, stderr io.Writer) int {
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
