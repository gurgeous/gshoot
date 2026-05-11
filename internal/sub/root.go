package sub

import (
	"fmt"
	"io"

	"github.com/gurgeous/gshoot/internal/ux"
	"github.com/spf13/cobra"
)

var (
	version = "dev"
	tagline = fmt.Sprintf("Magically %s from Google Sheets.", ux.Brand.Render("import/export CSVs"))
	rootCmd = &cobra.Command{
		RunE:          RootHandler,
		Use:           "gshoot",
		Short:         tagline,
		SilenceErrors: true,
		SilenceUsage:  true,
		CompletionOptions: cobra.CompletionOptions{
			HiddenDefaultCmd: true,
		},
	}
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

	rootCmd.SetArgs(args)
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(stderr)

	if err := rootCmd.Execute(); err != nil {
		writeError(stderr, err)
		return 1
	}

	return 0
}
