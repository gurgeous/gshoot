package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strings"

	"github.com/gurgeous/gshoot/commands"
	"github.com/gurgeous/gshoot/gog"
	"github.com/gurgeous/gshoot/ux"
)

//
// Main entrypoint
//

// goreleaser populates these by default, let's use 'em
var commit, date, version string

// wraps command.Main and handles err
func main() {
	err := commands.Main(os.Args[1:], versionString())
	if err != nil {
		reportError(os.Stderr, err)
		os.Exit(1)
	}
}

func reportError(w io.Writer, err error) {
	if _, ok := errors.AsType[*gog.AuthError](err); !ok {
		fmt.Fprintln(w, fatalText(err))
		return
	}

	const friendly = `
Uh oh, it looks like gog auth isn't working. Try "gog drive ls". Once you have
that working, come back and try gshoot again.
`

	fmt.Fprintln(w, ux.Warn.Render(err.Error()))
	fmt.Fprintln(w)
	fmt.Fprintln(w, ux.Warn.Render(strings.TrimSpace(friendly)))
	fmt.Fprintln(w)
	fmt.Fprintln(w, fatalText(errors.New("gog authentication failed")))
}

func fatalText(err error) string {
	lines := strings.Split(err.Error(), "\n")
	for i, line := range lines {
		prefix := "        "
		if i == 0 {
			prefix = "gshoot: "
		}
		lines[i] = ux.Fatal.Render(fmt.Sprintf("%s%-64s", prefix, line))
	}
	return strings.Join(lines, "\n")
}

// calculate version string, from goreleaser or debug.ReadBuildInfo
func versionString() string {
	modified := false

	if version == "" {
		version = "built from source"
		if info, ok := debug.ReadBuildInfo(); ok {
			for _, setting := range info.Settings {
				switch setting.Key {
				case "vcs.revision":
					commit = setting.Value
				case "vcs.time":
					date = setting.Value
				case "vcs.modified":
					if setting.Value == "true" {
						modified = true
					}
				}
			}
		}
	}

	c := commit[:7]
	if modified {
		// only possible in dev
		c += "*"
	}

	return fmt.Sprintf("gshoot %s (%s, %s)", version, c, date[:16])
}
