package main

import (
	"fmt"
	"os"
	"runtime/debug"
	"strings"

	"github.com/gurgeous/gshoot/commands"
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
		fmt.Fprintln(os.Stderr, fatalText(err))
		os.Exit(1)
	}
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
