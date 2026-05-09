package smoke

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gurgeous/gshoot/internal/auth"
	"github.com/gurgeous/gshoot/internal/ux"
	"golang.org/x/oauth2"
)

const (
	spreadsheetName = "gshoot-smoke"
	downSheetName   = "down-basic"
	expectedDownCSV = "name,count\nalpha,1\nbeta,2\n"
)

var (
	executablePath = os.Executable
	resolveAuth    = auth.Resolve
	newTokenSource = auth.NewTokenSource
	newSmokeClient = func(ctx context.Context, tokenSource oauth2.TokenSource) (Client, error) {
		return NewGoogleClient(ctx, tokenSource)
	}
	runCommand = func(name string, args ...string) error {
		cmd := exec.Command(name, args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	smokeTmpDir = func() string {
		return filepath.Join("tmp", "output")
	}
	readFile = os.ReadFile
	mkdirAll = func(path string) error {
		return os.MkdirAll(path, 0o755)
	}
)

// Client manages smoke fixtures directly via the Google APIs.
type Client interface {
	ResetDownFixture(ctx context.Context, spreadsheetName, sheetName string, values [][]string) (string, error)
}

// Run executes the manual smoke-test entrypoint.
func Run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("smoke", flag.ContinueOnError)
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
		fmt.Fprintln(stderr, ux.Error.Render("missing gshoot command"))
		return 1
	}

	ctx := context.Background()
	resolved, err := resolveAuth(auth.Options{
		Command: auth.CommandUp,
	})
	if err != nil {
		fmt.Fprintln(stderr, ux.Error.Render(err.Error()))
		return 1
	}

	tokenSource, err := newTokenSource(ctx, resolved)
	if err != nil {
		fmt.Fprintln(stderr, ux.Error.Render(err.Error()))
		return 1
	}

	client, err := newSmokeClient(ctx, tokenSource)
	if err != nil {
		fmt.Fprintln(stderr, ux.Error.Render(err.Error()))
		return 1
	}

	if err := mkdirAll(smokeTmpDir()); err != nil {
		fmt.Fprintln(stderr, ux.Error.Render(err.Error()))
		return 1
	}

	fmt.Fprintln(stdout, ux.Info.Render("resetting "+spreadsheetName+"/"+downSheetName+"..."))
	spreadsheetID, err := client.ResetDownFixture(ctx, spreadsheetName, downSheetName, downFixtureValues())
	if err != nil {
		fmt.Fprintln(stderr, ux.Error.Render(err.Error()))
		return 1
	}

	outputPath := filepath.Join(smokeTmpDir(), "smoke-down.csv")
	command := append(append([]string(nil), gshootCommand...), "down", spreadsheetName, downSheetName, "--output", outputPath)
	fmt.Fprintln(stdout, ux.Info.Render("running "+strings.Join(command, " ")))
	if err := runCommand(command[0], command[1:]...); err != nil {
		fmt.Fprintln(stderr, ux.Error.Render(err.Error()))
		return 1
	}

	data, err := readFile(outputPath)
	if err != nil {
		fmt.Fprintln(stderr, ux.Error.Render(err.Error()))
		return 1
	}
	if string(data) != expectedDownCSV {
		fmt.Fprintln(stderr, ux.Error.Render("unexpected down output:\n"+string(data)))
		return 1
	}

	fmt.Fprintln(stdout, ux.Success.Render("smoke ok: https://docs.google.com/spreadsheets/d/"+spreadsheetID+"/edit"))
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

func downFixtureValues() [][]string {
	return [][]string{
		{"name", "count"},
		{"alpha", "1"},
		{"beta", "2"},
	}
}
