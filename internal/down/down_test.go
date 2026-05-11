package down

import (
	"context"
	"errors"
	"testing"

	"github.com/gurgeous/gshoot/internal/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/sheets/v4"
)

func TestDownloadDefaultSheet(t *testing.T) {
	restore := stubDownloadDeps()
	defer restore()

	listSpreadsheets = func(context.Context, *google.Client) ([]*drive.File, error) {
		return []*drive.File{
			{Id: "new", Name: "budget", ModifiedTime: "2026-05-07T08:00:00Z"},
			{Id: "old", Name: "Budget", ModifiedTime: "2026-05-06T08:00:00Z"},
		}, nil
	}
	listSheets = func(_ context.Context, _ *google.Client, spreadsheetID string) ([]*sheets.Sheet, error) {
		if spreadsheetID != "new" {
			t.Fatalf("ListSheets() spreadsheet = %q, want new", spreadsheetID)
		}
		return []*sheets.Sheet{
			{Properties: &sheets.SheetProperties{SheetId: 10, Title: "Sheet1"}},
			{Properties: &sheets.SheetProperties{SheetId: 20, Title: "Other"}},
		}, nil
	}
	getValues = func(_ context.Context, _ *google.Client, spreadsheetID, sheetTitle string) ([][]string, error) {
		if spreadsheetID != "new" || sheetTitle != "Sheet1" {
			t.Fatalf("GetValues() args = (%q, %q)", spreadsheetID, sheetTitle)
		}
		return [][]string{
			{"name", "count"},
			{"alpha"},
			{"beta", "2", "extra"},
		}, nil
	}

	values, err := Download(context.Background(), &google.Client{}, "Budget", "")
	if err != nil {
		t.Fatalf("Download() error = %v", err)
	}

	want := [][]string{
		{"name", "count", ""},
		{"alpha", "", ""},
		{"beta", "2", "extra"},
	}
	if !equalRows(values, want) {
		t.Fatalf("Download() values = %#v, want %#v", values, want)
	}
}

func TestDownloadNamedSheet(t *testing.T) {
	restore := stubDownloadDeps()
	defer restore()

	listSpreadsheets = func(context.Context, *google.Client) ([]*drive.File, error) {
		return []*drive.File{{Id: "1", Name: "Budget", ModifiedTime: "2026-05-07T08:00:00Z"}}, nil
	}
	listSheets = func(context.Context, *google.Client, string) ([]*sheets.Sheet, error) {
		return []*sheets.Sheet{
			{Properties: &sheets.SheetProperties{SheetId: 10, Title: "Sheet1"}},
			{Properties: &sheets.SheetProperties{SheetId: 20, Title: "Summary"}},
		}, nil
	}
	getValues = func(_ context.Context, _ *google.Client, spreadsheetID, sheetTitle string) ([][]string, error) {
		if spreadsheetID != "1" || sheetTitle != "Summary" {
			t.Fatalf("GetValues() args = (%q, %q)", spreadsheetID, sheetTitle)
		}
		return [][]string{{"month", "total"}, {"May", "10"}}, nil
	}

	values, err := Download(context.Background(), &google.Client{}, "Budget", "summary")
	if err != nil {
		t.Fatalf("Download() error = %v", err)
	}
	if !equalRows(values, [][]string{{"month", "total"}, {"May", "10"}}) {
		t.Fatalf("Download() values = %#v", values)
	}
}

func TestDownloadSpreadsheetNotFound(t *testing.T) {
	restore := stubDownloadDeps()
	defer restore()

	listSpreadsheets = func(context.Context, *google.Client) ([]*drive.File, error) {
		return nil, nil
	}

	_, err := Download(context.Background(), &google.Client{}, "Budget", "")
	if err == nil {
		t.Fatal("Download() error = nil, want error")
	}

	var notFound *SpreadsheetNotFoundError
	if !errors.As(err, &notFound) {
		t.Fatalf("Download() error = %T, want SpreadsheetNotFoundError", err)
	}
}

func TestDownloadSheetNotFound(t *testing.T) {
	restore := stubDownloadDeps()
	defer restore()

	listSpreadsheets = func(context.Context, *google.Client) ([]*drive.File, error) {
		return []*drive.File{{Id: "1", Name: "Budget", ModifiedTime: "2026-05-07T08:00:00Z"}}, nil
	}
	listSheets = func(context.Context, *google.Client, string) ([]*sheets.Sheet, error) {
		return []*sheets.Sheet{
			{Properties: &sheets.SheetProperties{SheetId: 10, Title: "Sheet1"}},
		}, nil
	}

	_, err := Download(context.Background(), &google.Client{}, "Budget", "Summary")
	if err == nil {
		t.Fatal("Download() error = nil, want error")
	}

	var notFound *SheetNotFoundError
	if !errors.As(err, &notFound) {
		t.Fatalf("Download() error = %T, want SheetNotFoundError", err)
	}
	if notFound.Spreadsheet != "Budget" {
		t.Fatalf("SheetNotFoundError spreadsheet = %q, want Budget", notFound.Spreadsheet)
	}
}

func TestDownloadNoSheets(t *testing.T) {
	restore := stubDownloadDeps()
	defer restore()

	listSpreadsheets = func(context.Context, *google.Client) ([]*drive.File, error) {
		return []*drive.File{{Id: "1", Name: "Budget", ModifiedTime: "2026-05-07T08:00:00Z"}}, nil
	}
	listSheets = func(context.Context, *google.Client, string) ([]*sheets.Sheet, error) {
		return nil, nil
	}

	_, err := Download(context.Background(), &google.Client{}, "Budget", "")
	if err == nil {
		t.Fatal("Download() error = nil, want error")
	}

	var noSheets *NoSheetsError
	if !errors.As(err, &noSheets) {
		t.Fatalf("Download() error = %T, want NoSheetsError", err)
	}
}

func equalRows(lhs, rhs [][]string) bool {
	if len(lhs) != len(rhs) {
		return false
	}
	for i := range lhs {
		if len(lhs[i]) != len(rhs[i]) {
			return false
		}
		for j := range lhs[i] {
			if lhs[i][j] != rhs[i][j] {
				return false
			}
		}
	}
	return true
}

func stubDownloadDeps() func() {
	origListSpreadsheets := listSpreadsheets
	origListSheets := listSheets
	origGetValues := getValues
	return func() {
		listSpreadsheets = origListSpreadsheets
		listSheets = origListSheets
		getValues = origGetValues
	}
}
