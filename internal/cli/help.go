package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/gurgeous/gshoot/internal/util"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func writeHelp(w io.Writer, cmd *cobra.Command) {
	if isRootCmd(cmd) {
		writeRootHelp(w, cmd)
	} else {
		writeCommandHelp(w, cmd)
	}
}

func writeRootHelp(w io.Writer, cmd *cobra.Command) {
	fmt.Fprintf(w, "Usage: %s <command> [flags]\n", cmd.Name())

	if text := commandSummary(cmd); text != "" {
		fmt.Fprintln(w)
		fmt.Fprintln(w, text)
	}

	flags := visibleFlags(cmd.LocalFlags())
	commands := availableCommands(cmd)
	padding := rootHelpPadding(cmd, flags, commands)

	if len(flags) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Flags:")
		writeFlags(w, flags, padding)
	}

	if len(commands) == 0 {
		return
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	for _, sub := range commands {
		fmt.Fprintf(w, "  %s%s\n", util.RPad(sub.Name(), padding+2), sub.Short)
	}

	fmt.Fprintln(w)
	fmt.Fprintf(w, "Run %q for more information on a command.\n", "gshoot <command> --help")
}

func writeCommandHelp(w io.Writer, cmd *cobra.Command) {
	if text := commandSummary(cmd); text != "" {
		fmt.Fprintln(w, text)
		fmt.Fprintln(w)
	}
	fmt.Fprintf(w, "USAGE\n  %s\n", cmd.UseLine())

	if commands := availableCommands(cmd); len(commands) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "COMMANDS")
		for _, sub := range commands {
			fmt.Fprintf(w, "  %s%s\n", util.RPad(sub.Name(), cmd.NamePadding()), sub.Short)
		}
	}

	if flags := visibleFlags(cmd.LocalFlags()); len(flags) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "FLAGS")
		writeFlags(w, flags, cmd.NamePadding())
	}

	if flags := visibleFlags(cmd.InheritedFlags()); len(flags) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "INHERITED FLAGS")
		writeFlags(w, flags, cmd.NamePadding())
	}

	if example := strings.TrimSpace(cmd.Example); example != "" {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "EXAMPLES")
		fmt.Fprintln(w, util.IndentBlock(example))
	}
}

func commandSummary(cmd *cobra.Command) string {
	if cmd.Long != "" {
		return strings.TrimSpace(cmd.Long)
	}
	return strings.TrimSpace(cmd.Short)
}

func availableCommands(cmd *cobra.Command) []*cobra.Command {
	var commands []*cobra.Command
	for _, sub := range cmd.Commands() {
		if !sub.IsAvailableCommand() || sub.IsAdditionalHelpTopicCommand() {
			continue
		}
		commands = append(commands, sub)
	}
	return commands
}

func rootHelpPadding(cmd *cobra.Command, flags []helpFlag, commands []*cobra.Command) int {
	padding := cmd.NamePadding()
	for _, flag := range flags {
		padding = max(len(flag.name), padding)
	}
	for _, sub := range commands {
		padding = max(len(sub.Name()), padding)
	}
	return padding
}

type helpFlag struct {
	name string
	help string
}

func visibleFlags(flags *pflag.FlagSet) []helpFlag {
	if flags == nil {
		return nil
	}
	var out []helpFlag
	flags.VisitAll(func(flag *pflag.Flag) {
		if flag.Hidden || flag.Deprecated != "" {
			return
		}
		name := "--" + flag.Name
		if flag.Shorthand != "" && flag.ShorthandDeprecated == "" {
			name = "-" + flag.Shorthand + ", " + name
		}
		if flag.Value.Type() != "bool" {
			name += " " + flag.Value.Type()
		}
		out = append(out, helpFlag{name: name, help: flag.Usage})
	})
	return out
}

func writeFlags(w io.Writer, flags []helpFlag, minPadding int) {
	padding := minPadding
	for _, flag := range flags {
		padding = max(len(flag.name), padding)
	}
	for _, flag := range flags {
		fmt.Fprintf(w, "  %s%s\n", util.RPad(flag.name, padding+2), flag.help)
	}
}
