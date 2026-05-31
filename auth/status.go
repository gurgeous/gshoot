package auth

import (
	"os"

	lipgloss "charm.land/lipgloss/v2"
	"github.com/gurgeous/gshoot/util"
	"github.com/gurgeous/gshoot/ux"
)

// ShowStatus writes a short auth status summary.
func (m *Manager) ShowStatus() {
	_, _ = lipgloss.Fprintln(os.Stdout, ux.Success.Render("--- gshoot auth status ---"))

	// calculate intro text
	intro := "Authenticating with Google Sheets is quite tricky. Don't blame me, I have no idea why they made it so hard!"
	if !m.HasClientSecrets() {
		intro += "\n\nFor starters, we need your *client secrets file*. When you register to use the Google Docs API, Google will give you this file. We use the client secrets file to access Google APIs and get oauth started. When you download it from google it has a crazy name like:"
		intro += "\n\n*client_secret_XXXXXXXXXXXX.apps.googleusercontent.com.json*"
		intro += "\n\nand it contains JSON like this:"
		intro += "\n\n*{\"installed\":{ <secret stuff> }}*"
		intro += "\n\nOnce you have that file from Google, import it into gshoot:\n\n**$ gshoot auth login --client-secret <client_secret_XXXXXXXXX.json>**"
	}
	_, _ = lipgloss.Fprintln(os.Stdout)
	_, _ = lipgloss.Fprintln(os.Stdout, ux.Markdown(intro))

	if m.HasClientSecrets() {
		_, _ = lipgloss.Fprintln(os.Stdout)
		_, _ = lipgloss.Fprintln(os.Stdout, "Client secrets file: "+missing(m.ClientPath))
		_, _ = lipgloss.Fprintln(os.Stdout, "Token file:          "+missing(m.TokenPath))
	}

	_, _ = lipgloss.Fprintln(os.Stdout)
	outro := "See our [Github README](" + AuthReadmeURL + ") for full instructions."
	_, _ = lipgloss.Fprintln(os.Stdout, ux.Markdown(outro))
}

// missing formats one status line for an auth file path.
func missing(path string) string {
	missing := !util.FileExists(path)
	var state string
	if missing {
		state = ux.Error.Render("missing")
	} else {
		state = ux.Success.Render("present")
	}
	return path + " [" + state + "]"
}
