package down

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gurgeous/gshoot/internal/google"
	"github.com/gurgeous/gshoot/internal/testutil/googletest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDownload(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/drive/v3/files":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"files": []map[string]string{
					{"id": "sheet-1", "name": "Budget", "modifiedTime": "2026-05-07T12:00:00Z"},
				},
			})
		case r.URL.Path == "/v4/spreadsheets/sheet-1":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"sheets": []map[string]any{
					{"properties": map[string]any{"sheetId": 0, "title": "Sheet1"}},
					{"properties": map[string]any{"sheetId": 1, "title": "Summary"}},
				},
			})
		case strings.HasPrefix(r.URL.Path, "/v4/spreadsheets/sheet-1/values/"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"values": []any{
					[]any{"month", "total"},
					[]any{"May"},
				},
			})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	})

	rows, err := Download(context.Background(), client, "budget", "")
	require.NoError(t, err)
	assert.Equal(t, [][]string{{"month", "total"}, {"May", ""}}, rows)
}

func TestDownloadSpreadsheetNotFound(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"files": []any{}})
	})

	_, err := Download(context.Background(), client, "Budget", "")
	require.Error(t, err)

	var notFound *SpreadsheetNotFoundError
	assert.True(t, errors.As(err, &notFound))
}

func TestDownloadSheetNotFound(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/drive/v3/files":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"files": []map[string]string{
					{"id": "sheet-1", "name": "Budget", "modifiedTime": "2026-05-07T12:00:00Z"},
				},
			})
		case "/v4/spreadsheets/sheet-1":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"sheets": []map[string]any{
					{"properties": map[string]any{"sheetId": 0, "title": "Sheet1"}},
				},
			})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	})

	_, err := Download(context.Background(), client, "Budget", "Summary")
	require.Error(t, err)

	var notFound *SheetNotFoundError
	assert.True(t, errors.As(err, &notFound))
	assert.Equal(t, "Budget", notFound.Spreadsheet)
}

func TestDownloadNoSheets(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/drive/v3/files":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"files": []map[string]string{
					{"id": "sheet-1", "name": "Budget", "modifiedTime": "2026-05-07T12:00:00Z"},
				},
			})
		case "/v4/spreadsheets/sheet-1":
			_ = json.NewEncoder(w).Encode(map[string]any{"sheets": []any{}})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	})

	_, err := Download(context.Background(), client, "Budget", "")
	require.Error(t, err)

	var noSheets *NoSheetsError
	assert.True(t, errors.As(err, &noSheets))
}

func newTestClient(t *testing.T, handler http.HandlerFunc) *google.Client {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return googletest.NewClient(t, server.URL)
}
