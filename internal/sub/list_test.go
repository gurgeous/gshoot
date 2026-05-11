package sub

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gurgeous/gshoot/internal/testutil/googletest"
	"github.com/stretchr/testify/assert"
)

//
// good
//

func TestListCommand(t *testing.T) {
	withRawTokenAuth(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Path, "/drive/v3/files"; got != want {
			t.Fatalf("path = %q, want %q", got, want)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"files": []map[string]string{
				{"id": "1", "name": "Alpha", "modifiedTime": "2026-05-07T12:00:00Z"},
				{"id": "2", "name": "Beta", "modifiedTime": "2026-05-07T11:00:00Z"},
			},
		})
	}))
	defer server.Close()
	googletest.WithGoogleAPI(t, server.URL)

	code, stdout, _ := testMain("list")
	assert.Equal(t, 0, code)
	assert.Contains(t, stdout, "Alpha")
	assert.Contains(t, stdout, "Beta")
}

//
// bad
//

func TestListCommandError(t *testing.T) {
	withRawTokenAuth(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusInternalServerError)
	}))
	defer server.Close()
	googletest.WithGoogleAPI(t, server.URL)

	code, _, stderr := testMain("list")
	assert.Equal(t, 1, code)
	assert.Contains(t, stderr, "HTTP response code 500")
}
