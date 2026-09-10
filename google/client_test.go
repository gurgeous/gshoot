package google

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestListSpreadsheetFilesPaginatesAndSorts(t *testing.T) {
	client, dir := fakeGog(t,
		`{"files":[{"id":"1","name":"Alpha","modifiedByMeTime":"2026-05-07T11:00:00Z"}],"nextPageToken":"next"}`,
		`{"files":[{"id":"2","name":"Beta","modifiedByMeTime":"2026-05-07T12:00:00Z"}]}`,
	)
	files, err := client.ListSpreadsheetFiles(context.Background(), 20)
	assert.NoError(t, err)
	assert.Equal(t, "Beta", files[0].Name)
	log := readTestFile(t, filepath.Join(dir, "log"))
	assert.Contains(t, log, "--page next")
	assert.Contains(t, log, "modifiedByMeTime")
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

func TestBatchUpdateUsesNamedGogCommands(t *testing.T) {
	metadata := `{"spreadsheetId":"sheet-1","title":"Budget","sheets":[{"properties":{"sheetId":7,"title":"Data","gridProperties":{"rowCount":10,"columnCount":5}}}]}`
	client, dir := fakeGog(t, metadata, `{}`, metadata, `{}`, metadata, `{}`)
	_, err := client.BatchUpdate(context.Background(), "sheet-1", []Request{
		{PasteData: &PasteDataRequest{Coordinate: GridCoordinate{SheetID: 7}, Data: "name,count\nalpha,1\n", Delimiter: ",", Type: "PASTE_NORMAL"}},
		{SetBasicFilter: &SetBasicFilterRequest{Filter: BasicFilter{Range: GridRange{SheetID: 7, EndRowIndex: 2, EndColumnIndex: 2}}}},
		{UpdateDimensionProperties: &UpdateDimensionPropertiesRequest{Range: DimensionRange{SheetID: 7, Dimension: "COLUMNS", EndIndex: 1}, Properties: DimensionProperties{PixelSize: 120}}},
	})
	assert.NoError(t, err)
	log := readTestFile(t, filepath.Join(dir, "log"))
	assert.Contains(t, log, "sheets update sheet-1 'Data'!A1 --values-json @- --input USER_ENTERED")
	assert.Contains(t, log, "sheets filter set sheet-1 'Data'!A1:B2 --force")
	assert.Contains(t, log, "sheets resize-columns sheet-1 'Data'!A:A --width 120")
	assert.JSONEq(t, `[["name","count"],["alpha","1"]]`, readTestFile(t, filepath.Join(dir, "stdin.2")))
}

func TestAddSheetPreservesExactGridSize(t *testing.T) {
	before := `{"spreadsheetId":"sheet-1","sheets":[{"properties":{"sheetId":0,"title":"Sheet1","gridProperties":{"rowCount":1000,"columnCount":26}}}]}`
	after := `{"spreadsheetId":"sheet-1","sheets":[{"properties":{"sheetId":7,"title":"Data","gridProperties":{"rowCount":1000,"columnCount":26}}}]}`
	client, dir := fakeGog(t, before, `{"sheetId":7}`, after, `{}`, `{}`)
	_, err := client.BatchUpdate(context.Background(), "sheet-1", []Request{{
		AddSheet: &AddSheetRequest{Properties: SheetProperties{
			Title:          "Data",
			GridProperties: &GridProperties{RowCount: 4, ColumnCount: 3},
		}},
	}})
	assert.NoError(t, err)
	log := readTestFile(t, filepath.Join(dir, "log"))
	assert.Contains(t, log, "sheets delete-dimension sheet-1 Data --dimension ROWS --start 5 --end 1000 --force")
	assert.Contains(t, log, "sheets delete-dimension sheet-1 Data --dimension COLUMNS --start 4 --end 26 --force")
}

func TestReplaceClearsAllCellData(t *testing.T) {
	metadata := `{"spreadsheetId":"sheet-1","sheets":[{"properties":{"sheetId":7,"title":"Data","gridProperties":{"rowCount":10,"columnCount":5}}}]}`
	client, dir := fakeGog(t, metadata, `{}`, `{}`, `{}`, `{}`)
	_, err := client.BatchUpdate(context.Background(), "sheet-1", []Request{{
		UpdateCells: &UpdateCellsRequest{Range: GridRange{SheetID: 7}, Fields: "*"},
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
