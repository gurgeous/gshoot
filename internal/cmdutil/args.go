package cmdutil

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NoArgs rejects positional args and reports the expected usage.
func NoArgs(usage string) cobra.PositionalArgs {
	return func(_ *cobra.Command, args []string) error {
		if len(args) == 0 {
			return nil
		}
		return fmt.Errorf("expected `%s`", usage)
	}
}
