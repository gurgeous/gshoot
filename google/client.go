package google

import (
	"context"
	"fmt"
	"strings"

	"github.com/gurgeous/gshoot/auth"
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
//

func ReadOnlyScopes() []string {
	return []string{}
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
	tokenSource, err := auth.NewTokenSource(ctx, scopes)
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
// https://developers.google.com/workspace/drive/api/reference/rest/v3/files/list
func (c *Client) ListSpreadsheets(ctx context.Context, limit int) ([]*drive.File, error) {
	if limit <= 0 {
		limit = defaultSpreadsheetLimit
	}

	res, err := c.Drive.Files.List().
		Context(ctx).
		Q("mimeType='application/vnd.google-apps.spreadsheet' and trashed=false").
		OrderBy("modifiedByMeTime desc, name").
		PageSize(int64(limit)).
		Fields("files(*)").
		Do()
	if err != nil {
		return nil, err
	}
	// litter.Dump(res.Files[0])
	return res.Files, nil
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

// GetSheets returns the sheets (tabs) in a spreadsheet.
// https://developers.google.com/workspace/sheets/api/reference/rest/v4/spreadsheets/get
func (c *Client) GetSheets(ctx context.Context, spreadsheetID string) ([]*sheets.Sheet, error) {
	res, err := c.Sheets.Spreadsheets.Get(spreadsheetID).
		Context(ctx).
		Fields("sheets(properties(*))").
		Do()
	if err != nil {
		return nil, err
	}
	return res.Sheets, nil
}

// FindSheet returns the sheet with this name, or the first sheet when name is empty (case insensitive)
// see ListSpreadsheets for scopes
func (c *Client) FindSheet(ctx context.Context, spreadsheetID, name string) (*sheets.Sheet, error) {
	items, err := c.GetSheets(ctx, spreadsheetID)
	if err != nil {
		return nil, err
	}
	if name == "" {
		return items[0], nil
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
	// https://developers.google.com/workspace/sheets/api/reference/rest/v4/spreadsheets.values/get
	res, err := c.Sheets.Spreadsheets.Values.Get(spreadsheetID, sheetRange(sheet)).
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

	return Rectangularize(rows), nil
}

func Rectangularize(rows Rows) Rows {
	cols := 0
	for _, row := range rows {
		cols = max(cols, len(row))
	}

	out := make([][]string, 0, len(rows))
	for _, src := range rows {
		dst := append([]string(nil), src...)
		if len(dst) < cols {
			dst = append(dst, make([]string, cols-len(dst))...)
		}
		out = append(out, dst)
	}
	return out
}

//
// helpers
//

// Turn sheet title into a quote range (for Values.Get, etc)
func sheetRange(sheet *sheets.Sheet) string {
	escaped := strings.ReplaceAll(sheet.Properties.Title, "'", "''")
	return fmt.Sprintf("'%s'", escaped)
}
