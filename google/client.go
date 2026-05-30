package google

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/gurgeous/gshoot/auth"
	"github.com/gurgeous/gshoot/util"
)

const spreadsheetMimeType = "application/vnd.google-apps.spreadsheet"

const (
	driveBaseURL  = "https://www.googleapis.com"
	sheetsBaseURL = "https://sheets.googleapis.com"
)

//
// google api client
//

type Client struct {
	httpClient    *http.Client
	driveBaseURL  string
	sheetsBaseURL string
}

// NewClient creates a Google API client with saved auth.
func NewClient(ctx context.Context) (*Client, error) {
	manager, err := auth.NewManager()
	if err != nil {
		return nil, err
	}
	httpClient, err := manager.HTTPClient(ctx)
	if err != nil {
		return nil, err
	}

	return &Client{
		httpClient:    httpClient,
		driveBaseURL:  driveBaseURL,
		sheetsBaseURL: sheetsBaseURL,
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
	q.Set("q", fmt.Sprintf("mimeType='%s' and trashed=false", spreadsheetMimeType))
	q.Set("orderBy", "modifiedByMeTime desc, name")
	q.Set("pageSize", fmt.Sprint(limit))
	q.Set("fields", "files(id,name,modifiedByMeTime)")

	var res struct {
		Files []*File `json:"files"`
	}
	if err := c.driveReq(ctx, "/drive/v3/files", q, &res); err != nil {
		return nil, err
	}
	return res.Files, nil
}

// CreateSpreadsheet creates a Google Sheets file.
// https://developers.google.com/drive/api/reference/rest/v3/files/create
func (c *Client) CreateSpreadsheet(ctx context.Context, name string) (*File, error) {
	q := url.Values{}
	q.Set("fields", "id,name")

	body := File{
		Name:     name,
		MimeType: spreadsheetMimeType,
	}
	var res File
	if err := c.driveReqJSON(ctx, http.MethodPost, "/drive/v3/files", q, body, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// GetSheets returns the sheets (tabs) in a spreadsheet.
// https://developers.google.com/workspace/sheets/api/reference/rest/v4/spreadsheets/get
func (c *Client) GetSheets(ctx context.Context, spreadsheetID string) ([]*Sheet, error) {
	spreadsheet, err := c.GetSpreadsheet(ctx, spreadsheetID)
	if err != nil {
		return nil, err
	}
	return spreadsheet.Sheets, nil
}

// GetSpreadsheet returns spreadsheet metadata.
// https://developers.google.com/workspace/sheets/api/reference/rest/v4/spreadsheets/get
func (c *Client) GetSpreadsheet(ctx context.Context, spreadsheetID string) (*Spreadsheet, error) {
	return c.getSpreadsheet(ctx, spreadsheetID, false)
}

// GetSpreadsheetWithGridData returns spreadsheet metadata plus grid data for ranges.
// https://developers.google.com/workspace/sheets/api/reference/rest/v4/spreadsheets/get
func (c *Client) GetSpreadsheetWithGridData(ctx context.Context, spreadsheetID string, ranges ...string) (*Spreadsheet, error) {
	return c.getSpreadsheet(ctx, spreadsheetID, true, ranges...)
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
	if err := c.sheetsReq(ctx, path, nil, &res); err != nil {
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

	return Rows(util.CSVRectangularize(rows)), nil
}

// BatchUpdate sends one or more Sheets mutation requests and returns API replies.
// https://developers.google.com/workspace/sheets/api/reference/rest/v4/spreadsheets/batchUpdate
func (c *Client) BatchUpdate(ctx context.Context, spreadsheetID string, requests []Request) (*BatchUpdateResponse, error) {
	path := fmt.Sprintf("/v4/spreadsheets/%s:batchUpdate", url.PathEscape(spreadsheetID))
	body := map[string]any{"requests": requests}
	var res BatchUpdateResponse
	if err := c.sheetsReqJSON(ctx, http.MethodPost, path, nil, body, &res); err != nil {
		return nil, err
	}
	return &res, nil
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
func (c *Client) FindSheet(ctx context.Context, spreadsheetID, name string) (*Sheet, error) {
	items, err := c.GetSheets(ctx, spreadsheetID)
	if err != nil {
		return nil, err
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

func (c *Client) getSpreadsheet(ctx context.Context, spreadsheetID string, includeGridData bool, ranges ...string) (*Spreadsheet, error) {
	path := fmt.Sprintf("/v4/spreadsheets/%s", url.PathEscape(spreadsheetID))
	q := url.Values{}
	fields := "sheets(properties(sheetId,title,gridProperties))"
	if includeGridData {
		fields = "sheets(properties(sheetId,title,gridProperties),basicFilter,data(rowData(values(userEnteredValue)),columnMetadata(pixelSize)))"
		q.Set("includeGridData", "true")
	}
	q.Set("fields", fields)
	for _, rng := range ranges {
		q.Add("ranges", rng)
	}

	var res spreadsheetResponse
	if err := c.sheetsReq(ctx, path, q, &res); err != nil {
		return nil, err
	}
	return res.spreadsheet(), nil
}

// driveReq sends a Drive GET request and decodes JSON.
func (c *Client) driveReq(ctx context.Context, path string, q url.Values, dst any) error {
	return c.driveReqJSON(ctx, http.MethodGet, path, q, nil, dst)
}

// driveReqJSON sends a Drive JSON request.
func (c *Client) driveReqJSON(ctx context.Context, method string, path string, q url.Values, body any, dst any) error {
	return c.reqJSON(ctx, c.driveBaseURL, method, path, q, body, dst)
}

// sheetsReq sends a Sheets GET request and decodes JSON.
func (c *Client) sheetsReq(ctx context.Context, path string, q url.Values, dst any) error {
	return c.sheetsReqJSON(ctx, http.MethodGet, path, q, nil, dst)
}

// sheetsReqJSON sends a Sheets JSON request.
func (c *Client) sheetsReqJSON(ctx context.Context, method string, path string, q url.Values, body any, dst any) error {
	return c.reqJSON(ctx, c.sheetsBaseURL, method, path, q, body, dst)
}

// reqJSON sends a JSON request, checks Google errors, and decodes JSON.
func (c *Client) reqJSON(ctx context.Context, baseURL string, method string, path string, q url.Values, body any, dst any) error {
	// path+q => url
	url := strings.TrimRight(baseURL, "/") + path
	if len(q) > 0 {
		url += "?" + q.Encode()
	}

	var bodyReader io.Reader
	if body != nil {
		var buf bytes.Buffer
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return fmt.Errorf("encode google api request: %w", err)
		}
		bodyReader = &buf
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
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

	if dst == nil {
		return nil
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
