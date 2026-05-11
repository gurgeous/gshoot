package down

import (
	"context"
	"fmt"

	"github.com/gurgeous/gshoot/internal/google"
)

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

// Download finds a spreadsheet and sheet, then fetches rectangular CSV rows.
func Download(ctx context.Context, client *google.Client, spreadsheetName, sheetName string) ([][]string, error) {
	spreadsheet, err := client.FindSpreadsheet(ctx, spreadsheetName)
	if err != nil {
		return nil, err
	}
	if spreadsheet == nil {
		return nil, &SpreadsheetNotFoundError{Name: spreadsheetName}
	}

	sheets, err := client.ListSheets(ctx, spreadsheet.Id)
	if err != nil {
		return nil, err
	}
	if len(sheets) == 0 {
		return nil, &NoSheetsError{Spreadsheet: spreadsheet.Name}
	}

	sheet, err := client.FindSheet(ctx, spreadsheet.Id, sheetName)
	if err != nil {
		return nil, err
	}
	if sheet == nil {
		return nil, &SheetNotFoundError{Spreadsheet: spreadsheet.Name, Sheet: sheetName}
	}

	values, err := client.GetRows(ctx, spreadsheet.Id, sheet)
	if err != nil {
		return nil, err
	}

	return google.Rectangularize(values), nil
}
