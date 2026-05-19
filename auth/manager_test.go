package auth

import (
	"os"
	"testing"

	"github.com/adrg/xdg"
	"github.com/stretchr/testify/assert"
)

func TestNewManagerDoesNotCreateConfigDir(t *testing.T) {
	home := t.TempDir()
	t.Cleanup(xdg.Reload)
	t.Setenv("HOME", home)
	xdg.Reload()

	manager := NewManager()

	_, err := os.Stat(manager.ConfigDir)
	assert.ErrorIs(t, err, os.ErrNotExist)
}
