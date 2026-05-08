package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/gurgeous/gshoot/internal/auth"
	"github.com/gurgeous/gshoot/internal/down"
	"github.com/gurgeous/gshoot/internal/listing"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"golang.org/x/oauth2"
)

var (
	resolveAuth     = auth.Resolve
	newTokenSource  = auth.NewTokenSource
	loginAuth       = auth.Login
	logoutAuth      = auth.Logout
	statusAuth      = auth.InspectStatus
	printAuthStatus = auth.PrintStatus
	newDownClient   = func(ctx context.Context, tokenSource oauth2.TokenSource) (down.Client, error) {
		return down.NewGoogleClient(ctx, tokenSource)
	}
	newListingClient = func(ctx context.Context, tokenSource oauth2.TokenSource) (listing.Client, error) {
		return listing.NewGoogleClient(ctx, tokenSource)
	}
)

// Run executes the gshoot CLI.
func Run(args []string, stdout, stderr io.Writer) int {
	cmd := newRootCmd(stdout, stderr)
	cmd.SetArgs(args)

	if err := cmd.Execute(); err != nil {
		writeError(stderr, err)
		return 1
	}

	return 0
}

func newRootCmd(stdout, stderr io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "gshoot",
		Short:         "Magically import/export CSVs from Google Sheets.",
		Example:       "",
		SilenceUsage:  true,
		SilenceErrors: true,
		CompletionOptions: cobra.CompletionOptions{
			HiddenDefaultCmd: true,
		},
	}

	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetHelpFunc(func(command *cobra.Command, _ []string) {
		writeHelp(stdout, command)
	})
	cmd.AddCommand(
		newAuthCmd(stdout, stderr),
		newStubCmd("up", "Upload a local CSV file to a Google Sheet"),
		newDownCmd(stdout, stderr),
		newListCmd(stdout, stderr),
	)

	return cmd
}

func newAuthCmd(stdout, stderr io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Login (or logout) from Google Sheets",
	}
	cmd.AddCommand(
		newAuthLoginCmd(stdout, stderr),
		newAuthStatusCmd(stdout, stderr),
		newAuthLogoutCmd(stdout, stderr),
	)
	return cmd
}

func newAuthLoginCmd(stdout, stderr io.Writer) *cobra.Command {
	var clientSecretPath string

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Run browser OAuth login",
		Example: strings.Join([]string{
			"gshoot auth login",
			"  gshoot auth login --client-secret ~/Downloads/client_secret.json",
		}, "\n"),
		Args:  noArgs("gshoot auth login"),
		RunE: func(_ *cobra.Command, _ []string) error {
			return loginAuth(context.Background(), auth.LoginOptions{
				Env:              auth.NewEnv(nil),
				ClientSecretPath: clientSecretPath,
				Stdout:           stdout,
				Stderr:           stderr,
			})
		},
	}
	cmd.Flags().StringVar(&clientSecretPath, "client-secret", "", "path to a downloaded Google Desktop app OAuth client JSON")
	return cmd
}

func newAuthStatusCmd(stdout, _ io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show auth status",
		Args:  noArgs("gshoot auth status"),
		RunE: func(_ *cobra.Command, _ []string) error {
			printAuthStatus(stdout, statusAuth(auth.NewEnv(nil)))
			return nil
		},
	}
	return cmd
}

func newAuthLogoutCmd(stdout, _ io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Clear cached OAuth token",
		Args:  noArgs("gshoot auth logout"),
		RunE: func(_ *cobra.Command, _ []string) error {
			removed, err := logoutAuth(auth.NewEnv(nil))
			if err != nil {
				return err
			}
			if removed {
				fmt.Fprintln(stdout, "Removed cached OAuth token. OAuth client config was kept.")
			} else {
				fmt.Fprintln(stdout, "No cached OAuth token was present.")
			}
			return nil
		},
	}
	return cmd
}

func newDownCmd(stdout, stderr io.Writer) *cobra.Command {
	var outputPath string

	cmd := &cobra.Command{
		Use:   "down <spreadsheet> [sheet]",
		Short: "Download a Google Sheet as CSV",
		Example: strings.Join([]string{
			"gshoot down Budget",
			"  gshoot down Budget Q1 --output q1.csv",
		}, "\n"),
		Args:  downArgs,
		RunE: func(_ *cobra.Command, args []string) error {
			ctx := context.Background()
			resolved, err := resolveAuth(auth.Options{
				Env:     auth.NewEnv(nil),
				Command: auth.CommandDown,
			})
			if err != nil {
				return err
			}

			tokenSource, err := newTokenSource(ctx, resolved)
			if err != nil {
				return err
			}

			client, err := newDownClient(ctx, tokenSource)
			if err != nil {
				return err
			}

			sheetName := ""
			if len(args) == 2 {
				sheetName = args[1]
			}

			result, err := down.NewService(client).Download(ctx, args[0], sheetName)
			if err != nil {
				var spreadsheetErr *down.SpreadsheetNotFoundError
				if errors.As(err, &spreadsheetErr) {
					return spreadsheetNotFoundError(spreadsheetErr.Name)
				}

				var sheetErr *down.SheetNotFoundError
				if errors.As(err, &sheetErr) {
					return sheetNotFoundError(sheetErr.Spreadsheet, sheetErr.Sheet)
				}

				var noSheetsErr *down.NoSheetsError
				if errors.As(err, &noSheetsErr) {
					return noSheetsError(noSheetsErr.Spreadsheet)
				}
				return err
			}

			writer := stdout
			if outputPath != "" {
				file, err := os.Create(outputPath)
				if err != nil {
					return fmt.Errorf("create output file: %w", err)
				}
				defer file.Close()
				writer = file
			}

			return down.WriteCSV(writer, result.Values)
		},
	}
	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "where to write the CSV")
	return cmd
}

func newStubCmd(use, short string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Run:   func(*cobra.Command, []string) {},
	}
	return cmd
}

func noArgs(usage string) cobra.PositionalArgs {
	return func(_ *cobra.Command, args []string) error {
		if len(args) == 0 {
			return nil
		}
		return fmt.Errorf("expected `%s`", usage)
	}
}

func downArgs(_ *cobra.Command, args []string) error {
	if len(args) >= 1 && len(args) <= 2 {
		return nil
	}
	return fmt.Errorf("expected `gshoot down <spreadsheet> [sheet]`")
}

func newListCmd(stdout, stderr io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List your Google Sheets",
		Example: "  gshoot list",
		Args:  noArgs("gshoot list"),
		RunE: func(*cobra.Command, []string) error {
			ctx := context.Background()
			resolved, err := resolveAuth(auth.Options{
				Env:     auth.NewEnv(nil),
				Command: auth.CommandList,
			})
			if err != nil {
				return err
			}

			tokenSource, err := newTokenSource(ctx, resolved)
			if err != nil {
				return err
			}

			client, err := newListingClient(ctx, tokenSource)
			if err != nil {
				return err
			}

			items, err := listing.NewService(client).ListRecent(ctx, 10)
			if err != nil {
				return err
			}

			for _, item := range items {
				fmt.Fprintf(stdout, "%s  %s\n", item.ModifiedTime.UTC().Format(time.RFC3339), item.Name)
				if len(item.SheetNames) > 1 {
					preview := item.SheetNames
					suffix := ""
					if len(preview) > 3 {
						preview = preview[:3]
						suffix = ", ..."
					}
					fmt.Fprintf(stdout, "  %s%s\n", strings.Join(preview, ", "), suffix)
				}
			}

			return nil
		},
	}
	return cmd
}

func writeHelp(w io.Writer, cmd *cobra.Command) {
	if isRootCmd(cmd) {
		writeRootHelp(w, cmd)
		return
	}
	writeCommandHelp(w, cmd)
}

func writeRootHelp(w io.Writer, cmd *cobra.Command) {
	if text := commandSummary(cmd); text != "" {
		fmt.Fprintln(w, text)
		fmt.Fprintln(w)
	}
	fmt.Fprintf(w, "USAGE\n  %s <command> <subcommand> [flags]\n", cmd.Name())

	commands := availableCommands(cmd)
	if len(commands) == 0 {
		return
	}

	fmt.Fprintln(w)
	for _, sub := range commands {
		fmt.Fprintf(w, "  %s%s\n", rpad(sub.Name(), cmd.NamePadding()), sub.Short)
	}
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
			fmt.Fprintf(w, "  %s%s\n", rpad(sub.Name(), cmd.NamePadding()), sub.Short)
		}
	}

	if flags := trimmedFlagUsages(cmd.LocalFlags()); flags != "" {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "FLAGS")
		fmt.Fprint(w, indent(flags, "  "))
		fmt.Fprintln(w)
	}

	if flags := trimmedFlagUsages(cmd.InheritedFlags()); flags != "" {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "INHERITED FLAGS")
		fmt.Fprint(w, indent(flags, "  "))
		fmt.Fprintln(w)
	}

	if example := strings.TrimSpace(cmd.Example); example != "" {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "EXAMPLES")
		fmt.Fprintln(w, indent(example, "  "))
	}
}

func isRootCmd(cmd *cobra.Command) bool {
	return cmd != nil && !cmd.HasParent()
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

func trimmedFlagUsages(flags *pflag.FlagSet) string {
	if flags == nil {
		return ""
	}
	return strings.TrimRight(flags.FlagUsages(), "\n")
}

func indent(text, prefix string) string {
	if text == "" {
		return ""
	}
	var buf bytes.Buffer
	for i, line := range strings.Split(text, "\n") {
		if i > 0 {
			buf.WriteByte('\n')
		}
		buf.WriteString(prefix)
		buf.WriteString(line)
	}
	return buf.String()
}

func rpad(text string, padding int) string {
	if len(text) >= padding {
		return text + " "
	}
	return text + strings.Repeat(" ", padding-len(text))
}
