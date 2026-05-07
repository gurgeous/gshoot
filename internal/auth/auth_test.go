package auth

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestScopesForCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cmd  Command
		want []string
	}{
		{
			name: "up",
			cmd:  CommandUp,
			want: []string{
				"https://www.googleapis.com/auth/drive",
				"https://www.googleapis.com/auth/spreadsheets",
			},
		},
		{
			name: "down",
			cmd:  CommandDown,
			want: []string{
				"https://www.googleapis.com/auth/drive.readonly",
				"https://www.googleapis.com/auth/spreadsheets.readonly",
			},
		},
		{
			name: "list",
			cmd:  CommandList,
			want: []string{
				"https://www.googleapis.com/auth/drive.readonly",
				"https://www.googleapis.com/auth/spreadsheets.readonly",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := ScopesForCommand(tt.cmd); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("ScopesForCommand() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestConfigDir(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		env  Env
		want string
	}{
		{
			name: "override",
			env: Env{
				"GSHOOT_CONFIG_DIR": "/tmp/gshoot-config",
				"HOME":              "/home/amd",
			},
			want: "/tmp/gshoot-config",
		},
		{
			name: "xdg",
			env: Env{
				"XDG_CONFIG_HOME": "/home/amd/.config-alt",
				"HOME":            "/home/amd",
			},
			want: "/home/amd/.config-alt/gshoot",
		},
		{
			name: "home default",
			env: Env{
				"HOME": "/home/amd",
			},
			want: "/home/amd/.config/gshoot",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := ConfigDir(tt.env); got != tt.want {
				t.Fatalf("ConfigDir() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveSourcePrecedence(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	explicitCreds := writeFile(t, filepath.Join(tempDir, "explicit.json"), `{"type":"service_account"}`)
	adcCreds := writeFile(t, filepath.Join(tempDir, "adc.json"), `{"type":"authorized_user"}`)
	cachedClient := writeFile(t, filepath.Join(tempDir, "oauth-client.json"), `{"installed":{"client_id":"cid","client_secret":"secret","auth_uri":"https://accounts.google.com/o/oauth2/auth","token_uri":"https://oauth2.googleapis.com/token","redirect_uris":["http://127.0.0.1"]}}`)
	cachedToken := writeFile(t, filepath.Join(tempDir, "oauth-token.json"), `{"access_token":"cached-token","refresh_token":"refresh-token","token_type":"Bearer"}`)

	tests := []struct {
		name string
		env  Env
		want SourceKind
		path string
	}{
		{
			name: "raw token wins",
			env: Env{
				"GSHOOT_TOKEN":            "token-123",
				"GSHOOT_CREDENTIALS_FILE": explicitCreds,
				"GSHOOT_CONFIG_DIR":       tempDir,
				"GOOGLE_APPLICATION_CREDENTIALS": adcCreds,
				"HOME":                    "/home/amd",
			},
			want: SourceKindRawToken,
		},
		{
			name: "credentials file env",
			env: Env{
				"GSHOOT_CREDENTIALS_FILE": explicitCreds,
				"GSHOOT_CONFIG_DIR":       tempDir,
				"GOOGLE_APPLICATION_CREDENTIALS": adcCreds,
				"HOME":                    "/home/amd",
			},
			want: SourceKindCredentialsFile,
			path: explicitCreds,
		},
		{
			name: "cached oauth before adc",
			env: Env{
				"GSHOOT_CONFIG_DIR":       tempDir,
				"GOOGLE_APPLICATION_CREDENTIALS": adcCreds,
				"HOME":                    "/home/amd",
			},
			want: SourceKindCachedOAuth,
			path: cachedToken,
		},
		{
			name: "adc env",
			env: Env{
				"GOOGLE_APPLICATION_CREDENTIALS": adcCreds,
				"HOME":                           "/home/amd",
			},
			want: SourceKindApplicationDefaultCredentials,
			path: adcCreds,
		},
	}

	_ = cachedClient

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := Resolve(Options{Env: tt.env, Command: CommandDown})
			if err != nil {
				t.Fatalf("Resolve() error = %v", err)
			}

			if got.Source.Kind != tt.want {
				t.Fatalf("Resolve() source = %q, want %q", got.Source.Kind, tt.want)
			}

			if tt.path != "" && got.Source.Path != tt.path {
				t.Fatalf("Resolve() path = %q, want %q", got.Source.Path, tt.path)
			}
		})
	}
}

func TestResolveWellKnownADC(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	adcPath := writeFile(t, filepath.Join(home, ".config", "gcloud", "application_default_credentials.json"), `{"type":"authorized_user"}`)

	got, err := Resolve(Options{
		Env:     Env{"HOME": home},
		Command: CommandList,
	})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if got.Source.Kind != SourceKindApplicationDefaultCredentials {
		t.Fatalf("Resolve() source = %q, want %q", got.Source.Kind, SourceKindApplicationDefaultCredentials)
	}

	if got.Source.Path != adcPath {
		t.Fatalf("Resolve() path = %q, want %q", got.Source.Path, adcPath)
	}
}

func TestResolveMissingAuth(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	_, err := Resolve(Options{
		Env:     Env{"HOME": home},
		Command: CommandDown,
	})
	if err == nil {
		t.Fatal("Resolve() error = nil, want error")
	}

	msg := err.Error()
	for _, want := range []string{
		"GSHOOT_TOKEN",
		"GSHOOT_CREDENTIALS_FILE",
		filepath.Join(home, ".config", "gshoot", oauthTokenFileName),
		filepath.Join(home, ".config", "gcloud", "application_default_credentials.json"),
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("error %q missing %q", msg, want)
		}
	}
}

func TestLoadCredentialFile(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	tests := []struct {
		name string
		body string
		want CredentialKind
	}{
		{
			name: "service account",
			body: `{"type":"service_account","client_email":"robot@example.com","private_key":"abc"}`,
			want: CredentialKindServiceAccount,
		},
		{
			name: "authorized user",
			body: `{"type":"authorized_user","client_id":"cid","client_secret":"secret","refresh_token":"refresh"}`,
			want: CredentialKindAuthorizedUser,
		},
		{
			name: "oauth client",
			body: `{"installed":{"client_id":"cid","client_secret":"secret","auth_uri":"https://accounts.google.com/o/oauth2/auth","token_uri":"https://oauth2.googleapis.com/token","redirect_uris":["http://127.0.0.1"]}}`,
			want: CredentialKindOAuthClient,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			path := writeFile(t, filepath.Join(tempDir, tt.name+".json"), tt.body)
			got, err := LoadCredentialFile(path)
			if err != nil {
				t.Fatalf("LoadCredentialFile() error = %v", err)
			}

			if got.Kind != tt.want {
				t.Fatalf("LoadCredentialFile() kind = %q, want %q", got.Kind, tt.want)
			}
		})
	}
}

func TestLoadOAuthToken(t *testing.T) {
	t.Parallel()

	path := writeFile(t, filepath.Join(t.TempDir(), "oauth-token.json"), `{"access_token":"a","refresh_token":"r","token_type":"Bearer","expiry":"2026-05-07T22:00:00Z"}`)
	got, err := LoadOAuthToken(path)
	if err != nil {
		t.Fatalf("LoadOAuthToken() error = %v", err)
	}

	if got.AccessToken != "a" || got.RefreshToken != "r" || got.TokenType != "Bearer" {
		t.Fatalf("LoadOAuthToken() = %#v, want parsed token", got)
	}
}

func writeFile(t *testing.T, path, body string) string {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}
