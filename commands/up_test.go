package commands

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUpCommandRenamesBlankDefaultSheet(t *testing.T) {
	csvPath := writeCSV(t, "name,count\nalpha,1\n")
	batches := []map[string]any{}

	err, stdout, _ := testCommand(t, &UpCmd{Spreadsheet: "Budget", CSVPath: csvPath}, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/drive/v3/files":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"files": []map[string]string{{"id": "sheet-1", "name": "Budget"}},
			})
		case r.URL.Path == "/v4/spreadsheets/sheet-1":
			writeSpreadsheet(w, "Sheet1", 0, nil)
		case strings.HasPrefix(r.URL.Path, "/v4/spreadsheets/sheet-1/values/"):
			_ = json.NewEncoder(w).Encode(map[string]any{"values": []any{}})
		case r.URL.Path == "/v4/spreadsheets/sheet-1:batchUpdate":
			batches = append(batches, readBatch(t, r))
			writeBatchReply(w)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	})

	assert.NoError(t, err)
	assert.Equal(t, "https://docs.google.com/spreadsheets/d/sheet-1/edit\n", stdout)
	assertBatchContains(t, batches, "updateSheetProperties", `"sheetId":0`, `"title":"gsheet_1"`)
	assertBatchContains(t, batches, "pasteData", `"data":"name,count\nalpha,1\n"`)
}

func TestUpCommandReplaceClearsExistingSheet(t *testing.T) {
	csvPath := writeCSV(t, "name,count\nalpha,1\n")
	batches := []map[string]any{}

	err, _, _ := testCommand(t, &UpCmd{Spreadsheet: "Budget", CSVPath: csvPath, Replace: true}, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/drive/v3/files":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"files": []map[string]string{{"id": "sheet-1", "name": "Budget"}},
			})
		case r.URL.Path == "/v4/spreadsheets/sheet-1":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"sheets": []map[string]any{
					{"properties": map[string]any{"sheetId": 0, "title": "Sheet1"}},
					{"properties": map[string]any{"sheetId": 7, "title": "gsheet_up"}},
				},
			})
		case r.URL.Path == "/v4/spreadsheets/sheet-1:batchUpdate":
			batches = append(batches, readBatch(t, r))
			writeBatchReply(w)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	})

	assert.NoError(t, err)
	assertBatchContains(t, batches, "updateCells", `"sheetId":7`, `"fields":"*"`)
	assertBatchContains(t, batches, "pasteData", `"type":"PASTE_NORMAL"`)
}

func TestUpCommandRefillMergesAndExtendsFormulas(t *testing.T) {
	csvPath := writeCSV(t, "id,name\na,Ada\nb,Bob\nc,Cyd\nd,Dee\n")
	batches := []map[string]any{}

	err, _, _ := testCommand(t, &UpCmd{Spreadsheet: "Budget", CSVPath: csvPath, Refill: true}, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/drive/v3/files":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"files": []map[string]string{{"id": "sheet-1", "name": "Budget"}},
			})
		case r.URL.Path == "/v4/spreadsheets/sheet-1":
			if r.URL.Query().Get("includeGridData") == "true" {
				writeSpreadsheet(w, "gsheet_up", 7, refillGridData())
			} else {
				writeSpreadsheet(w, "gsheet_up", 7, nil)
			}
		case strings.HasPrefix(r.URL.Path, "/v4/spreadsheets/sheet-1/values/"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"values": [][]string{
					{"id", "calc"},
					{"a", "1"},
					{"b", "2"},
					{"c", "3"},
				},
			})
		case r.URL.Path == "/v4/spreadsheets/sheet-1:batchUpdate":
			batches = append(batches, readBatch(t, r))
			writeBatchReply(w)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	})

	assert.NoError(t, err)
	assertBatchContains(t, batches, "pasteData", `"type":"PASTE_VALUES"`)
	assertBatchContains(t, batches, "pasteData", `"data":"id,calc,name\na,=A2,Ada\nb,=A3,Bob\nc,=A4,Cyd\nd,,Dee\n"`)
	assertBatchContains(t, batches, "copyPaste", `"pasteType":"PASTE_FORMULA"`, `"endRowIndex":5`)
}

func TestUpCommandCreatesSpreadsheet(t *testing.T) {
	csvPath := writeCSV(t, "name,count\nalpha,1\n")
	var createBody map[string]any

	err, stdout, _ := testCommand(t, &UpCmd{Spreadsheet: "New Budget", CSVPath: csvPath}, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/drive/v3/files" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{"files": []any{}})
		case r.URL.Path == "/drive/v3/files" && r.Method == http.MethodPost:
			err := json.NewDecoder(r.Body).Decode(&createBody)
			assert.NoError(t, err)
			_ = json.NewEncoder(w).Encode(map[string]string{"id": "sheet-new", "name": "New Budget"})
		case r.URL.Path == "/v4/spreadsheets/sheet-new":
			writeSpreadsheet(w, "Sheet1", 0, nil)
		case strings.HasPrefix(r.URL.Path, "/v4/spreadsheets/sheet-new/values/"):
			_ = json.NewEncoder(w).Encode(map[string]any{"values": []any{}})
		case r.URL.Path == "/v4/spreadsheets/sheet-new:batchUpdate":
			writeBatchReply(w)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	})

	assert.NoError(t, err)
	assert.Equal(t, "New Budget", createBody["name"])
	assert.Equal(t, "application/vnd.google-apps.spreadsheet", createBody["mimeType"])
	assert.Equal(t, "https://docs.google.com/spreadsheets/d/sheet-new/edit\n", stdout)
}

func TestUpCommandAppliesFilterNumericAndLayout(t *testing.T) {
	csvPath := writeCSV(t, "name,count,rate\nalpha,1,1.25\nbeta,2,2.5\n")
	batches := []map[string]any{}

	err, _, _ := testCommand(t, &UpCmd{
		Spreadsheet: "Budget",
		CSVPath:     csvPath,
		Filter:      true,
		Numeric:     true,
		Layout:      true,
	}, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/drive/v3/files":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"files": []map[string]string{{"id": "sheet-1", "name": "Budget"}},
			})
		case r.URL.Path == "/v4/spreadsheets/sheet-1":
			if r.URL.Query().Get("includeGridData") == "true" {
				writeSpreadsheet(w, "gsheet_1", 0, layoutGridData())
			} else {
				writeSpreadsheet(w, "Sheet1", 0, nil)
			}
		case strings.HasPrefix(r.URL.Path, "/v4/spreadsheets/sheet-1/values/"):
			_ = json.NewEncoder(w).Encode(map[string]any{"values": []any{}})
		case r.URL.Path == "/v4/spreadsheets/sheet-1:batchUpdate":
			batches = append(batches, readBatch(t, r))
			writeBatchReply(w)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	})

	assert.NoError(t, err)
	assertBatchContains(t, batches, "setBasicFilter", `"endRowIndex":3`, `"endColumnIndex":3`)
	assertBatchContains(t, batches, "repeatCell", `"pattern":"#,##0"`)
	assertBatchContains(t, batches, "repeatCell", `"pattern":"#,##0.00"`)
	assertBatchContains(t, batches, "autoResizeDimensions", `"dimension":"COLUMNS"`)
	assertBatchContains(t, batches, "updateDimensionProperties", `"pixelSize":140`)
}

func TestUpCommandNumericSkipsLeadingZeroColumns(t *testing.T) {
	csvPath := writeCSV(t, "zip,ratio\n01234,0.123\n98765,0.5\n")
	batches := []map[string]any{}

	err, _, _ := testCommand(t, &UpCmd{
		Spreadsheet: "Budget",
		CSVPath:     csvPath,
		Numeric:     true,
	}, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/drive/v3/files":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"files": []map[string]string{{"id": "sheet-1", "name": "Budget"}},
			})
		case r.URL.Path == "/v4/spreadsheets/sheet-1":
			writeSpreadsheet(w, "Sheet1", 0, nil)
		case strings.HasPrefix(r.URL.Path, "/v4/spreadsheets/sheet-1/values/"):
			_ = json.NewEncoder(w).Encode(map[string]any{"values": []any{}})
		case r.URL.Path == "/v4/spreadsheets/sheet-1:batchUpdate":
			batches = append(batches, readBatch(t, r))
			writeBatchReply(w)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	})

	assert.NoError(t, err)
	assertBatchContains(t, batches, "repeatCell", `"startColumnIndex":1`, `"pattern":"#,##0.000"`)
	assertBatchMissing(t, batches, "repeatCell", `"startColumnIndex":0`)
}

func writeCSV(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "upload.csv")
	err := os.WriteFile(path, []byte(content), 0o644)
	assert.NoError(t, err)
	return path
}

func readBatch(t *testing.T, r *http.Request) map[string]any {
	t.Helper()

	var body map[string]any
	err := json.NewDecoder(r.Body).Decode(&body)
	assert.NoError(t, err)
	return body
}

func writeBatchReply(w http.ResponseWriter) {
	_ = json.NewEncoder(w).Encode(map[string]any{
		"replies": []map[string]any{
			{"addSheet": map[string]any{"properties": map[string]any{"sheetId": 9, "title": "gsheet_1"}}},
		},
	})
}

func writeSpreadsheet(w http.ResponseWriter, title string, sheetID int, data any) {
	sheet := map[string]any{"properties": map[string]any{"sheetId": sheetID, "title": title}}
	if data != nil {
		sheet["basicFilter"] = map[string]any{"range": map[string]any{"sheetId": sheetID, "endRowIndex": 4}}
		sheet["data"] = data
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"sheets": []map[string]any{sheet}})
}

func refillGridData() []map[string]any {
	return []map[string]any{{
		"rowData": []map[string]any{
			{"values": []map[string]any{
				{"userEnteredValue": map[string]string{"stringValue": "id"}},
				{"userEnteredValue": map[string]string{"stringValue": "calc"}},
			}},
			{"values": []map[string]any{
				{"userEnteredValue": map[string]string{"stringValue": "a"}},
				{"userEnteredValue": map[string]string{"formulaValue": "=A2"}},
			}},
			{"values": []map[string]any{
				{"userEnteredValue": map[string]string{"stringValue": "b"}},
				{"userEnteredValue": map[string]string{"formulaValue": "=A3"}},
			}},
			{"values": []map[string]any{
				{"userEnteredValue": map[string]string{"stringValue": "c"}},
				{"userEnteredValue": map[string]string{"formulaValue": "=A4"}},
			}},
		},
	}}
}

func layoutGridData() []map[string]any {
	return []map[string]any{{
		"columnMetadata": []map[string]any{
			{"pixelSize": 120},
			{"pixelSize": 80},
			{"pixelSize": 100},
		},
	}}
}

func assertBatchContains(t *testing.T, batches []map[string]any, requestName string, snippets ...string) {
	t.Helper()

	for _, batch := range batches {
		raw, err := json.Marshal(batch)
		assert.NoError(t, err)
		text := string(raw)
		if !strings.Contains(text, requestName) {
			continue
		}
		missing := false
		for _, snippet := range snippets {
			if !strings.Contains(text, snippet) {
				missing = true
			}
		}
		if !missing {
			return
		}
	}
	t.Fatalf("batch request %q missing snippets %v in %#v", requestName, snippets, batches)
}

func assertBatchMissing(t *testing.T, batches []map[string]any, requestName string, snippets ...string) {
	t.Helper()

	for _, batch := range batches {
		raw, err := json.Marshal(batch)
		assert.NoError(t, err)
		text := string(raw)
		if !strings.Contains(text, requestName) {
			continue
		}
		found := true
		for _, snippet := range snippets {
			if !strings.Contains(text, snippet) {
				found = false
			}
		}
		if found {
			t.Fatalf("batch request %q unexpectedly had snippets %v in %#v", requestName, snippets, batch)
		}
	}
}
