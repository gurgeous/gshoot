package google

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/gurgeous/gshoot/auth"
	"golang.org/x/oauth2"
)

//
// scopes
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

//
// google api client
//

type Client struct {
	httpClient *http.Client
	baseURL    string
}

//
// a file from google docs
//

type File struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	ModifiedByMeTime string `json:"modifiedByMeTime"`
}

//
// one sheet from a google spreadsheet file
//

type Sheet struct {
	ID    int64  `json:"sheetId"`
	Title string `json:"title"`
}

//
// data from a sheet
//

type Rows [][]string

// NewClient creates a Google API client with auth for the requested scopes.
func NewClient(ctx context.Context, scopes []string) (*Client, error) {
	tokenSource, err := auth.NewManager().TokenSource(ctx, scopes)
	if err != nil {
		return nil, err
	}

	return &Client{
		httpClient: oauth2.NewClient(ctx, tokenSource),
		baseURL:    "https://www.googleapis.com",
	}, nil
}

//
// api requests
//

// ListSpreadsheets returns recently modified spreadsheets.
// https://developers.google.com/workspace/drive/api/reference/rest/v3/files/list
func (c *Client) ListSpreadsheets(ctx context.Context, limit int) ([]*File, error) {
	if limit <= 0 {
		limit = 100
	}

	q := url.Values{}
	q.Set("q", "mimeType='application/vnd.google-apps.spreadsheet' and trashed=false")
	q.Set("orderBy", "modifiedByMeTime desc, name")
	q.Set("pageSize", fmt.Sprint(limit))
	q.Set("fields", "files(id,name,modifiedByMeTime)")

	var res struct {
		Files []*File `json:"files"`
	}
	if err := c.req(ctx, "/drive/v3/files", q, &res); err != nil {
		return nil, err
	}
	return res.Files, nil
}

// GetSheets returns the sheets (tabs) in a spreadsheet.
// https://developers.google.com/workspace/sheets/api/reference/rest/v4/spreadsheets/get
func (c *Client) GetSheets(ctx context.Context, spreadsheetID string) ([]*Sheet, error) {
	path := fmt.Sprintf("/v4/spreadsheets/%s", url.PathEscape(spreadsheetID))
	q := url.Values{}
	q.Set("fields", "sheets(properties(sheetId,title))")

	var res struct {
		Sheets []struct {
			Properties *Sheet `json:"properties"`
		} `json:"sheets"`
	}
	if err := c.req(ctx, path, q, &res); err != nil {
		return nil, err
	}

	sheets := make([]*Sheet, 0, len(res.Sheets))
	for _, item := range res.Sheets {
		if item.Properties != nil {
			sheets = append(sheets, item.Properties)
		}
	}
	return sheets, nil
}

// GetRows returns stringified cell values for a sheet.
// https://developers.google.com/workspace/sheets/api/reference/rest/v4/spreadsheets.values/get
func (c *Client) GetRows(ctx context.Context, spreadsheetID string, sheetTitle string) (Rows, error) {
	path := fmt.Sprintf(
		"/v4/spreadsheets/%s/values/%s",
		url.PathEscape(spreadsheetID),
		url.PathEscape(sheetRange(sheetTitle)),
	)

	var res struct {
		Values [][]any `json:"values"`
	}
	if err := c.req(ctx, path, nil, &res); err != nil {
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

//
// nice wrappers
//

// FindSpreadsheet returns the most recent spreadsheet with this name (case insensitive).
func (c *Client) FindSpreadsheet(ctx context.Context, name string) (*File, error) {
	items, err := c.ListSpreadsheets(ctx, 0)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if strings.EqualFold(item.Name, name) {
			return item, nil
		}
	}
	return nil, nil
}

// FindSheet returns the sheet with this name, or the first sheet when name is empty.
// Real spreadsheets always have sheets; an empty list is treated as malformed API data.
func (c *Client) FindSheet(ctx context.Context, spreadsheetID, name string) (*Sheet, error) {
	items, err := c.GetSheets(ctx, spreadsheetID)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}
	if name == "" {
		return items[0], nil
	}
	for _, item := range items {
		if strings.EqualFold(item.Title, name) {
			return item, nil
		}
	}
	return nil, nil
}

//
// standalone stuff
//

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

func (c *Client) req(ctx context.Context, path string, q url.Values, dst any) error {
	// path+q => url
	url := strings.TrimRight(c.baseURL, "/") + path
	if len(q) > 0 {
		url += "?" + q.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 4<<10))
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			msg = res.Status
		}
		return fmt.Errorf("google api: %s", msg)
	}

	if err := json.NewDecoder(res.Body).Decode(dst); err != nil {
		return fmt.Errorf("decode google api response: %w", err)
	}
	return nil
}

// Turn sheet title into a quoted range for Values.Get.
func sheetRange(sheetTitle string) string {
	escaped := strings.ReplaceAll(sheetTitle, "'", "''")
	return fmt.Sprintf("'%s'", escaped)
}
