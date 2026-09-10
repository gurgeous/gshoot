package commands

import (
	"testing"

	"github.com/gurgeous/gshoot/google"
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

func TestUploadNames(t *testing.T) {
	spreadsheet := &google.Spreadsheet{Sheets: []*google.Sheet{{Title: "input"}, {Title: "input_2"}}}
	cmd := &UpCmd{CSVPath: "/tmp/input.csv"}
	assert.Equal(t, "input_3", sheetTitle(cmd, spreadsheet))
	cmd.Replace = true
	assert.Equal(t, "input", sheetTitle(cmd, spreadsheet))
}

func TestNumericFormats(t *testing.T) {
	uploader := &uploader{rows: google.Rows{
		{"zip", "count", "amount", "name"},
		{"00123", "1000", "1.25", "Ada"},
		{"00456", "2000", "2.500", "Bob"},
	}}
	assert.Equal(t, map[int]string{1: "#,##0", 2: "#,##0.000"}, uploader.numericFormats())
}

func TestRefillerKeepsRemoteOnlyColumns(t *testing.T) {
	refill := &refiller{
		localHeaders:  []string{"id", "name"},
		localRows:     google.Rows{{"id", "name"}, {"a", "Ada 2"}},
		remoteHeaders: []string{"id", "KEEP", "name"},
		remoteRows: google.Rows{
			{"id", "KEEP", "name"},
			{"a", "keep a", "Ada"},
			{"b", "keep b", "Bob"},
		},
		remoteGridRows: google.Rows{
			{"id", "KEEP", "name"},
			{"a", "keep a", "Ada"},
			{"b", "keep b", "Bob"},
		},
		remoteSheetData: &google.SheetData{Rows: make([]google.RowData, 3)},
		remoteCols:      []int{1},
		pasteHeaders:    []string{"id", "KEEP", "name"},
		sharedCols:      []int{0, 2},
	}
	assert.Equal(t, google.Rows{
		{"id", "KEEP", "name"},
		{"a", "keep a", "Ada 2"},
		{"", "keep b", ""},
	}, refill.pasteRows())
}
