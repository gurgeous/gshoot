package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestListCommand(t *testing.T) {
	err, stdout, _, log := testCommand(t, &ListCmd{Limit: 5}, `{"files":[{"id":"1","name":"Alpha","modifiedByMeTime":"2026-05-07T12:00:00Z"},{"id":"2","name":"Beta","modifiedByMeTime":"2026-05-07T11:00:00Z"}]}`)
	assert.NoError(t, err)
	assert.Contains(t, stdout, "Alpha")
	assert.Contains(t, stdout, "Beta")
	assert.Contains(t, log, "drive ls --all --max 1000")
}
