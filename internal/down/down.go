package down

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

// DriveSpreadsheet is the minimal Drive-side spreadsheet metadata.
type DriveSpreadsheet struct {
	ID           string
	Name         string
	ModifiedTime time.Time
}

// Sheet is the minimal sheet metadata needed for downloads.
type Sheet struct {
	ID    int64
	Title string
}

// Client provides spreadsheet lookup and read operations.
type Client interface {
	ListSpreadsheets(ctx context.Context) ([]DriveSpreadsheet, error)
	ListSheets(ctx context.Context, spreadsheetID string) ([]Sheet, error)
	GetValues(ctx context.Context, spreadsheetID, sheetTitle string) ([][]string, error)
}

// SpreadsheetNotFoundError reports a missing spreadsheet.
type SpreadsheetNotFoundError struct {
	Name string
}

func (e *SpreadsheetNotFoundError) Error() string {
	return fmt.Sprintf("spreadsheet not found: %s", e.Name)
}

// SheetNotFoundError reports a missing sheet.
type SheetNotFoundError struct {
	Spreadsheet string
	Sheet       string
}

func (e *SheetNotFoundError) Error() string {
	return fmt.Sprintf("sheet not found in %s: %s", e.Spreadsheet, e.Sheet)
}

// NoSheetsError reports a spreadsheet with no sheets at all.
type NoSheetsError struct {
	Spreadsheet string
}

func (e *NoSheetsError) Error() string {
	return fmt.Sprintf("spreadsheet has no sheets: %s", e.Spreadsheet)
}

// Result is one completed download result.
type Result struct {
	Spreadsheet DriveSpreadsheet
	Sheet       Sheet
	Values      [][]string
}

// Service downloads sheet data.
type Service struct {
	client Client
}

// NewService creates a download service.
func NewService(client Client) Service {
	return Service{client: client}
}

// Download finds a spreadsheet and sheet, then fetches rectangular CSV rows.
func (s Service) Download(ctx context.Context, spreadsheetName, sheetName string) (Result, error) {
	driveItems, err := s.client.ListSpreadsheets(ctx)
	if err != nil {
		return Result{}, err
	}

	spreadsheet, ok := findSpreadsheet(driveItems, spreadsheetName)
	if !ok {
		return Result{}, &SpreadsheetNotFoundError{Name: spreadsheetName}
	}

	sheets, err := s.client.ListSheets(ctx, spreadsheet.ID)
	if err != nil {
		return Result{}, err
	}
	if len(sheets) == 0 {
		return Result{}, &NoSheetsError{Spreadsheet: spreadsheet.Name}
	}

	sheet, ok := chooseSheet(sheets, sheetName)
	if !ok {
		return Result{}, &SheetNotFoundError{Spreadsheet: spreadsheet.Name, Sheet: sheetName}
	}

	values, err := s.client.GetValues(ctx, spreadsheet.ID, sheet.Title)
	if err != nil {
		return Result{}, err
	}

	return Result{
		Spreadsheet: spreadsheet,
		Sheet:       sheet,
		Values:      rectangularize(values),
	}, nil
}

// WriteCSV writes rows as CSV.
func WriteCSV(w io.Writer, rows [][]string) error {
	writer := csv.NewWriter(w)
	writer.WriteAll(rows)
	return writer.Error()
}

func findSpreadsheet(items []DriveSpreadsheet, target string) (DriveSpreadsheet, bool) {
	sorted := append([]DriveSpreadsheet(nil), items...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].ModifiedTime.Equal(sorted[j].ModifiedTime) {
			return sorted[i].Name < sorted[j].Name
		}
		return sorted[i].ModifiedTime.After(sorted[j].ModifiedTime)
	})

	for _, item := range sorted {
		if nameMatch(item.Name, target) {
			return item, true
		}
	}
	return DriveSpreadsheet{}, false
}

func chooseSheet(sheets []Sheet, target string) (Sheet, bool) {
	if target == "" {
		return sheets[0], true
	}
	for _, sheet := range sheets {
		if nameMatch(sheet.Title, target) {
			return sheet, true
		}
	}
	return Sheet{}, false
}

func rectangularize(rows [][]string) [][]string {
	max := 0
	for _, row := range rows {
		if len(row) > max {
			max = len(row)
		}
	}

	out := make([][]string, 0, len(rows))
	for _, row := range rows {
		copyRow := append([]string(nil), row...)
		if len(copyRow) < max {
			copyRow = append(copyRow, make([]string, max-len(copyRow))...)
		}
		out = append(out, copyRow)
	}
	return out
}

func nameMatch(lhs, rhs string) bool {
	return strings.EqualFold(lhs, rhs)
}
