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

	"github.com/gurgeous/gshoot/internal/env"
	"github.com/gurgeous/gshoot/internal/google"
	"github.com/gurgeous/gshoot/internal/testutil"
	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
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
		if got, want := r.URL.Query().Get("orderBy"), "modifiedTime desc,name"; got != want {
			t.Fatalf("orderBy = %q, want %q", got, want)
		}
		if got, want := r.URL.Query().Get("fields"), "files(id,name,modifiedTime)"; got != want {
			t.Fatalf("fields = %q, want %q", got, want)
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

	client := googleClient(t, server.URL)
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

	_, err := recent(context.Background(), googleClient(t, server.URL), 10)
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

func TestListCommandAuthError(t *testing.T) {
	testutil.WithEnv(t, map[string]string{"HOME": t.TempDir()}, envVars())

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := NewListCommand()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want error")
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(err.Error(), "no auth found") {
		t.Fatalf("error = %q, want auth guidance", err.Error())
	}
}

func envVars() map[string]*string {
	return map[string]*string{
		"GOOGLE_APPLICATION_CREDENTIALS": &env.GOOGLE_APPLICATION_CREDENTIALS,
		"GSHOOT_CONFIG_DIR":              &env.GSHOOT_CONFIG_DIR,
		"GSHOOT_CREDENTIALS_FILE":        &env.GSHOOT_CREDENTIALS_FILE,
		"GSHOOT_THEME":                   &env.GSHOOT_THEME,
		"GSHOOT_TOKEN":                   &env.GSHOOT_TOKEN,
	}
}

func googleClient(t *testing.T, serverURL string) *google.Client {
	t.Helper()

	httpClient := &http.Client{
		Transport: &oauth2.Transport{
			Source: oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "token"}),
			Base:   http.DefaultTransport,
		},
	}
	driveService, err := drive.NewService(
		context.Background(),
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(serverURL+"/drive/v3/"),
	)
	if err != nil {
		t.Fatalf("drive.NewService() error = %v", err)
	}
	return &google.Client{Drive: driveService}
}
