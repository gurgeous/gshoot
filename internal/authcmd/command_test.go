package authcmd

import "testing"

func TestNewCommandIncludesSubcommands(t *testing.T) {
	cmd := NewAuthCommand()
	for _, want := range []string{"login", "status", "logout"} {
		if _, _, err := cmd.Find([]string{want}); err != nil {
			t.Fatalf("Find(%q) error = %v", want, err)
		}
	}
}
