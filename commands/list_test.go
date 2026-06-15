package commands

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestListCommand(t *testing.T) {
	var gotPageSize string

	// good
	err, stdout, _ := testCommand(t, &ListCmd{}, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.URL.Path, "/drive/v3/files")
		gotPageSize = r.URL.Query().Get("pageSize")
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
	assert.Equal(t, "1", gotPageSize)

	err, _, _ = testCommand(t, &ListCmd{Limit: 5}, func(w http.ResponseWriter, r *http.Request) {
		gotPageSize = r.URL.Query().Get("pageSize")
		_ = json.NewEncoder(w).Encode(map[string]any{"files": []any{}})
	})
	assert.NoError(t, err)
	assert.Equal(t, "5", gotPageSize)

	err, _, _ = testCommand(t, &ListCmd{Limit: 101}, func(w http.ResponseWriter, r *http.Request) {
		gotPageSize = r.URL.Query().Get("pageSize")
		_ = json.NewEncoder(w).Encode(map[string]any{"files": []any{}})
	})
	assert.NoError(t, err)
	assert.Equal(t, "100", gotPageSize)

	// bad
	err, _, _ = testCommand(t, &ListCmd{}, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", 500)
	})
	assert.Error(t, err)
}

func TestListCommandExpiredLogin(t *testing.T) {
	err, _, _ := testCommandWithSetup(t, &ListCmd{}, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error":             "invalid_grant",
			"error_description": "Token has been expired or revoked.",
		})
	}, func(home string) {
		writeAuthFiles(t, home, authFilesOptions{
			HasClient: true,
			HasToken:  true,
			Expiry:    time.Now().Add(-time.Hour),
		})
	})

	assert.Error(t, err)
	if err != nil {
		assert.Contains(t, err.Error(), "Your Google login has expired")
		assert.Contains(t, err.Error(), "gshoot auth login")
		assert.NotContains(t, err.Error(), "invalid_grant")
	}
}
