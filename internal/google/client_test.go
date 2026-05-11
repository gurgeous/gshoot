package google

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

func TestListSpreadsheets(t *testing.T) {
	var gotPageSize string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/drive/v3/files", r.URL.Path)
		gotPageSize = r.URL.Query().Get("pageSize")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"files": []map[string]string{
				{"id": "1", "name": "Alpha", "modifiedByMeTime": "2026-05-07T12:00:00Z"},
			},
		})
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	files, err := client.ListSpreadsheets(context.Background(), 0)
	assert.NoError(t, err)
	assert.Len(t, files, 1)
	assert.Equal(t, "100", gotPageSize)
	assert.Equal(t, "Alpha", files[0].Name)
}

func TestFindSpreadsheet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"files": []map[string]string{
				{"id": "1", "name": "Alpha", "modifiedByMeTime": "2026-05-07T12:00:00Z"},
				{"id": "2", "name": "Budget", "modifiedByMeTime": "2026-05-07T11:00:00Z"},
			},
		})
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	file, err := client.FindSpreadsheet(context.Background(), "budget")
	assert.NoError(t, err)
	assert.NotNil(t, file)
	assert.Equal(t, "2", file.Id)
}

func TestFindSheet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v4/spreadsheets/sheet-1", r.URL.Path)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"sheets": []map[string]any{
				{"properties": map[string]any{"sheetId": 0, "title": "Sheet1"}},
				{"properties": map[string]any{"sheetId": 1, "title": "Summary"}},
			},
		})
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	sheet, err := client.FindSheet(context.Background(), "sheet-1", "summary")
	assert.NoError(t, err)
	assert.NotNil(t, sheet)
	assert.Equal(t, "Summary", sheet.Properties.Title)
}

func TestGetRows(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v4/spreadsheets/sheet-1/values/'Summary'", r.URL.Path)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"values": []any{
				[]any{"name", "count"},
				[]any{"alpha", 1},
			},
		})
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	rows, err := client.GetRows(context.Background(), "sheet-1", newSheet("Summary"))
	assert.NoError(t, err)
	assert.Equal(t, Rows{{"name", "count"}, {"alpha", "1"}}, rows)
}

func newSheet(title string) *sheets.Sheet {
	return &sheets.Sheet{Properties: &sheets.SheetProperties{Title: title}}
}

func newTestClient(t *testing.T, serverURL string) *Client {
	t.Helper()

	httpClient := &http.Client{
		Transport: &oauth2.Transport{
			Source: oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "token"}),
			Base:   http.DefaultTransport,
		},
	}
	driveService, err := drive.NewService(
		context.Background(),
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(serverURL+"/drive/v3/"),
	)
	assert.NoError(t, err)
	sheetsService, err := sheets.NewService(
		context.Background(),
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(serverURL+"/"),
	)
	assert.NoError(t, err)
	return &Client{Drive: driveService, Sheets: sheetsService}
}
