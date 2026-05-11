package sub

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gurgeous/gshoot/internal/testutil/googletest"
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

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	_ = Main([]string{"list"}, &stdout, &stderr)
	got := stdout.String()
	for _, want := range []string{"Alpha", "Beta"} {
		if !strings.Contains(got, want) {
			t.Fatalf("stdout missing %q:\n%s", want, got)
		}
	}
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

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Main([]string{"list"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("Main() code = %d, want 1", code)
	}
}
