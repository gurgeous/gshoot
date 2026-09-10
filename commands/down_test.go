package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDownCommand(t *testing.T) {
	file := `{"files":[{"id":"sheet-1","name":"Budget"}]}`
	metadata := `{"spreadsheetId":"sheet-1","title":"Budget","sheets":[{"properties":{"sheetId":0,"title":"Sheet1","gridProperties":{"rowCount":10,"columnCount":5}}}]}`
	err, stdout, _, log := testCommand(t, &DownCmd{Spreadsheet: "Budget"}, file, metadata, `{"values":[["name","count"],["alpha",1]]}`)
	assert.NoError(t, err)
	assert.Equal(t, "name,count\nalpha,1", stdout)
	assert.Contains(t, log, "sheets get sheet-1 'Sheet1'")
}

func TestDownCommandMissingSpreadsheet(t *testing.T) {
	err, _, _, _ := testCommand(t, &DownCmd{Spreadsheet: "Missing Budget"}, `{"files":[]}`)
	assert.ErrorContains(t, err, "Missing Budget")
	assert.ErrorContains(t, err, "gshoot list")
}
