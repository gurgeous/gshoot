package sub

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDownCommandStdout(t *testing.T) {
	withRawTokenAuth(t)
	withAPI(t, newDownAPIHandler(t))

	code, stdout, stderr := testMain("down", "Budget")
	require.Equal(t, 0, code)
	assert.Equal(t, "name,count\nalpha,1\n", stdout)
	assert.Contains(t, stderr, "fetching")
}

func TestDownCommandOutputFile(t *testing.T) {
	withRawTokenAuth(t)
	withAPI(t, newDownAPIHandler(t))

	path := filepath.Join(t.TempDir(), "out.csv")
	code, stdout, stderr := testMain("down", "Budget", "--output", path)
	require.Equal(t, 0, code)
	assert.Equal(t, "", stdout)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "name,count\nalpha,1\n", string(data))
	assert.Contains(t, stderr, "saving")
}

func newDownAPIHandler(t *testing.T) http.HandlerFunc {
	t.Helper()

	return func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/drive/v3/files":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"files": []map[string]string{
					{"id": "sheet-1", "name": "Budget", "modifiedTime": "2026-05-07T12:00:00Z"},
				},
			})
		case r.URL.Path == "/v4/spreadsheets/sheet-1":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"sheets": []map[string]any{
					{"properties": map[string]any{"sheetId": 0, "title": "Sheet1"}},
				},
			})
		case strings.HasPrefix(r.URL.Path, "/v4/spreadsheets/sheet-1/values/"):
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"values": [][]string{{"name", "count"}, {"alpha", "1"}},
			})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}
}
