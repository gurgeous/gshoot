package smoke

import (
	"flag"
	"fmt"
	"io"
)

// Run executes the manual smoke-test entrypoint.
func Run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("gshoot-smoke", flag.ContinueOnError)
	fs.SetOutput(stderr)

	gshootPath := fs.String("gshoot", "", "path to the gshoot binary")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	if *gshootPath == "" {
		fmt.Fprintln(stderr, "missing required --gshoot path")
		return 1
	}

	fmt.Fprintf(stdout, "gsmoke stub: using %s\n", *gshootPath)
	return 0
}
