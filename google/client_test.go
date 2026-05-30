package google

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
)

func TestListSpreadsheetFiles(t *testing.T) {
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
	files, err := client.ListSpreadsheetFiles(context.Background(), 0)
	assert.NoError(t, err)
	assert.Len(t, files, 1)
	assert.Equal(t, "100", gotPageSize)
	assert.Equal(t, "Alpha", files[0].Name)
}

func TestClientUsesDocumentedBaseURLs(t *testing.T) {
	assert.Equal(t, "https://www.googleapis.com", driveBaseURL)
	assert.Equal(t, "https://sheets.googleapis.com", sheetsBaseURL)
}

func TestClientRoutesRequestsByService(t *testing.T) {
	driveHits := 0
	sheetsHits := 0

	drive := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		driveHits++
		assert.Equal(t, "/drive/v3/files", r.URL.Path)
		_ = json.NewEncoder(w).Encode(map[string]any{"files": []any{}})
	}))
	defer drive.Close()

	sheets := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sheetsHits++
		assert.Equal(t, "/v4/spreadsheets/sheet-1", r.URL.Path)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"sheets": []map[string]any{
				{"properties": map[string]any{"sheetId": 0, "title": "Sheet1"}},
			},
		})
	}))
	defer sheets.Close()

	client := newTestClientWithBaseURLs(t, drive.URL, sheets.URL)
	_, err := client.ListSpreadsheetFiles(context.Background(), 1)
	assert.NoError(t, err)
	_, err = client.GetSpreadsheet(context.Background(), "sheet-1")
	assert.NoError(t, err)
	assert.Equal(t, 1, driveHits)
	assert.Equal(t, 1, sheetsHits)
}

func TestFindSpreadsheetFile(t *testing.T) {
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
	file, err := client.FindSpreadsheetFile(context.Background(), "budget")
	assert.NoError(t, err)
	assert.NotNil(t, file)
	assert.Equal(t, "2", file.ID)
}

func TestFindOrCreateSpreadsheetFileCreatesMissingFile(t *testing.T) {
	hits := []string{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits = append(hits, r.Method+" "+r.URL.Path)
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{"files": []any{}})
		case http.MethodPost:
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "new-1", "name": "Smoke"})
		}
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	file, err := client.FindOrCreateSpreadsheetFile(context.Background(), "Smoke")
	assert.NoError(t, err)
	assert.Equal(t, "new-1", file.ID)
	assert.Equal(t, []string{"GET /drive/v3/files", "POST /drive/v3/files"}, hits)
}

func TestWipeSpreadsheetAddsSheet1ThenDeletesOldSheets(t *testing.T) {
	var batch map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v4/spreadsheets/sheet-1":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"sheets": []map[string]any{
					{"properties": map[string]any{"sheetId": 7, "title": "Sheet1"}},
					{"properties": map[string]any{"sheetId": 8, "title": "Data"}},
				},
			})
		case "/v4/spreadsheets/sheet-1:batchUpdate":
			assert.NoError(t, json.NewDecoder(r.Body).Decode(&batch))
			_ = json.NewEncoder(w).Encode(map[string]any{"replies": []any{}})
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	err := client.WipeSpreadsheet(context.Background(), "sheet-1")
	assert.NoError(t, err)
	assertBatchRequest(t, batch, 0, "updateSheetProperties", `"title":"gshoot-wipe-`)
	assertBatchRequest(t, batch, 1, "addSheet", `"title":"Sheet1"`)
	assertBatchRequest(t, batch, 2, "deleteSheet", `"sheetId":7`)
	assertBatchRequest(t, batch, 3, "deleteSheet", `"sheetId":8`)
}

func TestCreateSpreadsheetFileSendsWritableFields(t *testing.T) {
	var body map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/drive/v3/files", r.URL.Path)
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":   "sheet-1",
			"name": "Budget",
		})
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	file, err := client.CreateSpreadsheetFile(context.Background(), "Budget")
	assert.NoError(t, err)
	assert.Equal(t, "sheet-1", file.ID)
	assert.Equal(t, "Budget", body["name"])
	assert.Equal(t, spreadsheetMimeType, body["mimeType"])
	assert.NotContains(t, body, "id")
	assert.NotContains(t, body, "modifiedByMeTime")
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
	assert.Equal(t, "Summary", sheet.Title)
}

func TestGetSpreadsheetFieldsRespectGridData(t *testing.T) {
	gotFields := []string{}
	gotIncludeGridData := []string{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotFields = append(gotFields, r.URL.Query().Get("fields"))
		gotIncludeGridData = append(gotIncludeGridData, r.URL.Query().Get("includeGridData"))
		_ = json.NewEncoder(w).Encode(map[string]any{
			"sheets": []map[string]any{
				{"properties": map[string]any{"sheetId": 0, "title": "Sheet1"}},
			},
		})
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	_, err := client.GetSpreadsheet(context.Background(), "sheet-1")
	assert.NoError(t, err)
	_, err = client.GetSpreadsheetWithGridData(context.Background(), "sheet-1")
	assert.NoError(t, err)

	assert.NotContains(t, gotFields[0], "data(")
	assert.Contains(t, gotFields[1], "data(")
	assert.Equal(t, "", gotIncludeGridData[0])
	assert.Equal(t, "true", gotIncludeGridData[1])
}

func TestGetRows(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v4/spreadsheets/sheet-1/values/%27Summary%27", r.URL.EscapedPath())
		_ = json.NewEncoder(w).Encode(map[string]any{
			"values": []any{
				[]any{"name", "count"},
				[]any{"alpha", 1},
			},
		})
	}))
	defer server.Close()

	client := newTestClient(t, server.URL)
	rows, err := client.GetRows(context.Background(), "sheet-1", "Summary")
	assert.NoError(t, err)
	assert.Equal(t, Rows{{"name", "count"}, {"alpha", "1"}}, rows)
}

func newTestClient(t *testing.T, serverURL string) *Client {
	t.Helper()

	return newTestClientWithBaseURLs(t, serverURL, serverURL)
}

func newTestClientWithBaseURLs(t *testing.T, driveURL, sheetsURL string) *Client {
	t.Helper()

	httpClient := &http.Client{
		Transport: &oauth2.Transport{
			Source: oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "token"}),
			Base:   http.DefaultTransport,
		},
	}
	return &Client{
		httpClient:    httpClient,
		driveBaseURL:  driveURL,
		sheetsBaseURL: sheetsURL,
	}
}

func assertBatchRequest(t *testing.T, batch map[string]any, index int, requestName string, snippets ...string) {
	t.Helper()

	requests, ok := batch["requests"].([]any)
	assert.True(t, ok)
	if !ok || index >= len(requests) {
		t.Fatalf("batch request %d missing in %#v", index, batch)
	}
	raw, err := json.Marshal(requests[index])
	assert.NoError(t, err)
	assert.Contains(t, string(raw), requestName)
	for _, snippet := range snippets {
		assert.Contains(t, string(raw), snippet)
	}
}
