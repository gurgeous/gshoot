package gog

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
)

func TestListSpreadsheetFilesUsesGogLimitAndOrder(t *testing.T) {
	client, dir := fakeGog(t, `{"files":[{"id":"1","name":"Alpha"},{"id":"2","name":"Beta"}],"nextPageToken":"next"}`)
	files, err := client.ListSpreadsheetFiles(context.Background(), 20)
	assert.NoError(t, err)
	assert.Equal(t, []*File{{ID: "1", Name: "Alpha"}, {ID: "2", Name: "Beta"}}, files)
	log := readTestFile(t, filepath.Join(dir, "log"))
	assert.Contains(t, log, "drive ls --all --max 20")
	assert.NotContains(t, log, "--page")
	assert.Contains(t, log, "--sort modifiedByMeTime --order desc")
}

func TestDebugLogsTimestampedWrappedGogCalls(t *testing.T) {
	client, _ := fakeGog(t, `{"files":[]}`)
	path := filepath.Join(t.TempDir(), "stderr")
	stderr, err := os.Create(path)
	assert.NoError(t, err)
	original := os.Stderr
	os.Stderr = stderr
	t.Cleanup(func() {
		os.Stderr = original
		assert.NoError(t, stderr.Close())
	})
	t.Setenv("GSHOOT_DEBUG", "1")

	_, err = client.listFiles(context.Background(), strings.Repeat("long condition ", 10), 20)
	assert.NoError(t, err)
	assert.NoError(t, stderr.Sync())

	debug := readTestFile(t, path)
	assert.Regexp(t, `^\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d\.\d{3}Z gog `, debug)
	assert.Contains(t, debug, "drive ls")
	assert.Contains(t, debug, "--query")
	for _, line := range strings.Split(debug, "\n") {
		assert.Less(t, utf8.RuneCountInString(line), 80, line)
	}
}

func TestFindSpreadsheetAcceptsNameIDAndURL(t *testing.T) {
	t.Run("name", func(t *testing.T) {
		client, dir := fakeGog(t, `{"files":[{"id":"1","name":"Bob's Budget"}]}`)
		file, err := client.FindSpreadsheetFile(context.Background(), "Bob's Budget")
		assert.NoError(t, err)
		assert.Equal(t, "1", file.ID)
		assert.Contains(t, readTestFile(t, filepath.Join(dir, "log")), `name = 'Bob\'s Budget'`)
	})
	for _, ref := range []string{"12345678901234567890", "https://docs.google.com/spreadsheets/d/12345678901234567890/edit"} {
		responses := []string{`{"spreadsheetId":"12345678901234567890","title":"Budget","sheets":[]}`}
		if !strings.Contains(ref, "://") {
			responses = append([]string{`{"files":[]}`}, responses...)
		}
		client, dir := fakeGog(t, responses...)
		file, err := client.FindSpreadsheetFile(context.Background(), ref)
		assert.NoError(t, err)
		assert.Equal(t, "Budget", file.Name)
		assert.Contains(t, readTestFile(t, filepath.Join(dir, "log")), "sheets metadata 12345678901234567890")
	}
}

func TestGetRowsUsesQuotedRange(t *testing.T) {
	client, dir := fakeGog(t, `{"values":[["name","count"],["alpha",1]]}`)
	rows, err := client.GetRows(context.Background(), "sheet-1", "Bob's Sheet")
	assert.NoError(t, err)
	assert.Equal(t, Rows{{"name", "count"}, {"alpha", "1"}}, rows)
	assert.Contains(t, readTestFile(t, filepath.Join(dir, "log")), `sheets get sheet-1 'Bob''s Sheet'`)
}

func TestGetHeaderAndAppendRows(t *testing.T) {
	client, dir := fakeGog(t, `{"values":[["id","amount"]]}`, `{}`)
	header, err := client.GetHeader(context.Background(), "sheet-1", "Bob's Sheet")
	assert.NoError(t, err)
	assert.Equal(t, []string{"id", "amount"}, header)
	assert.NoError(t, client.AppendRows(context.Background(), "sheet-1", "Bob's Sheet", Rows{{"a", "10"}}))

	log := readTestFile(t, filepath.Join(dir, "log"))
	assert.Contains(t, log, `sheets get sheet-1 'Bob''s Sheet'!1:1`)
	assert.Contains(t, log, `sheets append sheet-1 'Bob''s Sheet' --values-json @- --input USER_ENTERED --insert INSERT_ROWS`)
	assert.JSONEq(t, `[["a","10"]]`, readTestFile(t, filepath.Join(dir, "stdin.2")))
}

func TestDuplicateTab(t *testing.T) {
	client, dir := fakeGog(t, `{"spreadsheetId":"sheet-1","sourceSheetId":7,"sheetId":9,"title":"Data backup","index":1}`)
	sheet, err := client.DuplicateTab(context.Background(), "sheet-1", "Data", "Data backup")
	assert.NoError(t, err)
	assert.Equal(t, &Sheet{ID: 9, Title: "Data backup"}, sheet)
	assert.Contains(t, readTestFile(t, filepath.Join(dir, "log")), "sheets duplicate-tab sheet-1 Data Data backup")
}

func TestGridDataIncludesFilterRange(t *testing.T) {
	client, dir := fakeGog(t, `{"sheets":[{"properties":{"sheetId":7,"title":"Data","gridProperties":{"rowCount":10,"columnCount":5}},"basicFilter":{"range":{"sheetId":7,"startRowIndex":1,"endRowIndex":8,"startColumnIndex":2,"endColumnIndex":5}}}]}`)
	spreadsheet, err := client.GetSpreadsheetWithGridData(context.Background(), "sheet-1", "Data")
	assert.NoError(t, err)
	assert.Equal(t, &GridRange{SheetID: 7, StartRowIndex: 1, EndRowIndex: 8, StartColumnIndex: 2, EndColumnIndex: 5}, spreadsheet.Data[7].FilterRange)
	assert.Contains(t, readTestFile(t, filepath.Join(dir, "log")), "sheets raw sheet-1 --sheet Data --include-grid-data")
}

func TestApplyUsesNamedGogCommands(t *testing.T) {
	metadata := `{"spreadsheetId":"sheet-1","title":"Budget","sheets":[{"properties":{"sheetId":7,"title":"Data","gridProperties":{"rowCount":10,"columnCount":5}}}]}`
	client, dir := fakeGog(t, metadata, `{}`, metadata, `{}`, metadata, `{}`, metadata, `{}`)
	inherit := false
	_, err := client.Apply(context.Background(), "sheet-1", []Operation{
		{InsertDimension: &InsertDimensionOperation{SheetID: 7, Dimension: "cols", Start: 2, Count: 1, InheritFromBefore: &inherit}},
		{PasteRows: &PasteRowsOperation{SheetID: 7, RowIndex: 2, ColumnIndex: 1, Rows: Rows{{"alpha", "1"}}}},
		{SetFilter: &GridRange{SheetID: 7, EndRowIndex: 2, EndColumnIndex: 2}},
		{ResizeColumns: &ResizeColumnsOperation{Range: ColumnRange{SheetID: 7, EndIndex: 1}, PixelSize: 120}},
	})
	assert.NoError(t, err)
	log := readTestFile(t, filepath.Join(dir, "log"))
	assert.Contains(t, log, "sheets insert sheet-1 Data cols 2 --count 1 --inherit-from-before=false")
	assert.Contains(t, log, "sheets update sheet-1 'Data'!B3 --values-json @- --input USER_ENTERED")
	assert.Contains(t, log, "sheets filter set sheet-1 'Data'!A1:B2 --force")
	assert.Contains(t, log, "sheets resize-columns sheet-1 'Data'!A:A --width 120")
	assert.JSONEq(t, `[["alpha","1"]]`, readTestFile(t, filepath.Join(dir, "stdin.4")))
}

func TestApplySheetBatchUsesSpreadsheetAndValueBatches(t *testing.T) {
	client, dir := fakeGog(t, `{}`, `{}`, `{}`)
	sheet := &Sheet{ID: 7, Title: "Data"}

	err := client.ApplySheetBatch(context.Background(), "sheet-1", sheet, []Operation{
		{InsertDimension: &InsertDimensionOperation{
			SheetID: 7, Dimension: "cols", Start: 1, Count: 1,
		}},
		{FormatCells: &FormatCellsOperation{
			Range: GridRange{SheetID: 7, StartColumnIndex: 2, EndColumnIndex: 4},
		}},
	})
	assert.NoError(t, err)

	err = client.ApplySheetBatch(context.Background(), "sheet-1", sheet, []Operation{
		{PasteRows: &PasteRowsOperation{
			SheetID: 7, RowIndex: 1, ColumnIndex: 2,
			Rows: Rows{{"11", "=A2"}, {"12", "=A3"}},
		}},
	})
	assert.NoError(t, err)

	err = client.ApplySheetBatch(context.Background(), "sheet-1", sheet, []Operation{
		{SetFilter: &GridRange{SheetID: 7, EndRowIndex: 4, EndColumnIndex: 5}},
		{AutoResizeColumns: &ColumnRange{SheetID: 7, StartIndex: 2, EndIndex: 4}},
	})
	assert.NoError(t, err)

	log := readTestFile(t, filepath.Join(dir, "log"))
	assert.Contains(t, log, "api call sheets v4 sheets.spreadsheets.batchUpdate")
	assert.Contains(t, log, `--params {"spreadsheetId":"sheet-1"}`)
	assert.Contains(t, log, "--scope https://www.googleapis.com/auth/spreadsheets --allow-write --force")
	assert.Contains(t, log, "sheets batch-update sheet-1 --data-json @- --input USER_ENTERED")
	assert.JSONEq(t, `{
		"requests": [
			{"insertDimension":{"range":{"sheetId":7,"dimension":"COLUMNS","startIndex":0,"endIndex":1},"inheritFromBefore":false}},
			{"repeatCell":{"range":{"sheetId":7,"startColumnIndex":2,"endColumnIndex":4},"cell":{"userEnteredFormat":{}},"fields":"userEnteredFormat"}}
		]
	}`, readTestFile(t, filepath.Join(dir, "body.1")))
	assert.JSONEq(t, `[
		{"range":"'Data'!C2","values":[["11","=A2"],["12","=A3"]]}
	]`, readTestFile(t, filepath.Join(dir, "stdin.2")))
	assert.JSONEq(t, `{
		"requests": [
			{"setBasicFilter":{"filter":{"range":{"sheetId":7,"endRowIndex":4,"endColumnIndex":5}}}},
			{"autoResizeDimensions":{"dimensions":{"sheetId":7,"dimension":"COLUMNS","startIndex":2,"endIndex":4}}}
		]
	}`, readTestFile(t, filepath.Join(dir, "body.3")))
	assert.Equal(t, "-rw-------", readTestFile(t, filepath.Join(dir, "mode.1")))
	assert.Equal(t, "-rw-------", readTestFile(t, filepath.Join(dir, "mode.3")))
	for _, field := range strings.Fields(log) {
		if !strings.HasPrefix(field, "@/") {
			continue
		}
		_, err := os.Stat(strings.TrimPrefix(field, "@"))
		assert.True(t, os.IsNotExist(err))
	}
}

func TestPasteValuesPreservesFormulas(t *testing.T) {
	metadata := `{"spreadsheetId":"sheet-1","sheets":[{"properties":{"sheetId":7,"title":"Data","gridProperties":{"rowCount":10,"columnCount":5}}}]}`
	client, dir := fakeGog(t, metadata, `{}`)
	_, err := client.Apply(context.Background(), "sheet-1", []Operation{{
		PasteRows: &PasteRowsOperation{
			SheetID: 7,
			Rows:    Rows{{"id", "calc"}, {"a", "=A2"}},
		},
	}})
	assert.NoError(t, err)
	assert.Contains(t, readTestFile(t, filepath.Join(dir, "log")), "--input USER_ENTERED")
	assert.JSONEq(t, `[["id","calc"],["a","=A2"]]`, readTestFile(t, filepath.Join(dir, "stdin.2")))
}

func TestAddSheetPreservesExactGridSize(t *testing.T) {
	before := `{"spreadsheetId":"sheet-1","sheets":[{"properties":{"sheetId":0,"title":"Sheet1","gridProperties":{"rowCount":1000,"columnCount":26}}}]}`
	after := `{"spreadsheetId":"sheet-1","sheets":[{"properties":{"sheetId":7,"title":"Data","gridProperties":{"rowCount":1000,"columnCount":26}}}]}`
	client, dir := fakeGog(t, before, `{"sheetId":7}`, after, `{}`, `{}`)
	results, err := client.Apply(context.Background(), "sheet-1", []Operation{{
		AddSheet: &AddSheetOperation{
			Title:          "Data",
			GridProperties: &GridProperties{RowCount: 4, ColumnCount: 3},
		},
	}})
	assert.NoError(t, err)
	assert.Equal(t, int64(7), results[0].AddedSheet.ID)
	log := readTestFile(t, filepath.Join(dir, "log"))
	assert.Contains(t, log, "sheets delete-dimension sheet-1 Data --dimension ROWS --start 5 --end 1000 --force")
	assert.Contains(t, log, "sheets delete-dimension sheet-1 Data --dimension COLUMNS --start 4 --end 26 --force")
}

func TestReplaceClearsAllCellData(t *testing.T) {
	metadata := `{"spreadsheetId":"sheet-1","sheets":[{"properties":{"sheetId":7,"title":"Data","gridProperties":{"rowCount":10,"columnCount":5}}}]}`
	client, dir := fakeGog(t, metadata, `{}`, `{}`, `{}`, `{}`)
	_, err := client.Apply(context.Background(), "sheet-1", []Operation{{
		ClearCells: &ClearCellsOperation{Range: GridRange{SheetID: 7}, All: true},
	}})
	assert.NoError(t, err)
	log := readTestFile(t, filepath.Join(dir, "log"))
	assert.Contains(t, log, "sheets clear sheet-1 'Data'!A1:E10")
	assert.Contains(t, log, "sheets format sheet-1 'Data'!A1:E10 --format-json {} --format-fields userEnteredFormat")
	assert.Contains(t, log, "sheets validation clear sheet-1 'Data'!A1:E10 --filtered-rows-included")
	assert.Contains(t, log, "sheets update-note sheet-1 'Data'!A1:E10 --note")
}

func TestWipeReplacesTabsWithSheet1(t *testing.T) {
	metadata := `{"spreadsheetId":"sheet-1","sheets":[{"properties":{"sheetId":7,"title":"sheet1"}},{"properties":{"sheetId":8,"title":"Data"}}]}`
	client, dir := fakeGog(t, metadata, `{}`, `{}`, `{}`, `{}`)
	assert.NoError(t, client.WipeSpreadsheet(context.Background(), "sheet-1"))
	log := readTestFile(t, filepath.Join(dir, "log"))
	assert.Contains(t, log, "sheets rename-tab sheet-1 sheet1 gshoot-wipe")
	assert.Contains(t, log, "sheets add-tab sheet-1 Sheet1 --index 0")
	assert.Contains(t, log, "sheets delete-tab sheet-1 gshoot-wipe --force")
	assert.Contains(t, log, "sheets delete-tab sheet-1 Data --force")
}

func TestGogErrorIncludesStderr(t *testing.T) {
	client, dir := fakeGog(t, "")
	assert.NoError(t, os.WriteFile(filepath.Join(dir, "error.1"), []byte("authentication required"), 0o600))
	_, err := client.GetSpreadsheet(context.Background(), "sheet-1")
	assert.EqualError(t, err, "gog: authentication required")
}

func TestNewClientRequiresGog(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	_, err := NewClient(context.Background())
	assert.ErrorContains(t, err, "brew install openclaw/tap/gogcli")
}

func TestSpreadsheetID(t *testing.T) {
	assert.Equal(t, "abc12345678901234567", spreadsheetID("https://docs.google.com/spreadsheets/d/abc12345678901234567/edit#gid=0"))
	assert.Equal(t, "abc12345678901234567", spreadsheetID("abc12345678901234567"))
	assert.Empty(t, spreadsheetID("Quarterly Budget"))
}

func fakeGog(t *testing.T, responses ...string) (*Client, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "gog")
	script := `#!/bin/sh
dir="$FAKE_GOG_DIR"
n=0
test -f "$dir/count" && n=$(cat "$dir/count")
n=$((n + 1))
echo "$n" > "$dir/count"
printf '%s\n' "$*" >> "$dir/log"
previous=
for arg do
  if test "$previous" = "--body"; then
    case "$arg" in
      @*)
        ls -l "${arg#@}" | cut -c1-10 > "$dir/mode.$n"
        cp "${arg#@}" "$dir/body.$n"
        ;;
    esac
  fi
  previous="$arg"
done
cat > "$dir/stdin.$n"
if test -f "$dir/error.$n"; then cat "$dir/error.$n" >&2; exit 1; fi
test -f "$dir/response.$n" && cat "$dir/response.$n"
`
	assert.NoError(t, os.WriteFile(path, []byte(script), 0o700))
	for i, response := range responses {
		assert.NoError(t, os.WriteFile(filepath.Join(dir, "response."+strconv.Itoa(i+1)), []byte(response), 0o600))
	}
	t.Setenv("FAKE_GOG_DIR", dir)
	return &Client{gog: path}, dir
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	assert.NoError(t, err)
	return strings.TrimSpace(string(data))
}
