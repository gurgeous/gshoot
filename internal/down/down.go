package down

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/gurgeous/gshoot/internal/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/sheets/v4"
)

var (
	// wrap files.list, with pagination
	listSpreadsheets = func(ctx context.Context, client *google.Client) ([]*drive.File, error) {
		items := make([]*drive.File, 0, 64)
		pageToken := ""
		for {
			call := client.Drive.Files.List().
				Context(ctx).
				Q("mimeType='application/vnd.google-apps.spreadsheet' and trashed=false").
				OrderBy("modifiedTime desc,name").
				PageSize(1000).
				Fields("nextPageToken,files(id,name,modifiedTime)")
			if pageToken != "" {
				call = call.PageToken(pageToken)
			}

			res, err := call.Do()
			if err != nil {
				return nil, fmt.Errorf("list spreadsheets: %w", err)
			}
			items = append(items, res.Files...)

			if res.NextPageToken == "" {
				return items, nil
			}
			pageToken = res.NextPageToken
		}
	}

	listSheets = func(ctx context.Context, client *google.Client, spreadsheetID string) ([]*sheets.Sheet, error) {
		res, err := client.Sheets.Spreadsheets.Get(spreadsheetID).
			Context(ctx).
			Fields("sheets(properties(sheetId,title))").
			Do()
		if err != nil {
			return nil, fmt.Errorf("list sheets for %s: %w", spreadsheetID, err)
		}
		return res.Sheets, nil
	}

	getValues = func(ctx context.Context, client *google.Client, spreadsheetID, sheetTitle string) ([][]string, error) {
		res, err := client.Sheets.Spreadsheets.Values.Get(spreadsheetID, sheetRange(sheetTitle)).
			Context(ctx).
			Do()
		if err != nil {
			return nil, fmt.Errorf("get values for %s/%s: %w", spreadsheetID, sheetTitle, err)
		}

		rows := make([][]string, 0, len(res.Values))
		for _, row := range res.Values {
			cells := make([]string, 0, len(row))
			for _, cell := range row {
				cells = append(cells, fmt.Sprint(cell))
			}
			rows = append(rows, cells)
		}
		return rows, nil
	}
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
	driveItems, err := listSpreadsheets(ctx, client)
	if err != nil {
		return nil, err
	}

	spreadsheet, ok := findSpreadsheet(driveItems, spreadsheetName)
	if !ok {
		return nil, &SpreadsheetNotFoundError{Name: spreadsheetName}
	}

	sheets, err := listSheets(ctx, client, spreadsheet.Id)
	if err != nil {
		return nil, err
	}
	if len(sheets) == 0 {
		return nil, &NoSheetsError{Spreadsheet: spreadsheet.Name}
	}

	sheetTitle, ok := chooseSheet(sheets, sheetName)
	if !ok {
		return nil, &SheetNotFoundError{Spreadsheet: spreadsheet.Name, Sheet: sheetName}
	}

	values, err := getValues(ctx, client, spreadsheet.Id, sheetTitle)
	if err != nil {
		return nil, err
	}

	return rectangularize(values), nil
}

func findSpreadsheet(items []*drive.File, target string) (*drive.File, bool) {
	sorted := append([]*drive.File(nil), items...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].ModifiedTime == sorted[j].ModifiedTime {
			return sorted[i].Name < sorted[j].Name
		}
		return sorted[i].ModifiedTime > sorted[j].ModifiedTime
	})

	for _, item := range sorted {
		if nameMatch(item.Name, target) {
			return item, true
		}
	}
	return nil, false
}

func chooseSheet(sheets []*sheets.Sheet, target string) (string, bool) {
	if target == "" {
		return sheets[0].Properties.Title, true
	}
	for _, sheet := range sheets {
		if sheet.Properties != nil && nameMatch(sheet.Properties.Title, target) {
			return sheet.Properties.Title, true
		}
	}
	return "", false
}

func rectangularize(rows [][]string) [][]string {
	maxWidth := 0
	for _, row := range rows {
		if len(row) > maxWidth {
			maxWidth = len(row)
		}
	}

	out := make([][]string, 0, len(rows))
	for _, row := range rows {
		copyRow := append([]string(nil), row...)
		if len(copyRow) < maxWidth {
			copyRow = append(copyRow, make([]string, maxWidth-len(copyRow))...)
		}
		out = append(out, copyRow)
	}
	return out
}

func nameMatch(lhs, rhs string) bool {
	return strings.EqualFold(lhs, rhs)
}

func sheetRange(sheetTitle string) string {
	escaped := strings.ReplaceAll(sheetTitle, "'", "''")
	return fmt.Sprintf("'%s'", escaped)
}
