package commands

import (
	"strings"
	"testing"

	"github.com/gurgeous/gshoot/gog"
	"github.com/stretchr/testify/assert"
)

func TestUpCommandRejectsInvalidCSV(t *testing.T) {
	assert.Error(t, validateHeaders([]string{"", ""}, "csv"))
	assert.Error(t, validateHeaders([]string{"id", "id"}, "csv"))
}

func TestUpCommandRejectsConflictingModes(t *testing.T) {
	err := (&UpCmd{Refill: true, Replace: true}).Run()
	assert.EqualError(t, err, "use either --refill or --replace")
}

func TestUpCommandRejectsEmptyCSV(t *testing.T) {
	err, _, _, _ := testCommand(t, &UpCmd{Spreadsheet: "Budget", CSVPath: writeCSV(t, "")})
	assert.ErrorContains(t, err, "csv is empty")
}

func TestUpCommandReplaceClearsAndPastes(t *testing.T) {
	metadata := `{"spreadsheetId":"sheet-1","title":"Budget","sheets":[{"properties":{"sheetId":7,"title":"Data","gridProperties":{"rowCount":4,"columnCount":4}}}]}`
	err, stdout, _, log := testCommand(t, &UpCmd{
		Spreadsheet: "Budget",
		Sheet:       "Data",
		CSVPath:     writeCSV(t, "name,count\nalpha,1\n"),
		Replace:     true,
	},
		`{"files":[{"id":"sheet-1","name":"Budget"}]}`,
		metadata,
		metadata, `{}`, `{}`, `{}`, `{}`,
		metadata,
		metadata, `{}`,
	)
	assert.NoError(t, err)
	assert.Equal(t, "https://docs.google.com/spreadsheets/d/sheet-1/edit", stdout)
	assert.Contains(t, log, "sheets clear sheet-1 'Data'!A1:D4")
	assert.Contains(t, log, "sheets update sheet-1 'Data'!A1 --values-json @- --input USER_ENTERED")
	assert.Equal(t, 1, strings.Count(log, "sheets update sheet-1 'Data'!A1"))
}

func TestUpCommandCreatesSpreadsheet(t *testing.T) {
	sheet1 := `{"spreadsheetId":"sheet-new","title":"New Budget","sheets":[{"properties":{"sheetId":0,"title":"Sheet1","gridProperties":{"rowCount":4,"columnCount":4}}}]}`
	input := `{"spreadsheetId":"sheet-new","title":"New Budget","sheets":[{"properties":{"sheetId":0,"title":"input","gridProperties":{"rowCount":4,"columnCount":4}}}]}`
	err, stdout, _, log := testCommand(t, &UpCmd{
		Spreadsheet: "New Budget",
		CSVPath:     writeCSV(t, "name,count\nalpha,1\n"),
	},
		`{"files":[]}`,
		`{"spreadsheetId":"sheet-new","title":"New Budget"}`,
		sheet1,
		`{"values":[]}`,
		sheet1, `{}`,
		input,
		input, `{}`,
	)
	assert.NoError(t, err)
	assert.Equal(t, "https://docs.google.com/spreadsheets/d/sheet-new/edit", stdout)
	assert.Contains(t, log, "sheets create New Budget")
	assert.Contains(t, log, "sheets rename-tab sheet-new Sheet1 input")
}

func TestUpCommandRefillPreservesFormulas(t *testing.T) {
	metadata := `{"spreadsheetId":"sheet-1","title":"Budget","sheets":[{"properties":{"sheetId":7,"title":"Data","gridProperties":{"rowCount":4,"columnCount":5}}}]}`
	raw := `{"spreadsheetId":"sheet-1","title":"Budget","sheets":[{"properties":{"sheetId":7,"title":"Data","gridProperties":{"rowCount":4,"columnCount":5}},"data":[{"rowData":[{"values":[{"userEnteredValue":{"stringValue":"id"}},{"userEnteredValue":{"stringValue":"calc"}}]},{"values":[{"userEnteredValue":{"stringValue":"a"}},{"userEnteredValue":{"formulaValue":"=A2"}}]}]}]}]}`
	err, _, _, log := testCommand(t, &UpCmd{
		Spreadsheet: "Budget",
		Sheet:       "Data",
		CSVPath:     writeCSV(t, "id,name\na,Ada\n"),
		Refill:      true,
	},
		`{"files":[{"id":"sheet-1","name":"Budget"}]}`,
		metadata,
		`{"values":[["id","calc"],["a","1"]]}`,
		raw,
		metadata,
		metadata, `{}`,
		metadata, `{}`,
		metadata, `{}`,
	)
	assert.NoError(t, err)
	assert.Contains(t, log, "sheets update sheet-1 'Data'!A1 --values-json @- --input USER_ENTERED")
}

func TestUpCommandLayoutRejectsMissingColumnMetadata(t *testing.T) {
	metadata := `{"spreadsheetId":"sheet-1","title":"Budget","sheets":[{"properties":{"sheetId":7,"title":"Data","gridProperties":{"rowCount":4,"columnCount":4}}}]}`
	raw := `{"spreadsheetId":"sheet-1","title":"Budget","sheets":[{"properties":{"sheetId":7,"title":"Data","gridProperties":{"rowCount":4,"columnCount":4}},"data":[{}]}]}`
	err, _, _, _ := testCommand(t, &UpCmd{
		Spreadsheet: "Budget",
		Sheet:       "Data",
		CSVPath:     writeCSV(t, "name,count\nalpha,1\n"),
		Replace:     true,
		Layout:      true,
	},
		`{"files":[{"id":"sheet-1","name":"Budget"}]}`,
		metadata,
		metadata, `{}`, `{}`, `{}`, `{}`,
		metadata,
		metadata, `{}`,
		metadata, `{}`,
		raw,
	)
	assert.ErrorContains(t, err, `sheet "Data" has column metadata for 0 of 2 columns`)
}

func TestUploadNames(t *testing.T) {
	spreadsheet := &gog.Spreadsheet{Sheets: []*gog.Sheet{{Title: "input"}, {Title: "input_2"}}}
	cmd := &UpCmd{CSVPath: "/tmp/input.csv"}
	assert.Equal(t, "input_3", sheetTitle(cmd, spreadsheet))
	cmd.Replace = true
	assert.Equal(t, "input", sheetTitle(cmd, spreadsheet))
}

func TestNumericFormats(t *testing.T) {
	uploader := &uploader{rows: gog.Rows{
		{"zip", "count", "amount", "name"},
		{"00123", "1000", "1.25", "Ada"},
		{"00456", "2000", "2.500", "Bob"},
	}}
	assert.Equal(t, map[int]string{1: "#,##0", 2: "#,##0.000"}, uploader.numericFormats())
}

func TestRefillerKeepsRemoteOnlyColumns(t *testing.T) {
	refill := &refiller{
		localHeaders:  []string{"id", "name"},
		localRows:     gog.Rows{{"id", "name"}, {"a", "Ada 2"}},
		remoteHeaders: []string{"id", "KEEP", "name"},
		remoteRows: gog.Rows{
			{"id", "KEEP", "name"},
			{"a", "keep a", "Ada"},
			{"b", "keep b", "Bob"},
		},
		remoteGridRows: gog.Rows{
			{"id", "KEEP", "name"},
			{"a", "keep a", "Ada"},
			{"b", "keep b", "Bob"},
		},
		remoteSheetData: &gog.SheetData{Rows: make([]gog.RowData, 3)},
		remoteCols:      []int{1},
		pasteHeaders:    []string{"id", "KEEP", "name"},
		sharedCols:      []int{0, 2},
	}
	assert.Equal(t, gog.Rows{
		{"id", "KEEP", "name"},
		{"a", "keep a", "Ada 2"},
		{"", "keep b", ""},
	}, refill.pasteRows())
}
