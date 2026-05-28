package commands

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDownCommand(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/drive/v3/files":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"files": []map[string]string{
					{"id": "sheet-1", "name": "Budget", "modifiedByMeTime": "2026-05-07T12:00:00Z"},
				},
			})
		case r.URL.Path == "/sheets/v4/spreadsheets/sheet-1":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"sheets": []map[string]any{
					{"properties": map[string]any{"sheetId": 0, "title": "Sheet1"}},
				},
			})
		case strings.HasPrefix(r.URL.Path, "/sheets/v4/spreadsheets/sheet-1/values/"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"values": [][]string{{"name", "count"}, {"alpha", "1"}},
			})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}

	err, stdout, _ := testCommand(t, &DownCmd{Spreadsheet: "Budget"}, handler)
	assert.NoError(t, err)
	assert.Equal(t, "name,count\nalpha,1\n", stdout)

	path := filepath.Join(t.TempDir(), "out.csv")
	err, stdout, _ = testCommand(t, &DownCmd{Spreadsheet: "Budget", Output: path}, handler)
	assert.NoError(t, err)
	assert.Equal(t, "", stdout)
	data, _ := os.ReadFile(path)
	assert.Equal(t, "name,count\nalpha,1\n", string(data))
}
