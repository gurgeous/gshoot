package main

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/gurgeous/gshoot/commands"
	"github.com/gurgeous/gshoot/ux"
)

//
// Main entrypoint
//

// populated by goreleaser
var commit, date, version string

// wraps command.Main and handles err
func main() {
	err := commands.Main(os.Args[1:], versionString())
	if err != nil {
		fmt.Fprintln(os.Stderr, ux.Fatal.Render(fmt.Sprintf("gshoot: %-64s", err.Error())))
		os.Exit(1)
	}
}

// pull version string, either populated by gorelease or from debug.ReadBuildInfo
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
		c += "*"
	}

	return fmt.Sprintf("gshoot %s (%s, %s)", version, c, date[:16])
}
