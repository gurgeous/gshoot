package google

import (
	"context"
	"fmt"
	"strings"

	"github.com/gurgeous/gshoot/internal/auth"
	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

//
// const
//

const defaultSpreadsheetLimit = 100

//
// scopes - readonly and rw
//

func ReadOnlyScopes() []string {
	return []string{
		"https://www.googleapis.com/auth/drive.readonly",
		"https://www.googleapis.com/auth/spreadsheets.readonly",
	}
}

func ReadWriteScopes() []string {
	return []string{
		"https://www.googleapis.com/auth/drive",
		"https://www.googleapis.com/auth/spreadsheets",
	}
}

// Client holds shared Google API services.
type Client struct {
	Drive  *drive.Service
	Sheets *sheets.Service
}

// NewClient creates a Google API client with auth for the requested scopes.
func NewClient(ctx context.Context, scopes []string) (*Client, error) {
	// auth
	resolved, err := auth.Resolve()
	if err != nil {
		return nil, err
	}
	tokenSource, err := auth.NewTokenSource(ctx, resolved, scopes)
	if err != nil {
		return nil, err
	}

	// services
	httpClient := oauth2.NewClient(ctx, tokenSource)
	drive, err := drive.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, err
	}
	sheets, err := sheets.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, err
	}

	return &Client{Drive: drive, Sheets: sheets}, nil
}

// ListSpreadsheets returns recently modified spreadsheets.
func (c *Client) ListSpreadsheets(ctx context.Context, limit int) ([]*drive.File, error) {
	if limit <= 0 {
		limit = defaultSpreadsheetLimit
	}
	res, err := c.Drive.Files.List().
		Context(ctx).
		Q("mimeType='application/vnd.google-apps.spreadsheet' and trashed=false").
		OrderBy("modifiedTime desc,name").
		PageSize(int64(limit)).
		Fields("files(id,name,modifiedTime)").
		Do()
	if err != nil {
		return nil, err
	}
	return res.Files, nil
}

// ListSheets returns the sheets (tabs) in a spreadsheet.
func (c *Client) ListSheets(ctx context.Context, spreadsheetID string) ([]*sheets.Sheet, error) {
	res, err := c.Sheets.Spreadsheets.Get(spreadsheetID).
		Context(ctx).
		Fields("sheets(properties(sheetId,title))").
		Do()
	if err != nil {
		return nil, err
	}
	return res.Sheets, nil
}

// FindSpreadsheet returns the most recent spreadsheet with this name (case insensitive)
func (c *Client) FindSpreadsheet(ctx context.Context, name string) (*drive.File, error) {
	items, err := c.ListSpreadsheets(ctx, defaultSpreadsheetLimit)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if strings.EqualFold(item.Name, name) {
			return item, nil
		}
	}
	return nil, nil // failure
}

// FirstSheet returns the first sheet
func (c *Client) FirstSheet(ctx context.Context, spreadsheetID string) (*sheets.Sheet, error) {
	items, err := c.ListSheets(ctx, spreadsheetID)
	if err != nil {
		return nil, err
	}
	return items[0], nil
}

// FindSheet returns the sheet with this name, or the first sheet when name is empty (case insensitive)
func (c *Client) FindSheet(ctx context.Context, spreadsheetID, name string) (*sheets.Sheet, error) {
	items, err := c.ListSheets(ctx, spreadsheetID)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if strings.EqualFold(item.Properties.Title, name) {
			return item, nil
		}
	}
	return nil, nil
}

type Rows [][]string

// GetRows returns stringified cell values for a sheet.
func (c *Client) GetRows(ctx context.Context, spreadsheetID string, sheet *sheets.Sheet) (Rows, error) {
	if sheet == nil || sheet.Properties == nil {
		return nil, fmt.Errorf("get values for %s: missing sheet properties", spreadsheetID)
	}
	res, err := c.Sheets.Spreadsheets.Values.Get(spreadsheetID, sheetRange(sheet.Properties.Title)).
		Context(ctx).
		Do()
	if err != nil {
		return nil, err
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

//
// misc
//

func Rectangularize(rows Rows) Rows {
	width := 0
	for _, row := range rows {
		width = max(width, len(row))
	}

	out := make([][]string, 0, len(rows))
	for _, row := range rows {
		copyRow := append([]string(nil), row...)
		if len(copyRow) < width {
			copyRow = append(copyRow, make([]string, width-len(copyRow))...)
		}
		out = append(out, copyRow)
	}
	return out
}

//
// helpers
//

// Single quotes in sheet titles are escaped by doubling them.
func sheetRange(sheetTitle string) string {
	escaped := strings.ReplaceAll(sheetTitle, "'", "''")
	return fmt.Sprintf("'%s'", escaped)
}
