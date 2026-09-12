package commands

import (
	"strings"
	"testing"

	"github.com/gurgeous/gshoot/gog"
	"github.com/stretchr/testify/assert"
)

func TestHyperlinkerPlansBackupAndFormulaRuns(t *testing.T) {
	rows := gog.Rows{
		{"name", "link", "notes"},
		{"Ada \"Ace\"\nSr", "https://example.test/a", "first"},
		{"Blank link", "", "second"},
		{"", "https://example.test/blank", "third"},
		{"Bob", "not validated", "fourth"},
		{"Cal", "https://example.test/c", "fifth"},
	}
	h, err := newHyperlinker(rows, "name", "link")
	assert.NoError(t, err)
	assert.Len(t, h.cells, 3)
	assert.Equal(t, "name2", h.backupHeader)

	prepare := h.prepare(7)
	assert.Len(t, prepare, 2)
	assert.Equal(t, &gog.InsertDimensionOperation{
		SheetID: 7, Dimension: "cols", Start: 1, Count: 1, After: true,
	}, prepare[0].InsertDimension)
	assert.Equal(t, &gog.CopyCellsOperation{
		Source:      gog.GridRange{SheetID: 7, EndColumnIndex: 1},
		Destination: gog.GridRange{SheetID: 7, StartColumnIndex: 1, EndColumnIndex: 2},
		Type:        "NORMAL",
	}, prepare[1].CopyCells)

	values := h.values(7)
	assert.Len(t, values, 3)
	assert.Equal(t, &gog.PasteRowsOperation{
		SheetID: 7, ColumnIndex: 1, Rows: gog.Rows{{"name2"}},
	}, values[0].PasteRows)
	assert.Equal(t, &gog.PasteRowsOperation{
		SheetID: 7, RowIndex: 1, Rows: gog.Rows{{"=HYPERLINK(C2,\"Ada \"\"Ace\"\"\nSr\")"}},
	}, values[1].PasteRows)
	assert.Equal(t, &gog.PasteRowsOperation{
		SheetID: 7, RowIndex: 4, Rows: gog.Rows{
			{`=HYPERLINK(C5,"Bob")`},
			{`=HYPERLINK(C6,"Cal")`},
		},
	}, values[2].PasteRows)
}

func TestHyperlinkerAdjustsLinkColumnBeforeText(t *testing.T) {
	h, err := newHyperlinker(gog.Rows{
		{"link", "name"},
		{"https://example.test/a", "Ada"},
	}, "name", "link")
	assert.NoError(t, err)
	assert.Equal(t, `=HYPERLINK(A2,"Ada")`, h.values(7)[1].PasteRows.Rows[0][0])
	assert.Equal(t, 1, h.values(7)[1].PasteRows.ColumnIndex)
}

func TestHyperlinkerInfersAmazonLinksForASIN(t *testing.T) {
	h, err := newHyperlinker(gog.Rows{
		{"asin", "name"},
		{"B012345678", "Widget"},
		{"", "Blank"},
	}, "asin", "")
	assert.NoError(t, err)
	assert.Len(t, h.cells, 1)
	assert.Equal(t,
		`=HYPERLINK("https://www.amazon.com/dp/B012345678","B012345678")`,
		h.values(7)[1].PasteRows.Rows[0][0],
	)
}

func TestHyperlinkerRejectsUnknownInferredLink(t *testing.T) {
	_, err := newHyperlinker(gog.Rows{{"name"}, {"Ada"}}, "name", "")
	assert.ErrorContains(t, err, `cannot infer links for column "name"`)
}

func TestHyperlinkerRejectsInvalidColumns(t *testing.T) {
	rows := gog.Rows{{"name", "link", "name2"}, {"Ada", "https://example.test", "old"}}
	tests := []struct {
		name string
		text string
		link string
		want string
	}{
		{name: "missing text", text: "title", link: "link", want: `sheet has no "title" column`},
		{name: "missing link", text: "name", link: "url", want: `sheet has no "url" column`},
		{name: "same column", text: "name", link: "name", want: "columns must differ"},
		{name: "backup exists", text: "name", link: "link", want: `column "name2" already exists`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newHyperlinker(rows, tt.text, tt.link)
			assert.ErrorContains(t, err, tt.want)
		})
	}
}

func TestHyperlinkerNoEligibleRows(t *testing.T) {
	h, err := newHyperlinker(gog.Rows{
		{"name", "link"},
		{"Ada", ""},
		{"", "https://example.test"},
	}, "name", "link")
	assert.NoError(t, err)
	assert.Empty(t, h.cells)
	assert.Empty(t, h.prepare(7))
	assert.Empty(t, h.values(7))
}

func TestHyperlinkCommand(t *testing.T) {
	err, stdout, stderr, log := testCommand(t, &HyperlinkCmd{
		Spreadsheet: "People", TextColumn: "name", LinkColumn: "link",
	},
		`{"files":[{"id":"sheet-1","name":"People"}]}`,
		`{"spreadsheetId":"sheet-1","title":"People","sheets":[{"properties":{"sheetId":7,"title":"Data"}}]}`,
		`{"values":[["name","link"],["Ada","https://example.test/a"],["Skip",""]]}`,
		`{}`,
		`{}`,
	)
	assert.NoError(t, err)
	assert.Equal(t, "transformed: 1\nhttps://docs.google.com/spreadsheets/d/sheet-1/edit#gid=7", stdout)
	assert.Contains(t, stderr, "backing up column")
	assert.Contains(t, stderr, "transforming 1 row")
	assert.Equal(t, 1, strings.Count(log, "api call sheets v4 sheets.spreadsheets.batchUpdate"))
	assert.Equal(t, 1, strings.Count(log, "sheets batch-update sheet-1"))
}

func TestHyperlinkCommandNoEligibleRowsDoesNotWrite(t *testing.T) {
	err, stdout, _, log := testCommand(t, &HyperlinkCmd{
		Spreadsheet: "People", TextColumn: "name", LinkColumn: "link",
	},
		`{"files":[{"id":"sheet-1","name":"People"}]}`,
		`{"spreadsheetId":"sheet-1","title":"People","sheets":[{"properties":{"sheetId":7,"title":"Data"}}]}`,
		`{"values":[["name","link"],["Ada",""]]}`,
	)
	assert.NoError(t, err)
	assert.Equal(t, "transformed: 0\nhttps://docs.google.com/spreadsheets/d/sheet-1/edit#gid=7", stdout)
	assert.NotContains(t, log, "api call")
	assert.NotContains(t, log, "sheets batch-update")
}

func TestHyperlinkCommandPreservesBackupWhenFormulaWriteFails(t *testing.T) {
	err, _, _, log := testCommand(t, &HyperlinkCmd{
		Spreadsheet: "People", TextColumn: "name", LinkColumn: "link",
	},
		`{"files":[{"id":"sheet-1","name":"People"}]}`,
		`{"spreadsheetId":"sheet-1","title":"People","sheets":[{"properties":{"sheetId":7,"title":"Data"}}]}`,
		`{"values":[["name","link"],["Ada","https://example.test/a"]]}`,
		`{}`,
		`ERROR: Google API error (429 rateLimitExceeded): quota exceeded`,
	)
	assert.ErrorContains(t, err, `backup column "name2" is intact`)
	assert.Equal(t, 1, strings.Count(log, "api call sheets v4 sheets.spreadsheets.batchUpdate"))
	assert.Equal(t, 1, strings.Count(log, "sheets batch-update sheet-1"))
}
