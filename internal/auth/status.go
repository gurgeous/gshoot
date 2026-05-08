package auth

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
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
func InspectStatus(env Env) Status {
	configDir := ConfigDir(env)
	status := Status{
		ConfigDir:       configDir,
		OAuthClientPath: filepath.Join(configDir, oauthClientFileName),
		OAuthTokenPath:  filepath.Join(configDir, oauthTokenFileName),
	}
	status.HasOAuthClient = fileExists(status.OAuthClientPath)
	status.HasCachedToken = fileExists(status.OAuthTokenPath)
	status.ReadyForLogin = status.HasOAuthClient

	resolved, err := Resolve(Options{Env: env, Command: CommandList})
	if err == nil {
		status.ResolvedSource = resolved.Source.Kind
		status.ResolvedPath = resolved.Source.Path
		status.LoggedIn = true
	}

	return status
}

// PrintStatus writes a friendly auth summary.
func PrintStatus(w io.Writer, status Status) {
	fmt.Fprintf(w, "Config dir: %s\n", status.ConfigDir)
	fmt.Fprintf(w, "OAuth client: %s\n", presentLine(status.HasOAuthClient, status.OAuthClientPath))
	fmt.Fprintf(w, "Cached token: %s\n", presentLine(status.HasCachedToken, status.OAuthTokenPath))

	switch {
	case status.LoggedIn:
		fmt.Fprintf(w, "Status: authenticated via %s", status.ResolvedSource)
		if status.ResolvedPath != "" {
			fmt.Fprintf(w, " (%s)", status.ResolvedPath)
		}
		fmt.Fprintln(w)
	case status.ReadyForLogin:
		fmt.Fprintln(w, "Status: not logged in yet")
		fmt.Fprintln(w, "Next step: run `gshoot auth login`")
	default:
		fmt.Fprintln(w, "Status: no auth configured")
		fmt.Fprintln(w, "Next step: run `gshoot auth login --client-secret /path/to/client_secret.json`")
	}
}

// Logout clears the cached OAuth session while keeping the client config.
func Logout(env Env) (bool, error) {
	tokenPath := filepath.Join(ConfigDir(env), oauthTokenFileName)
	err := os.Remove(tokenPath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, fmt.Errorf("remove cached oauth token: %w", err)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func presentLine(ok bool, path string) string {
	if ok {
		return "present (" + path + ")"
	}
	return "missing (" + path + ")"
}
