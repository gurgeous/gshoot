package commands

import (
	"github.com/gurgeous/gshoot/auth"
	"github.com/gurgeous/gshoot/util"
	"github.com/gurgeous/gshoot/ux"
)

// ShowAuthStatus writes a short auth status summary.
func (a *App) ShowAuthStatus(m *auth.Manager) {
	a.Println(ux.Success.Render("--- gshoot auth status ---"))

	intro := "Authenticating with Google Sheets is quite tricky. Don't blame me, I have no idea why they made it so hard!"
	if !m.HasClientSecrets() {
		intro += "\n\nFor starters, we need your *client secrets file*. When you register to use the Google Docs API, Google will give you this file. We use the client secrets file to access Google APIs and get oauth started. When you download it from google it has a crazy name like:"
		intro += "\n\n*client_secret_XXXXXXXXXXXX.apps.googleusercontent.com.json*"
		intro += "\n\nand it contains JSON like this:"
		intro += "\n\n*{\"installed\":{ <secret stuff> }}*"
		intro += "\n\nOnce you have that file from Google, import it into gshoot:\n\n**$ gshoot auth login --client-secret <client_secret_XXXXXXXXX.json>**"
	}
	a.Println()
	a.Println(ux.Markdown(intro))

	if m.HasClientSecrets() {
		a.Println()
		a.Println("Client secrets file: " + authFileStatus(m.ClientPath))
		a.Println("Token file:          " + authFileStatus(m.TokenPath))
	}

	a.Println()
	outro := "See our [Github README](" + auth.AuthReadmeURL + ") for full instructions."
	a.Println(ux.Markdown(outro))
}

// authFileStatus formats one auth file path status line.
func authFileStatus(path string) string {
	missing := !util.FileExists(path)
	var state string
	if missing {
		state = ux.Error.Render("missing")
	} else {
		state = ux.Success.Render("present")
	}
	return path + " [" + state + "]"
}
