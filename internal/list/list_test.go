package list

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gurgeous/gshoot/internal/testutil"
	"google.golang.org/api/drive/v3"
)

func TestRecent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.Header.Get("Authorization"), "Bearer token"; got != want {
			t.Fatalf("Authorization = %q, want %q", got, want)
		}
		if got, want := r.URL.Path, "/drive/v3/files"; got != want {
			t.Fatalf("path = %q, want %q", got, want)
		}
		if got, want := r.URL.Query().Get("pageSize"), "10"; got != want {
			t.Fatalf("pageSize = %q, want %q", got, want)
		}
		if got, want := r.URL.Query().Get("q"), "mimeType='application/vnd.google-apps.spreadsheet' and trashed=false"; got != want {
			t.Fatalf("q = %q, want %q", got, want)
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

	client := testutil.NewDriveTestClient(t, server.URL)
	files, err := recent(context.Background(), client, 10)
	if err != nil {
		t.Fatalf("recent() error = %v", err)
	}
	if len(files) != 2 || files[0].Name != "Alpha" || files[1].Name != "Beta" {
		t.Fatalf("recent() = %#v, want Alpha/Beta", files)
	}
}

func TestRecentHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusInternalServerError)
	}))
	defer server.Close()

	_, err := recent(context.Background(), testutil.NewDriveTestClient(t, server.URL), 10)
	if err == nil {
		t.Fatal("recent() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "list spreadsheets") {
		t.Fatalf("recent() error = %q, want list error", err.Error())
	}
}

func TestPrintFiles(t *testing.T) {
	origLocal := time.Local
	time.Local = time.FixedZone("PDT", -7*60*60)
	defer func() { time.Local = origLocal }()

	var out bytes.Buffer
	printFiles(&out, []*drive.File{
		{Id: "1", Name: "Alpha", ModifiedTime: "2026-05-07T12:00:00Z"},
		{Id: "2", Name: "Beta", ModifiedTime: "2026-05-07T11:00:00Z"},
	})

	got := out.String()
	for _, want := range []string{"Alpha", "Beta", "PDT"} {
		if !strings.Contains(got, want) {
			t.Fatalf("printFiles() output missing %q:\n%s", want, got)
		}
	}
	if strings.Count(got, "\n") != 2 {
		t.Fatalf("printFiles() = %q, want 2 rows", got)
	}
}
