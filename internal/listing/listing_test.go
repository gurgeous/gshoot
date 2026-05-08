package listing

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestListRecent(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	client := &fakeClient{
		spreadsheets: []DriveSpreadsheet{
			{ID: "1", Name: "Alpha", ModifiedTime: now},
			{ID: "2", Name: "Beta", ModifiedTime: now.Add(-time.Hour)},
		},
		sheetNames: map[string][]string{
			"1": {"Sheet1", "Sheet2"},
			"2": {"Only"},
		},
	}

	got, err := NewService(client).ListRecent(context.Background(), 10)
	if err != nil {
		t.Fatalf("ListRecent() error = %v", err)
	}

	want := []Spreadsheet{
		{ID: "1", Name: "Alpha", ModifiedTime: now, SheetNames: []string{"Sheet1", "Sheet2"}},
		{ID: "2", Name: "Beta", ModifiedTime: now.Add(-time.Hour), SheetNames: []string{"Only"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ListRecent() = %#v, want %#v", got, want)
	}

	if client.lastLimit != 10 {
		t.Fatalf("ListSpreadsheets() limit = %d, want 10", client.lastLimit)
	}
}

func TestListRecentDriveError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("drive failed")
	client := &fakeClient{listErr: wantErr}

	_, err := NewService(client).ListRecent(context.Background(), 10)
	if !errors.Is(err, wantErr) {
		t.Fatalf("ListRecent() error = %v, want %v", err, wantErr)
	}
}

func TestListRecentSheetsError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("sheets failed")
	client := &fakeClient{
		spreadsheets: []DriveSpreadsheet{{ID: "1", Name: "Alpha"}},
		sheetErrs:    map[string]error{"1": wantErr},
	}

	_, err := NewService(client).ListRecent(context.Background(), 10)
	if !errors.Is(err, wantErr) {
		t.Fatalf("ListRecent() error = %v, want %v", err, wantErr)
	}
}

type fakeClient struct {
	spreadsheets []DriveSpreadsheet
	sheetNames   map[string][]string
	sheetErrs    map[string]error
	listErr      error
	lastLimit    int
}

func (f *fakeClient) ListSpreadsheets(_ context.Context, limit int) ([]DriveSpreadsheet, error) {
	f.lastLimit = limit
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.spreadsheets, nil
}

func (f *fakeClient) ListSheetNames(_ context.Context, spreadsheetID string) ([]string, error) {
	if err := f.sheetErrs[spreadsheetID]; err != nil {
		return nil, err
	}
	return f.sheetNames[spreadsheetID], nil
}
