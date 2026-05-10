package auth

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/gurgeous/gshoot/internal/util"
	"github.com/gurgeous/gshoot/internal/ux"
)

// Status summarizes the current auth state.
type Status struct {
	ConfigDir       string
	OAuthClientPath string
	OAuthTokenPath  string
	HasOAuthClient  bool
	HasCachedToken  bool
	ResolvedSource  SourceKind
	ResolvedPath    string
	LoggedIn        bool
	ReadyForLogin   bool
}

// InspectStatus gathers auth status without network access.
func InspectStatus() Status {
	configDir := ConfigDir()
	status := Status{
		ConfigDir:       configDir,
		OAuthClientPath: filepath.Join(configDir, oauthClientFileName),
		OAuthTokenPath:  filepath.Join(configDir, oauthTokenFileName),
	}
	status.HasOAuthClient = util.FileExists(status.OAuthClientPath)
	status.HasCachedToken = util.FileExists(status.OAuthTokenPath)
	status.ReadyForLogin = status.HasOAuthClient

	resolved, err := Resolve(Options{Command: CommandList})
	if err == nil {
		status.ResolvedSource = resolved.Source.Kind
		status.ResolvedPath = resolved.Source.Path
		status.LoggedIn = true
	}

	return status
}

// PrintStatus writes a friendly auth summary.
func PrintStatus(w io.Writer, status Status) {
	fmt.Fprintln(w, ux.Subtle.Render("Config dir: "+status.ConfigDir))
	fmt.Fprintln(w, ux.Subtle.Render("OAuth client: "+presentLine(status.HasOAuthClient, status.OAuthClientPath)))
	fmt.Fprintln(w, ux.Subtle.Render("Cached token: "+presentLine(status.HasCachedToken, status.OAuthTokenPath)))

	switch {
	case status.LoggedIn:
		msg := fmt.Sprintf("Status: authenticated via %s", status.ResolvedSource)
		if status.ResolvedPath != "" {
			msg += " (" + status.ResolvedPath + ")"
		}
		fmt.Fprintln(w, ux.Success.Render(msg))
	case status.ReadyForLogin:
		fmt.Fprintln(w, ux.Warn.Render("Status: not logged in yet"))
		fmt.Fprintln(w, ux.Info.Render("Next step: run `gshoot auth login`"))
	default:
		fmt.Fprintln(w, ux.Warn.Render("Status: no auth configured"))
		fmt.Fprintln(w, ux.Info.Render("Next step: run `gshoot auth login --client-secret /path/to/client_secret.json`"))
	}
}

// Logout clears the cached OAuth session while keeping the client config.
func Logout() (bool, error) {
	tokenPath := filepath.Join(ConfigDir(), oauthTokenFileName)
	err := os.Remove(tokenPath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, fmt.Errorf("remove cached oauth token: %w", err)
}

func presentLine(ok bool, path string) string {
	if ok {
		return "present (" + path + ")"
	}
	return "missing (" + path + ")"
}
