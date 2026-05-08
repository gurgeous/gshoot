package down

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"
)

func TestServiceDownloadDefaultSheet(t *testing.T) {
	t.Parallel()

	client := &fakeClient{
		spreadsheets: []DriveSpreadsheet{
			{ID: "new", Name: "budget", ModifiedTime: time.Date(2026, 5, 7, 8, 0, 0, 0, time.UTC)},
			{ID: "old", Name: "Budget", ModifiedTime: time.Date(2026, 5, 6, 8, 0, 0, 0, time.UTC)},
		},
		sheets: map[string][]Sheet{
			"new": {
				{ID: 10, Title: "Sheet1"},
				{ID: 20, Title: "Other"},
			},
		},
		values: map[string][][]string{
			"new/Sheet1": {
				{"name", "count"},
				{"alpha"},
				{"beta", "2", "extra"},
			},
		},
	}

	result, err := NewService(client).Download(context.Background(), "Budget", "")
	if err != nil {
		t.Fatalf("Download() error = %v", err)
	}

	if result.Spreadsheet.ID != "new" {
		t.Fatalf("Download() spreadsheet id = %q, want newest match", result.Spreadsheet.ID)
	}
	if result.Sheet.Title != "Sheet1" {
		t.Fatalf("Download() sheet title = %q, want first sheet", result.Sheet.Title)
	}

	want := [][]string{
		{"name", "count", ""},
		{"alpha", "", ""},
		{"beta", "2", "extra"},
	}
	if !equalRows(result.Values, want) {
		t.Fatalf("Download() values = %#v, want %#v", result.Values, want)
	}
}

func TestServiceDownloadNamedSheet(t *testing.T) {
	t.Parallel()

	client := &fakeClient{
		spreadsheets: []DriveSpreadsheet{
			{ID: "1", Name: "Budget", ModifiedTime: time.Date(2026, 5, 7, 8, 0, 0, 0, time.UTC)},
		},
		sheets: map[string][]Sheet{
			"1": {
				{ID: 10, Title: "Sheet1"},
				{ID: 20, Title: "Summary"},
			},
		},
		values: map[string][][]string{
			"1/Summary": {
				{"month", "total"},
				{"May", "10"},
			},
		},
	}

	result, err := NewService(client).Download(context.Background(), "Budget", "summary")
	if err != nil {
		t.Fatalf("Download() error = %v", err)
	}

	if result.Sheet.Title != "Summary" {
		t.Fatalf("Download() sheet title = %q, want Summary", result.Sheet.Title)
	}
}

func TestServiceDownloadSpreadsheetNotFound(t *testing.T) {
	t.Parallel()

	_, err := NewService(&fakeClient{}).Download(context.Background(), "Budget", "")
	if err == nil {
		t.Fatal("Download() error = nil, want error")
	}

	var notFound *SpreadsheetNotFoundError
	if !errors.As(err, &notFound) {
		t.Fatalf("Download() error = %T, want SpreadsheetNotFoundError", err)
	}
}

func TestServiceDownloadSheetNotFound(t *testing.T) {
	t.Parallel()

	client := &fakeClient{
		spreadsheets: []DriveSpreadsheet{
			{ID: "1", Name: "Budget", ModifiedTime: time.Date(2026, 5, 7, 8, 0, 0, 0, time.UTC)},
		},
		sheets: map[string][]Sheet{
			"1": {
				{ID: 10, Title: "Sheet1"},
			},
		},
	}

	_, err := NewService(client).Download(context.Background(), "Budget", "Summary")
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

func TestWriteCSV(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	rows := [][]string{
		{"name", "count"},
		{"alpha", "1"},
		{"beta", "2,3"},
	}

	if err := WriteCSV(&out, rows); err != nil {
		t.Fatalf("WriteCSV() error = %v", err)
	}

	want := "name,count\nalpha,1\nbeta,\"2,3\"\n"
	if out.String() != want {
		t.Fatalf("WriteCSV() = %q, want %q", out.String(), want)
	}
}

type fakeClient struct {
	spreadsheets []DriveSpreadsheet
	sheets       map[string][]Sheet
	values       map[string][][]string
}

func (f *fakeClient) ListSpreadsheets(context.Context) ([]DriveSpreadsheet, error) {
	return f.spreadsheets, nil
}

func (f *fakeClient) ListSheets(_ context.Context, spreadsheetID string) ([]Sheet, error) {
	return f.sheets[spreadsheetID], nil
}

func (f *fakeClient) GetValues(_ context.Context, spreadsheetID, sheetTitle string) ([][]string, error) {
	return f.values[spreadsheetID+"/"+sheetTitle], nil
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
