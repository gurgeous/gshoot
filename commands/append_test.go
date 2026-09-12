package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAppendCommand(t *testing.T) {
	err, stdout, stderr, log := testCommand(t, &AppendCmd{
		Spreadsheet: "Budget",
		CSVPath:     writeCSV(t, "id,amount\na,10\nb,20\n"),
	},
		`{"files":[{"id":"sheet-1","name":"Budget"}]}`,
		`{"spreadsheetId":"sheet-1","title":"Budget","sheets":[{"properties":{"sheetId":7,"title":"Data"}}]}`,
		`{"values":[["id","amount"]]}`,
		`{"updatedRange":"Data!A4:B5","updatedRows":2,"updatedColumns":2,"updatedCells":4}`,
	)
	assert.NoError(t, err)
	assert.Equal(t, "https://docs.google.com/spreadsheets/d/sheet-1/edit#gid=7", stdout)
	assert.Contains(t, stderr, "appending 2 rows")
	assert.Contains(t, log, "sheets get sheet-1 'Data'!1:1")
	assert.Contains(t, log, "sheets append sheet-1 'Data' --values-json @- --input USER_ENTERED --insert INSERT_ROWS")
}

func TestAppendCommandRequiresIdenticalColumns(t *testing.T) {
	err, _, _, log := testCommand(t, &AppendCmd{
		Spreadsheet: "Budget",
		Sheet:       "Data",
		CSVPath:     writeCSV(t, "id,amount\na,10\n"),
	},
		`{"files":[{"id":"sheet-1","name":"Budget"}]}`,
		`{"spreadsheetId":"sheet-1","title":"Budget","sheets":[{"properties":{"sheetId":7,"title":"Data"}}]}`,
		`{"values":[["amount","id"]]}`,
	)
	assert.EqualError(t, err, `csv columns must exactly match sheet "Data"`)
	assert.NotContains(t, log, "sheets append")
}

func TestAppendCommandWithOnlyHeadersDoesNotWrite(t *testing.T) {
	err, stdout, _, log := testCommand(t, &AppendCmd{
		Spreadsheet: "Budget",
		CSVPath:     writeCSV(t, "id,amount\n"),
	},
		`{"files":[{"id":"sheet-1","name":"Budget"}]}`,
		`{"spreadsheetId":"sheet-1","title":"Budget","sheets":[{"properties":{"sheetId":7,"title":"Data"}}]}`,
		`{"values":[["id","amount"]]}`,
	)
	assert.NoError(t, err)
	assert.Equal(t, "https://docs.google.com/spreadsheets/d/sheet-1/edit#gid=7", stdout)
	assert.NotContains(t, log, "sheets append")
}
