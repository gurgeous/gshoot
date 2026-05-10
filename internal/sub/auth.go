package sub

import (
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Login (or logout) from Google Sheets",
}

func init() {
	rootCmd.AddCommand(authCmd)
}
