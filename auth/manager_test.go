package auth

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/adrg/xdg"
	"github.com/stretchr/testify/assert"
)

func TestNewManagerDoesNotCreateConfigDir(t *testing.T) {
	home := t.TempDir()
	t.Cleanup(xdg.Reload)
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	xdg.Reload()

	manager := NewManager()

	_, err := os.Stat(manager.ConfigDir)
	assert.ErrorIs(t, err, os.ErrNotExist)
}
