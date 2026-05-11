package commands

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestListCommand(t *testing.T) {
	// good
	err, stdout, _ := testCommand(t, &ListCmd{}, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.URL.Path, "/drive/v3/files")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"files": []map[string]string{
				{"id": "1", "name": "Alpha", "modifiedByMeTime": "2026-05-07T12:00:00Z"},
				{"id": "2", "name": "Beta", "modifiedByMeTime": "2026-05-07T11:00:00Z"},
			},
		})
	})
	assert.NoError(t, err)
	assert.Contains(t, stdout, "Alpha")
	assert.Contains(t, stdout, "Beta")

	// bad
	err, _, _ = testCommand(t, &ListCmd{}, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", 500)
	})
	assert.Error(t, err)
}
