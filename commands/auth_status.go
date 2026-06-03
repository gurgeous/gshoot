package commands

import (
	"fmt"

	"github.com/gurgeous/gshoot/auth"
	"github.com/gurgeous/gshoot/util"
	"github.com/gurgeous/gshoot/ux"
)

//
// Human-readable auth status. We trigger this from several places
//

func ShowAuthStatus() error {
	m, err := auth.NewManager()
	if err != nil {
		return err
	}

	fmt.Println(ux.Success.Render("--- gshoot auth status ---"))

	fmt.Println()
	fmt.Println("Client secrets file: " + authFileStatus(m.ClientPath))
	fmt.Println("Token file:          " + authFileStatus(m.TokenPath))

	if !m.LoggedIn() {
		intro := "Authenticating with Google Sheets is quite tricky. Don't blame me, I have no idea why they made it so hard!"
		if !m.HasClientSecrets() {
			intro += "\n\nFor starters, we need your *client secrets file*. When you register to use the Google Sheets API, Google will give you this file. We use the client secrets file to access Google APIs and get OAuth started. When you download it from Google, it has a crazy name like:"
			intro += "\n\n*client_secret_XXXXXXXXXXXX.apps.googleusercontent.com.json*"
			intro += "\n\nand it contains JSON like this:"
			intro += "\n\n*{\"installed\":{ <secret stuff> }}*"
			intro += "\n\nOnce you have that file from Google, import it into gshoot:\n\n**$ gshoot auth login --client-secret <client_secret_XXXXXXXXX.json>**"
		}
		fmt.Println()
		fmt.Println(ux.Markdown(intro))
	}

	fmt.Println()
	if !m.LoggedIn() {
		outro := "See our [GitHub README](" + auth.AuthReadmeURL + ") for full instructions."
		fmt.Println(ux.Markdown(outro))
	} else {
		fmt.Println("You are logged in and gshoot commands should work.")
	}

	return nil
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
