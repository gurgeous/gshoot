package commands

import (
	"bytes"
	"strings"
	"testing"

	"github.com/gurgeous/gshoot/gog"
	"github.com/stretchr/testify/assert"
)

func TestJoinerPlansRowsAndColumns(t *testing.T) {
	left := gog.Rows{
		{"asin", "title", "price"},
		{"a", "Alpha", "10"},
		{"b", "Beta", "20"},
		{"", "Blank", "30"},
	}
	right := gog.Rows{
		{"asin", "price", "rank"},
		{"a", "11", "1"},
		{"c", "31", "3"},
		{"", "41", "4"},
	}

	join, err := newJoiner(left, right, "asin", nil)
	assert.NoError(t, err)
	assert.Equal(t, []string{"title"}, join.leftColumns)
	assert.Equal(t, []string{"rank"}, join.rightColumns)
	assert.Equal(t, []columnMatch{{left: 2, right: 1, source: "price", target: "price2"}}, join.matchColumns)
	assert.Equal(t, rowCounts{left: 2, right: 2, match: 1}, join.rowCounts)
	assert.Equal(t, gog.Rows{
		{"asin", "join", "title", "price", "price2", "rank"},
		{"a", "match", "Alpha", "10", "11", "1"},
		{"b", "left", "Beta", "20", "", ""},
		{"", "left", "Blank", "30", "", ""},
		{"c", "right", "", "", "31", "3"},
		{"", "right", "", "", "41", "4"},
	}, join.rows)
}

func TestJoinerIgnoresBlankHeaders(t *testing.T) {
	left := gog.Rows{{"id", "", "price"}, {"1", "keep", "10"}}
	right := gog.Rows{{"id", "", "price", ""}, {"1", "ignore", "11", "ignore"}}

	join, err := newJoiner(left, right, "id", nil)
	assert.NoError(t, err)
	assert.Empty(t, join.leftColumns)
	assert.Empty(t, join.rightColumns)
	assert.Equal(t, []string{"id", "join", "", "price", "price2"}, join.rows[0])
	assert.Equal(t, []string{"1", "match", "keep", "10", "11"}, join.rows[1])
}

func TestJoinCommandBacksUpBeforeWriting(t *testing.T) {
	csv := writeCSV(t, "id,price,rank\n1,11,2\n3,30,1\n")
	raw := `{"title":"Budget","sheets":[{"properties":{"sheetId":7,"title":"Data","gridProperties":{"rowCount":10,"columnCount":5}},"basicFilter":{"range":{"sheetId":7,"endRowIndex":3,"endColumnIndex":2}}}]}`
	metadata := `{"title":"Budget","sheets":[{"properties":{"sheetId":7,"title":"Data","gridProperties":{"rowCount":20,"columnCount":10}}}]}`
	responses := []string{
		`{"files":[{"id":"sheet-1","name":"Budget"}]}`,
		raw,
		`{"values":[["id","price"],["1","10"],["2","20"]]}`,
		`{"file":{"id":"backup-1","name":"Budget backup"}}`,
	}
	for range 15 {
		responses = append(responses, metadata, `{}`)
	}

	err, stdout, stderr, log := testCommand(t, &JoinCmd{
		Spreadsheet: "Budget", CSVPath: csv, Key: "id", Force: true,
	}, responses...)
	assert.NoError(t, err)
	assert.Contains(t, stdout, "Column plan\n  key:   id")
	assert.Contains(t, stdout, "Row counts\n  left:  1\n  match: 1\n  right: 1")
	assert.Contains(t, stdout, `Workaround: "join" is inserted after the first column until gog can insert before column A.`)
	assert.Contains(t, stdout, "spreadsheet: https://docs.google.com/spreadsheets/d/sheet-1/edit")
	assert.Contains(t, stdout, "backup: https://docs.google.com/spreadsheets/d/backup-1/edit")
	assert.Contains(t, stderr, "creating backup...\njoining...")
	assert.NotContains(t, stderr, "creating backup and joining")
	assert.Less(t, strings.Index(log, "sheets copy sheet-1"), strings.Index(log, "sheets insert sheet-1"))
	assert.Contains(t, log, "sheets filter set sheet-1 'Data'!A1:E4 --force")
	assert.Equal(t, 4, strings.Count(log, "sheets update sheet-1"))
	assert.Contains(t, log, "sheets update sheet-1 'Data'!B1 ")
	assert.Contains(t, log, "sheets update sheet-1 'Data'!D1 ")
	assert.Contains(t, log, "sheets update sheet-1 'Data'!E1 ")
	assert.Contains(t, log, "sheets update sheet-1 'Data'!A4 ")
	assert.NotContains(t, log, "sheets update sheet-1 'Data'!A1 ")
	assert.NotContains(t, log, "sheets update sheet-1 'Data'!C1 ")
}

func TestJoinCommandDoesNotWriteWhenBackupFails(t *testing.T) {
	csv := writeCSV(t, "id,price\n1,11\n")
	raw := `{"title":"Budget","sheets":[{"properties":{"sheetId":7,"title":"Data","gridProperties":{"rowCount":10,"columnCount":5}}}]}`
	err, _, stderr, log := testCommand(t, &JoinCmd{
		Spreadsheet: "Budget", CSVPath: csv, Key: "id", Force: true,
	},
		`{"files":[{"id":"sheet-1","name":"Budget"}]}`,
		raw,
		`{"values":[["id","price"],["1","10"]]}`,
		`not-json`,
	)
	assert.ErrorContains(t, err, "create backup")
	assert.Contains(t, stderr, "creating backup...")
	assert.NotContains(t, stderr, "joining...")
	assert.NotContains(t, log, "sheets insert")
}

func TestJoinerPreviewAndOperations(t *testing.T) {
	left := gog.Rows{{"id", "name", "price"}, {"1", "Ada", "10"}, {"2", "Bob", "20"}}
	right := gog.Rows{{"id", "price", "rank"}, {"1", "11", "2"}, {"3", "30", "1"}}
	join, err := newJoiner(left, right, "id", nil)
	assert.NoError(t, err)

	var preview bytes.Buffer
	join.preview(&preview)
	assert.Equal(t, "Column plan\n  key:   id\n  left:  name\n  match: price\n  right: rank\nRow counts\n  left:  1\n  match: 1\n  right: 1\n\nWorkaround: \"join\" is inserted after the first column until gog can insert before column A.\n", preview.String())

	operations := join.operations(7, true)
	assert.Equal(t, "cols", operations[0].InsertDimension.Dimension)
	assert.Equal(t, 3, operations[0].InsertDimension.Start)
	assert.True(t, operations[0].InsertDimension.After)
	assert.Equal(t, "cols", operations[1].InsertDimension.Dimension)
	assert.Equal(t, 4, operations[1].InsertDimension.Start)
	assert.True(t, operations[1].InsertDimension.After)
	assert.Equal(t, 1, operations[2].InsertDimension.Start)
	assert.True(t, operations[2].InsertDimension.After)
	assert.Equal(t, "rows", operations[3].InsertDimension.Dimension)
	assert.Equal(t, 3, operations[3].InsertDimension.Start)
	assert.True(t, *operations[3].InsertDimension.InheritFromBefore)
	existingPastes := map[int]gog.Rows{}
	formattedColumns := []int{}
	for _, operation := range operations {
		if operation.FormatCells != nil {
			formattedColumns = append(formattedColumns, operation.FormatCells.Range.StartColumnIndex)
		}
		if operation.PasteRows != nil && operation.PasteRows.RowIndex == 0 {
			existingPastes[operation.PasteRows.ColumnIndex] = operation.PasteRows.Rows
		}
	}
	assert.Equal(t, []int{1, 4, 5}, formattedColumns)
	assert.Equal(t, map[int]gog.Rows{
		1: {{"join"}, {"match"}, {"left"}},
		4: {{"price2"}, {"11"}, {""}},
		5: {{"rank"}, {"2"}, {""}},
	}, existingPastes)
	assert.Equal(t, 3, operations[10].PasteRows.RowIndex)
	assert.Equal(t, gog.Rows{{"3", "right", "", "", "30", "1"}}, operations[10].PasteRows.Rows)
	assert.Equal(t, &gog.GridRange{SheetID: 7, EndRowIndex: 4, EndColumnIndex: 6}, operations[11].SetFilter)
}

func TestJoinerSelectsRightColumns(t *testing.T) {
	left := gog.Rows{{"id", "name", "price"}, {"1", "Ada", "10"}}
	right := gog.Rows{{"id", "price", "rank", "note"}, {"1", "11", "2", "ok"}}

	join, err := newJoiner(left, right, "id", []string{"note", "price"})
	assert.NoError(t, err)
	assert.Equal(t, []string{"name"}, join.leftColumns)
	assert.Equal(t, []string{"note"}, join.rightColumns)
	assert.Equal(t, "price", join.matchColumns[0].source)
	assert.Equal(t, []string{"id", "join", "name", "price", "price2", "note"}, join.rows[0])
}

func TestJoinerRejectsInvalidInputs(t *testing.T) {
	validLeft := gog.Rows{{"id", "price"}, {"1", "10"}}
	validRight := gog.Rows{{"id", "price"}, {"1", "11"}}

	tests := []struct {
		name     string
		left     gog.Rows
		right    gog.Rows
		key      string
		columns  []string
		contains string
	}{
		{name: "missing left key", left: gog.Rows{{"name"}, {"Ada"}}, right: validRight, key: "id", contains: `sheet has no "id" column`},
		{name: "missing right key", left: validLeft, right: gog.Rows{{"name"}, {"Ada"}}, key: "id", contains: `csv has no "id" column`},
		{name: "duplicate left key", left: gog.Rows{{"id"}, {"1"}, {"1"}}, right: validRight, key: "id", contains: "google sheet has duplicate key id=1"},
		{name: "duplicate right key", left: validLeft, right: gog.Rows{{"id"}, {"1"}, {"1"}}, key: "id", contains: "csv has duplicate key id=1"},
		{name: "no matches", left: validLeft, right: gog.Rows{{"id"}, {"2"}}, key: "id", contains: "no rows match"},
		{name: "join exists", left: gog.Rows{{"id", "join"}, {"1", "left"}}, right: validRight, key: "id", contains: `column "join" already exists`},
		{name: "generated exists", left: gog.Rows{{"id", "price", "price2"}, {"1", "10", "old"}}, right: validRight, key: "id", contains: `column "price2" already exists`},
		{name: "unknown selection", left: validLeft, right: validRight, key: "id", columns: []string{"rank"}, contains: `csv has no "rank" column`},
		{name: "key selected", left: validLeft, right: validRight, key: "id", columns: []string{"id"}, contains: `--columns must not include join key "id"`},
		{name: "duplicate selection", left: validLeft, right: validRight, key: "id", columns: []string{"price", "price"}, contains: `duplicate --columns value "price"`},
		{name: "blank selection", left: validLeft, right: validRight, key: "id", columns: []string{""}, contains: "--columns contains an empty column"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newJoiner(tt.left, tt.right, tt.key, tt.columns)
			assert.ErrorContains(t, err, tt.contains)
		})
	}
}
