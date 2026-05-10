package list

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gurgeous/gshoot/internal/testutil/googletest"
)

func TestRecent(t *testing.T) {
	// stub
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

	// run
	client := googletest.NewDriveClient(t, server.URL)
	files, err := recent(context.Background(), client, 10)
	if err != nil {
		t.Fatalf("recent() error = %v", err)
	}
	if files[0].Name != "Alpha" || files[1].Name != "Beta" {
		t.Fatalf("recent() = %#v, want Alpha/Beta", files)
	}

	// print
	var out bytes.Buffer
	printFiles(&out, files)
	got := out.String()
	for _, want := range []string{"Alpha", "Beta"} {
		if !strings.Contains(got, want) {
			t.Fatalf("printFiles() output missing %q:\n%s", want, got)
		}
	}
}

func TestRecentError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusInternalServerError)
	}))
	defer server.Close()

	_, err := recent(context.Background(), googletest.NewDriveClient(t, server.URL), 10)
	if err == nil {
		t.Fatal("recent() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "list spreadsheets") {
		t.Fatalf("recent() error = %q, want list error", err.Error())
	}
}
