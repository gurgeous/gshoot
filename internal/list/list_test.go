package list

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gurgeous/gshoot/internal/env"
)

func TestListCommand(t *testing.T) {
	origLocal := time.Local
	time.Local = time.FixedZone("PDT", -7*60*60)
	defer func() { time.Local = origLocal }()

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

	restoreHTTP := stubGoogleAPITransport(t, server.URL)
	defer restoreHTTP()
	withListEnv(t, map[string]string{
		"GSHOOT_TOKEN": "token",
		"HOME":         t.TempDir(),
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := NewListCommand()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	out := stdout.String()
	for _, want := range []string{
		"Alpha",
		"Beta",
		"PDT",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("stdout missing %q:\n%s", want, out)
		}
	}
	if strings.Count(out, "\n") != 2 {
		t.Fatalf("stdout = %q, want 2 rows", out)
	}
	for _, want := range []string{
		"listing spreadsheets...",
		"2 recent spreadsheets",
	} {
		if !strings.Contains(stderr.String(), want) {
			t.Fatalf("stderr missing %q:\n%s", want, stderr.String())
		}
	}
}

func TestListCommandHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusInternalServerError)
	}))
	defer server.Close()

	restoreHTTP := stubGoogleAPITransport(t, server.URL)
	defer restoreHTTP()
	withListEnv(t, map[string]string{
		"GSHOOT_TOKEN": "token",
		"HOME":         t.TempDir(),
	})

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
	if !strings.Contains(err.Error(), "list spreadsheets") {
		t.Fatalf("error = %q, want list error", err.Error())
	}
	if !strings.Contains(stderr.String(), "list failed") {
		t.Fatalf("stderr = %q, want failure status", stderr.String())
	}
}

func TestListCommandAuthError(t *testing.T) {
	withListEnv(t, map[string]string{"HOME": t.TempDir()})

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

func stubGoogleAPITransport(t *testing.T, serverURL string) func() {
	t.Helper()

	serverBase, err := url.Parse(serverURL)
	if err != nil {
		t.Fatalf("Parse(serverURL) error = %v", err)
	}

	orig := http.DefaultTransport
	http.DefaultTransport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		cloned := req.Clone(req.Context())
		cloned.URL = cloneURL(req.URL)
		cloned.URL.Scheme = serverBase.Scheme
		cloned.URL.Host = serverBase.Host
		cloned.Host = serverBase.Host
		return orig.RoundTrip(cloned)
	})

	return func() {
		http.DefaultTransport = orig
	}
}

func withListEnv(t *testing.T, overrides map[string]string) {
	t.Helper()

	vars := map[string]*string{
		"GOOGLE_APPLICATION_CREDENTIALS": &env.GOOGLE_APPLICATION_CREDENTIALS,
		"GSHOOT_CONFIG_DIR":              &env.GSHOOT_CONFIG_DIR,
		"GSHOOT_CREDENTIALS_FILE":        &env.GSHOOT_CREDENTIALS_FILE,
		"GSHOOT_THEME":                   &env.GSHOOT_THEME,
		"GSHOOT_TOKEN":                   &env.GSHOOT_TOKEN,
	}

	old := make(map[string]string, len(vars))
	oldSet := make(map[string]bool, len(vars))
	for name, ptr := range vars {
		old[name] = *ptr
		_, oldSet[name] = os.LookupEnv(name)
		reflect.ValueOf(ptr).Elem().SetString("")
		if err := os.Unsetenv(name); err != nil {
			t.Fatalf("Unsetenv(%s) error = %v", name, err)
		}
	}

	for name, value := range overrides {
		if ptr, ok := vars[name]; ok {
			reflect.ValueOf(ptr).Elem().SetString(value)
		}
		if err := os.Setenv(name, value); err != nil {
			t.Fatalf("Setenv(%s) error = %v", name, err)
		}
	}

	t.Cleanup(func() {
		for name, value := range old {
			reflect.ValueOf(vars[name]).Elem().SetString(value)
			if oldSet[name] {
				if err := os.Setenv(name, value); err != nil {
					t.Fatalf("restore env %s: %v", name, err)
				}
				continue
			}
			if err := os.Unsetenv(name); err != nil {
				t.Fatalf("unset env %s: %v", name, err)
			}
		}
	})
}

func cloneURL(u *url.URL) *url.URL {
	cloned := *u
	return &cloned
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}
