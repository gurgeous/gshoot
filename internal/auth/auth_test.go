package auth

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/adrg/xdg"
)

func TestScopesForCommand(t *testing.T) {
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
			if got := ScopesForCommand(tt.cmd); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("ScopesForCommand() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestScopesForCommandUnknown(t *testing.T) {
	if got := ScopesForCommand(Command("wat")); got != nil {
		t.Fatalf("ScopesForCommand() = %#v, want nil", got)
	}
}

func TestConfigDir(t *testing.T) {
	home := t.TempDir()
	tests := []struct {
		name string
		env  map[string]string
		want string
	}{
		{
			name: "override",
			env:  map[string]string{"GSHOOT_CONFIG_DIR": "/tmp/gshoot-config", "HOME": home},
			want: "/tmp/gshoot-config",
		},
		{
			name: "xdg",
			env:  map[string]string{"HOME": home},
			want: "/tmp/xdg/gshoot",
		},
		{
			name: "home default",
			env:  map[string]string{"HOME": home},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTestEnv(t, tt.env)
			if tt.name == "xdg" {
				if err := os.Setenv("XDG_CONFIG_HOME", "/tmp/xdg"); err != nil {
					t.Fatalf("Setenv(XDG_CONFIG_HOME): %v", err)
				}
				xdg.Reload()
				t.Cleanup(func() {
					if err := os.Unsetenv("XDG_CONFIG_HOME"); err != nil {
						t.Fatalf("Unsetenv(XDG_CONFIG_HOME): %v", err)
					}
					xdg.Reload()
				})
			}
			want := tt.want
			if tt.name == "home default" {
				want = filepath.Join(xdg.ConfigHome, "gshoot")
			}
			if got := ConfigDir(); got != want {
				t.Fatalf("ConfigDir() = %q, want %q", got, want)
			}
		})
	}
}

func TestResolveSourcePrecedence(t *testing.T) {
	tempDir := t.TempDir()
	home := filepath.Join(tempDir, "home")
	explicitCreds := writeFile(t, filepath.Join(tempDir, "explicit.json"), `{"type":"service_account"}`)
	adcCreds := writeFile(t, filepath.Join(tempDir, "adc.json"), `{"type":"authorized_user","client_id":"adc","client_secret":"secret","refresh_token":"refresh"}`)
	cachedToken := writeFile(t, filepath.Join(tempDir, "oauth-token.json"), `{"access_token":"cached-token","refresh_token":"refresh-token","token_type":"Bearer","expiry":"2026-05-07T22:00:00Z"}`)

	tests := []struct {
		name string
		env  map[string]string
		want SourceKind
		path string
	}{
		{
			name: "raw token wins",
			env: map[string]string{
				"GSHOOT_TOKEN":                   "token-123",
				"GSHOOT_CREDENTIALS_FILE":        explicitCreds,
				"GSHOOT_CONFIG_DIR":              tempDir,
				"GOOGLE_APPLICATION_CREDENTIALS": adcCreds,
				"HOME":                           home,
			},
			want: SourceKindRawToken,
		},
		{
			name: "credentials file env",
			env: map[string]string{
				"GSHOOT_CREDENTIALS_FILE":        explicitCreds,
				"GSHOOT_CONFIG_DIR":              tempDir,
				"GOOGLE_APPLICATION_CREDENTIALS": adcCreds,
				"HOME":                           home,
			},
			want: SourceKindCredentialsFile,
			path: explicitCreds,
		},
		{
			name: "cached oauth before adc",
			env: map[string]string{
				"GSHOOT_CONFIG_DIR":              tempDir,
				"GOOGLE_APPLICATION_CREDENTIALS": adcCreds,
				"HOME":                           home,
			},
			want: SourceKindCachedOAuth,
			path: cachedToken,
		},
		{
			name: "adc env",
			env: map[string]string{
				"GOOGLE_APPLICATION_CREDENTIALS": adcCreds,
				"HOME":                           home,
			},
			want: SourceKindApplicationDefaultCredentials,
			path: adcCreds,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withTestEnv(t, tt.env)
			got, err := Resolve(Options{Command: CommandDown})
			if err != nil {
				t.Fatalf("Resolve() error = %v", err)
			}

			if got.Source.Kind != tt.want {
				t.Fatalf("Resolve() source = %q, want %q", got.Source.Kind, tt.want)
			}

			if tt.path != "" && got.Source.Path != tt.path {
				t.Fatalf("Resolve() path = %q, want %q", got.Source.Path, tt.path)
			}

			if tt.want == SourceKindRawToken && got.Source.Token != "token-123" {
				t.Fatalf("Resolve() token = %q, want raw token", got.Source.Token)
			}

			if got.OAuthClientPath != filepath.Join(ConfigDir(), oauthClientFileName) {
				t.Fatalf("Resolve() OAuthClientPath = %q, want derived path", got.OAuthClientPath)
			}

			if got.OAuthTokenPath != filepath.Join(ConfigDir(), oauthTokenFileName) {
				t.Fatalf("Resolve() OAuthTokenPath = %q, want derived path", got.OAuthTokenPath)
			}
		})
	}
}

func TestResolveWellKnownADC(t *testing.T) {
	home := t.TempDir()
	adcPath := writeFile(t, filepath.Join(home, ".config", "gcloud", "application_default_credentials.json"), `{"type":"authorized_user","client_id":"adc","client_secret":"secret","refresh_token":"refresh"}`)

	withTestEnv(t, map[string]string{"HOME": home})
	got, err := Resolve(Options{Command: CommandList})
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
	home := t.TempDir()
	withTestEnv(t, map[string]string{"HOME": home})
	_, err := Resolve(Options{Command: CommandDown})
	if err == nil {
		t.Fatal("Resolve() error = nil, want error")
	}

	msg := err.Error()
	for _, want := range []string{
		"$GSHOOT_TOKEN",
		"$GSHOOT_CREDENTIALS_FILE",
		"$GOOGLE_APPLICATION_CREDENTIALS",
		filepath.Join(ConfigDir(), oauthTokenFileName),
		filepath.Join(home, ".config", "gcloud", "application_default_credentials.json"),
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("error %q missing %q", msg, want)
		}
	}
}

func TestResolveExplicitCredentialFileMissing(t *testing.T) {
	withTestEnv(t, map[string]string{"GSHOOT_CREDENTIALS_FILE": "/does/not/exist.json", "HOME": t.TempDir()})
	_, err := Resolve(Options{Command: CommandDown})
	if err == nil {
		t.Fatal("Resolve() error = nil, want error")
	}

	if !strings.Contains(err.Error(), "GSHOOT_CREDENTIALS_FILE") || !strings.Contains(err.Error(), "no such file") {
		t.Fatalf("Resolve() error = %q, want explicit env file failure", err.Error())
	}
}

func TestResolveApplicationCredentialsMissing(t *testing.T) {
	withTestEnv(t, map[string]string{"GOOGLE_APPLICATION_CREDENTIALS": "/does/not/exist.json", "HOME": t.TempDir()})
	_, err := Resolve(Options{Command: CommandList})
	if err == nil {
		t.Fatal("Resolve() error = nil, want error")
	}

	if !strings.Contains(err.Error(), "GOOGLE_APPLICATION_CREDENTIALS") || !strings.Contains(err.Error(), "no such file") {
		t.Fatalf("Resolve() error = %q, want explicit ADC env failure", err.Error())
	}
}

func TestResolveExplicitCredentialFileCorrupt(t *testing.T) {
	path := writeFile(t, filepath.Join(t.TempDir(), "bad.json"), `{"type":`)
	withTestEnv(t, map[string]string{"GSHOOT_CREDENTIALS_FILE": path, "HOME": t.TempDir()})
	_, err := Resolve(Options{Command: CommandDown})
	if err == nil {
		t.Fatal("Resolve() error = nil, want error")
	}

	if !strings.Contains(err.Error(), "GSHOOT_CREDENTIALS_FILE") {
		t.Fatalf("Resolve() error = %q, want explicit env context", err.Error())
	}
}

func TestResolveWellKnownADCCorrupt(t *testing.T) {
	home := t.TempDir()
	writeFile(t, filepath.Join(home, ".config", "gcloud", "application_default_credentials.json"), `{"type":`)

	withTestEnv(t, map[string]string{"HOME": home})
	_, err := Resolve(Options{Command: CommandList})
	if err == nil {
		t.Fatal("Resolve() error = nil, want error")
	}

	if !strings.Contains(err.Error(), "application default credentials") {
		t.Fatalf("Resolve() error = %q, want ADC context", err.Error())
	}
}

func TestResolveCachedOAuthInvalid(t *testing.T) {
	configDir := t.TempDir()
	writeFile(t, filepath.Join(configDir, oauthTokenFileName), `{"access_token":`)

	withTestEnv(t, map[string]string{"GSHOOT_CONFIG_DIR": configDir, "HOME": t.TempDir()})
	_, err := Resolve(Options{Command: CommandDown})
	if err == nil {
		t.Fatal("Resolve() error = nil, want error")
	}

	if !strings.Contains(err.Error(), "cached oauth token") {
		t.Fatalf("Resolve() error = %q, want cached token context", err.Error())
	}
}

func TestResolveUnknownCommand(t *testing.T) {
	withTestEnv(t, map[string]string{"HOME": t.TempDir()})
	_, err := Resolve(Options{Command: Command("wat")})
	if err == nil {
		t.Fatal("Resolve() error = nil, want error")
	}

	if !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("Resolve() error = %q, want unknown command", err.Error())
	}
}

func TestLoadCredentialFile(t *testing.T) {
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
		{
			name: "oauth client web",
			body: `{"web":{"client_id":"cid","client_secret":"secret","auth_uri":"https://accounts.google.com/o/oauth2/auth","token_uri":"https://oauth2.googleapis.com/token","redirect_uris":["http://127.0.0.1"]}}`,
			want: CredentialKindOAuthClient,
		},
		{
			name: "installed wins over type",
			body: `{"type":"authorized_user","installed":{"client_id":"cid","client_secret":"secret","auth_uri":"https://accounts.google.com/o/oauth2/auth","token_uri":"https://oauth2.googleapis.com/token","redirect_uris":["http://127.0.0.1"]}}`,
			want: CredentialKindOAuthClient,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

func TestLoadCredentialFileUnsupported(t *testing.T) {
	tests := []string{
		`{"type":"external_account"}`,
		`{}`,
	}

	for _, body := range tests {
		path := writeFile(t, filepath.Join(t.TempDir(), "unsupported.json"), body)
		_, err := LoadCredentialFile(path)
		if err == nil {
			t.Fatal("LoadCredentialFile() error = nil, want error")
		}
		if !strings.Contains(err.Error(), "unsupported credential file") {
			t.Fatalf("LoadCredentialFile() error = %q, want unsupported error", err.Error())
		}
	}
}

func TestLoadOAuthToken(t *testing.T) {
	path := writeFile(t, filepath.Join(t.TempDir(), "oauth-token.json"), `{"access_token":"a","refresh_token":"r","token_type":"Bearer","expiry":"2026-05-07T22:00:00Z"}`)
	got, err := LoadOAuthToken(path)
	if err != nil {
		t.Fatalf("LoadOAuthToken() error = %v", err)
	}

	if got.AccessToken != "a" || got.RefreshToken != "r" || got.TokenType != "Bearer" {
		t.Fatalf("LoadOAuthToken() = %#v, want parsed token", got)
	}
	if got.Expiry != time.Date(2026, 5, 7, 22, 0, 0, 0, time.UTC) {
		t.Fatalf("LoadOAuthToken() expiry = %v, want parsed time", got.Expiry)
	}
}

func TestLoadOAuthTokenInvalidJSON(t *testing.T) {
	path := writeFile(t, filepath.Join(t.TempDir(), "oauth-token.json"), `{"access_token":`)
	_, err := LoadOAuthToken(path)
	if err == nil {
		t.Fatal("LoadOAuthToken() error = nil, want error")
	}
}

func TestNewTokenSourceExpiredCachedOAuthWithoutClientConfig(t *testing.T) {
	resolved := Resolved{
		Scopes:          ScopesForCommand(CommandList),
		OAuthClientPath: filepath.Join(t.TempDir(), "oauth-client.json"),
		Source: Source{
			Kind: SourceKindCachedOAuth,
			OAuthToken: &OAuthToken{
				AccessToken:  "expired",
				RefreshToken: "refresh",
				TokenType:    "Bearer",
				Expiry:       time.Now().Add(-time.Hour),
			},
		},
	}

	_, err := NewTokenSource(context.Background(), resolved)
	if err == nil {
		t.Fatal("NewTokenSource() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "expired") || !strings.Contains(err.Error(), "oauth-client.json") {
		t.Fatalf("NewTokenSource() error = %q, want actionable error", err.Error())
	}
}

func TestNewTokenSourceValidCachedOAuthWithoutClientConfig(t *testing.T) {
	resolved := Resolved{
		Scopes:          ScopesForCommand(CommandList),
		OAuthClientPath: filepath.Join(t.TempDir(), "oauth-client.json"),
		Source: Source{
			Kind: SourceKindCachedOAuth,
			OAuthToken: &OAuthToken{
				AccessToken: "valid",
				TokenType:   "Bearer",
				Expiry:      time.Now().Add(time.Hour),
			},
		},
	}

	src, err := NewTokenSource(context.Background(), resolved)
	if err != nil {
		t.Fatalf("NewTokenSource() error = %v", err)
	}

	token, err := src.Token()
	if err != nil {
		t.Fatalf("Token() error = %v", err)
	}
	if token.AccessToken != "valid" {
		t.Fatalf("Token() access token = %q, want valid", token.AccessToken)
	}
}

func TestLoadMissingFiles(t *testing.T) {
	_, err := LoadCredentialFile("/does/not/exist.json")
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("LoadCredentialFile() error = %v, want os.ErrNotExist", err)
	}

	_, err = LoadOAuthToken("/does/not/exist.json")
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("LoadOAuthToken() error = %v, want os.ErrNotExist", err)
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
