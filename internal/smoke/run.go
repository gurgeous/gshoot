package smoke

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var executablePath = os.Executable

// Run executes the manual smoke-test entrypoint.
func Run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("gshoot-smoke", flag.ContinueOnError)
	fs.SetOutput(stderr)

	gshootPath := fs.String("gshoot", "", "path to the gshoot binary")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	gshootCommand := fs.Args()
	if len(gshootCommand) == 0 && *gshootPath != "" {
		gshootCommand = []string{*gshootPath}
	}
	if len(gshootCommand) == 0 {
		if inferred, err := inferGshootCommand(); err == nil {
			gshootCommand = inferred
		}
	}

	if len(gshootCommand) == 0 {
		fmt.Fprintln(stderr, "missing gshoot command")
		return 1
	}

	fmt.Fprintf(stdout, "gsmoke stub: using %s\n", strings.Join(gshootCommand, " "))
	return 0
}

func inferGshootCommand() ([]string, error) {
	exePath, err := executablePath()
	if err != nil {
		return nil, err
	}

	gshootPath := filepath.Join(filepath.Dir(exePath), "gshoot")
	if _, err := os.Stat(gshootPath); err != nil {
		return nil, err
	}

	return []string{gshootPath}, nil
}
