package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gurgeous/gshoot/internal/auth"
	"github.com/gurgeous/gshoot/internal/down"
	"github.com/gurgeous/gshoot/internal/listing"
	"golang.org/x/oauth2"
)

func TestRunRootHelp(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"--help"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run() code = %d, want 0", code)
	}

	output := stdout.String()
	for _, want := range []string{"gshoot", "auth", "up", "down", "list"} {
		if !strings.Contains(output, want) {
			t.Fatalf("help output missing %q:\n%s", want, output)
		}
	}
	if strings.Contains(output, "Flags:") {
		t.Fatalf("help output = %q, want no flags section", output)
	}

	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunUnknownCommand(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"wat"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("Run() code = %d, want 1", code)
	}

	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}

	if !strings.Contains(stderr.String(), "unknown command") {
		t.Fatalf("stderr = %q, want unknown command", stderr.String())
	}
}

func TestRunSubcommandHelp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "auth", args: []string{"auth", "--help"}, want: "Login (or logout) from Google Sheets"},
		{name: "auth status", args: []string{"auth", "status", "--help"}, want: "Show auth status"},
		{name: "auth logout", args: []string{"auth", "logout", "--help"}, want: "Clear cached OAuth token"},
		{name: "up", args: []string{"up", "--help"}, want: "Upload a local CSV file to a Google Sheet"},
		{name: "down", args: []string{"down", "--help"}, want: "Download a Google Sheet as CSV"},
		{name: "list", args: []string{"list", "--help"}, want: "List your Google Sheets"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var stdout bytes.Buffer
			var stderr bytes.Buffer

			code := Run(tt.args, &stdout, &stderr)
			if code != 0 {
				t.Fatalf("Run() code = %d, want 0", code)
			}

			if !strings.Contains(stdout.String(), tt.want) {
				t.Fatalf("stdout = %q, want %q", stdout.String(), tt.want)
			}

			if stderr.Len() != 0 {
				t.Fatalf("stderr = %q, want empty", stderr.String())
			}
		})
	}
}

func TestRunList(t *testing.T) {
	restore := stubListDeps(t)
	defer restore()

	resolveAuth = func(opts auth.Options) (auth.Resolved, error) {
		if opts.Command != auth.CommandList {
			t.Fatalf("Resolve() command = %q, want list", opts.Command)
		}
		return auth.Resolved{Scopes: auth.ScopesForCommand(auth.CommandList)}, nil
	}
	newTokenSource = func(_ context.Context, _ auth.Resolved) (oauth2.TokenSource, error) {
		return oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "token"}), nil
	}
	newListingClient = func(_ context.Context, _ oauth2.TokenSource) (listing.Client, error) {
		return &fakeListingClient{
			items: []listing.DriveSpreadsheet{
				{ID: "1", Name: "Alpha", ModifiedTime: time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)},
				{ID: "2", Name: "Beta", ModifiedTime: time.Date(2026, 5, 7, 11, 0, 0, 0, time.UTC)},
			},
			sheets: map[string][]string{
				"1": {"One", "Two", "Three", "Four"},
				"2": {"Only"},
			},
		}, nil
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"list"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run() code = %d, want 0", code)
	}

	output := stdout.String()
	for _, want := range []string{
		"2026-05-07T12:00:00Z  Alpha",
		"One, Two, Three, ...",
		"2026-05-07T11:00:00Z  Beta",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("stdout missing %q:\n%s", want, output)
		}
	}
	if strings.Contains(output, "Only") {
		t.Fatalf("stdout = %q, want no preview for single-sheet spreadsheet", output)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunListAuthError(t *testing.T) {
	restore := stubListDeps(t)
	defer restore()

	resolveAuth = func(auth.Options) (auth.Resolved, error) {
		return auth.Resolved{}, errors.New("no auth")
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"list"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("Run() code = %d, want 1", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "no auth") {
		t.Fatalf("stderr = %q, want auth error", stderr.String())
	}
}

func TestRunDownStdout(t *testing.T) {
	restore := stubDownDeps(t)
	defer restore()

	resolveAuth = func(opts auth.Options) (auth.Resolved, error) {
		if opts.Command != auth.CommandDown {
			t.Fatalf("Resolve() command = %q, want down", opts.Command)
		}
		return auth.Resolved{Scopes: auth.ScopesForCommand(auth.CommandDown)}, nil
	}
	newTokenSource = func(_ context.Context, _ auth.Resolved) (oauth2.TokenSource, error) {
		return oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "token"}), nil
	}
	newDownClient = func(_ context.Context, _ oauth2.TokenSource) (down.Client, error) {
		return &fakeDownClient{
			spreadsheets: []down.DriveSpreadsheet{
				{ID: "spreadsheet", Name: "Budget"},
			},
			sheets: map[string][]down.Sheet{
				"spreadsheet": {
					{ID: 10, Title: "Sheet1"},
				},
			},
			values: map[string][][]string{
				"spreadsheet/Sheet1": {
					{"name", "count"},
					{"alpha", "1"},
				},
			},
		}, nil
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"down", "Budget"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run() code = %d, want 0", code)
	}
	if stdout.String() != "name,count\nalpha,1\n" {
		t.Fatalf("stdout = %q, want csv", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunDownOutputFile(t *testing.T) {
	restore := stubDownDeps(t)
	defer restore()

	resolveAuth = func(auth.Options) (auth.Resolved, error) {
		return auth.Resolved{Scopes: auth.ScopesForCommand(auth.CommandDown)}, nil
	}
	newTokenSource = func(_ context.Context, _ auth.Resolved) (oauth2.TokenSource, error) {
		return oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "token"}), nil
	}
	newDownClient = func(_ context.Context, _ oauth2.TokenSource) (down.Client, error) {
		return &fakeDownClient{
			spreadsheets: []down.DriveSpreadsheet{
				{ID: "spreadsheet", Name: "Budget"},
			},
			sheets: map[string][]down.Sheet{
				"spreadsheet": {
					{ID: 10, Title: "Sheet1"},
				},
			},
			values: map[string][][]string{
				"spreadsheet/Sheet1": {
					{"name", "count"},
					{"alpha", "1"},
				},
			},
		}, nil
	}

	path := filepath.Join(t.TempDir(), "out.csv")
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"down", "Budget", "--output", path}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run() code = %d, want 0", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != "name,count\nalpha,1\n" {
		t.Fatalf("output file = %q, want csv", string(data))
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunDownMissingSpreadsheet(t *testing.T) {
	restore := stubDownDeps(t)
	defer restore()

	resolveAuth = func(auth.Options) (auth.Resolved, error) {
		return auth.Resolved{Scopes: auth.ScopesForCommand(auth.CommandDown)}, nil
	}
	newTokenSource = func(_ context.Context, _ auth.Resolved) (oauth2.TokenSource, error) {
		return oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "token"}), nil
	}
	newDownClient = func(_ context.Context, _ oauth2.TokenSource) (down.Client, error) {
		return &fakeDownClient{}, nil
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"down", "Budget"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("Run() code = %d, want 1", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "hint: run `gshoot list`") {
		t.Fatalf("stderr = %q, want list hint", stderr.String())
	}
}

func TestRunDownNoSheets(t *testing.T) {
	restore := stubDownDeps(t)
	defer restore()

	resolveAuth = func(auth.Options) (auth.Resolved, error) {
		return auth.Resolved{Scopes: auth.ScopesForCommand(auth.CommandDown)}, nil
	}
	newTokenSource = func(_ context.Context, _ auth.Resolved) (oauth2.TokenSource, error) {
		return oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "token"}), nil
	}
	newDownClient = func(_ context.Context, _ oauth2.TokenSource) (down.Client, error) {
		return &fakeDownClient{
			spreadsheets: []down.DriveSpreadsheet{
				{ID: "spreadsheet", Name: "Budget"},
			},
			sheets: map[string][]down.Sheet{},
		}, nil
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"down", "Budget"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("Run() code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "spreadsheet has no sheets") {
		t.Fatalf("stderr = %q, want no-sheets error", stderr.String())
	}
}

func stubListDeps(t *testing.T) func() {
	t.Helper()

	origResolve := resolveAuth
	origToken := newTokenSource
	origClient := newListingClient
	return func() {
		resolveAuth = origResolve
		newTokenSource = origToken
		newListingClient = origClient
	}
}

func stubDownDeps(t *testing.T) func() {
	t.Helper()

	origResolve := resolveAuth
	origToken := newTokenSource
	origClient := newDownClient
	return func() {
		resolveAuth = origResolve
		newTokenSource = origToken
		newDownClient = origClient
	}
}

type fakeListingClient struct {
	items  []listing.DriveSpreadsheet
	sheets map[string][]string
}

func (f *fakeListingClient) ListSpreadsheets(context.Context, int) ([]listing.DriveSpreadsheet, error) {
	return f.items, nil
}

func (f *fakeListingClient) ListSheetNames(_ context.Context, spreadsheetID string) ([]string, error) {
	return f.sheets[spreadsheetID], nil
}

type fakeDownClient struct {
	spreadsheets []down.DriveSpreadsheet
	sheets       map[string][]down.Sheet
	values       map[string][][]string
}

func (f *fakeDownClient) ListSpreadsheets(context.Context) ([]down.DriveSpreadsheet, error) {
	return f.spreadsheets, nil
}

func (f *fakeDownClient) ListSheets(_ context.Context, spreadsheetID string) ([]down.Sheet, error) {
	return f.sheets[spreadsheetID], nil
}

func (f *fakeDownClient) GetValues(_ context.Context, spreadsheetID, sheetTitle string) ([][]string, error) {
	return f.values[spreadsheetID+"/"+sheetTitle], nil
}
