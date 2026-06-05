package commands

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gurgeous/gshoot/google"
	"github.com/gurgeous/gshoot/util"
)

//
// up --refill
//
// GLOSSARY
//
// local: CSV data being uploaded.
// remote: existing spreadsheet data.
// grid data: remote cells fetched with formulas, formats, filters, and metadata.
// grid row: remote row from grid data, preserving formulas instead of display values.
// shared column: remote column whose header exists in the CSV.
// remote-only column: remote column whose header does not exist in the CSV.
// remote col: code shorthand for a remote-only column index.
// paste header: output header after keeping remote headers and appending new CSV headers.
// CSV height: number of CSV rows, including the header.
// stale row: remote row below CSV height.
// stale value: value in a shared column on a stale row.
// shrink: reduce sheet row count after refill.
// shrink-safe: remote-only columns are blank in stale rows.
// padding: extra rows/cols gshoot keeps around the data.
//

type refiller struct {
	// target sheet being refilled
	sheet *uploader

	// local CSV data
	localHeaders []string    // CSV header row
	localRows    google.Rows // CSV rows from disk

	// remote file
	remoteHeaders   []string          // remote header row
	remoteRows      google.Rows       // remote display values from the values API
	remoteGridRows  google.Rows       // grid rows, preserving formulas
	remoteSheetData *google.SheetData // grid data for formulas, formats, filters, metadata
	remoteCols      []int             // remote-only column indexes

	pasteHeaders []string // remote headers plus new CSV headers
	sharedCols   []int    // remote column indexes also present in the CSV
}

//
// load remote data and derive refill mappings
//

func newRefiller(u *uploader) (*refiller, error) {
	s := &refiller{sheet: u, localRows: u.rows}

	//
	// local
	//

	s.localHeaders = s.localRows[0]
	if err := validateHeaders(s.localHeaders, "csv"); err != nil {
		return nil, err
	}

	//
	// fetch remote
	//

	// rows/headers
	var err error
	s.remoteRows, err = u.client.GetRows(u.ctx, u.file.ID, u.title)
	if err != nil {
		return nil, err
	}
	s.remoteHeaders = s.remoteRows[0]
	if err := validateHeaders(s.remoteHeaders, "existing sheet"); err != nil {
		return nil, err
	}
	if len(s.remoteRows) == 0 {
		s.pasteHeaders = append([]string(nil), s.localHeaders...)
		return s, nil
	}

	// grid data (formulas, filters, formats, etc)
	spreadsheet, err := u.client.GetSpreadsheetWithGridData(u.ctx, u.file.ID, u.title)
	if err != nil {
		return nil, err
	}
	s.remoteSheetData = spreadsheet.Data[u.id]
	s.remoteGridRows = gridRows(s.remoteSheetData)

	//
	// pasteHeaders, sharedCols and remoteCols
	//

	localHeaderSet := map[string]bool{}
	for _, header := range s.localHeaders {
		if header != "" {
			localHeaderSet[header] = true
		}
	}

	s.pasteHeaders = append([]string(nil), s.remoteHeaders...)
	for _, header := range s.localHeaders {
		if !util.ContainsString(s.pasteHeaders, header) {
			s.pasteHeaders = append(s.pasteHeaders, header)
		}
	}
	for c, header := range s.remoteHeaders {
		switch {
		case header == "":
		case localHeaderSet[header]:
			s.sharedCols = append(s.sharedCols, c)
		default:
			s.remoteCols = append(s.remoteCols, c)
		}
	}

	return s, nil
}

//
// calculate paste rows
//

func (s *refiller) pasteRows() google.Rows {
	if len(s.remoteRows) == 0 {
		return s.localRows
	}

	height := max(s.remoteHeight(), len(s.localRows))
	if s.canShrink() {
		height = len(s.localRows)
	}

	rows := make(google.Rows, height)
	for r := range rows {
		rows[r] = make([]string, len(s.pasteHeaders))
	}

	// copy remote values first so remote-only columns survive the refill
	for ii, row := range s.remoteGridRows {
		if ii >= len(rows) {
			// Grid data can include formula-only rows that display blank and are absent from the values API.
			break
		}
		copy(rows[ii], row)
	}
	for ii, row := range s.remoteRows {
		if ii >= len(rows) || ii < len(s.remoteGridRows) {
			continue
		}
		copy(rows[ii], row)
	}

	// overlay CSV values by header, then clear stale values in shared columns
	for lc, header := range s.localHeaders {
		c := util.IndexOfString(s.pasteHeaders, header)
		for r, local := range s.localRows {
			rows[r][c] = local[lc]
		}
	}
	for _, c := range s.sharedCols {
		for r := len(s.localRows); r < len(rows); r++ {
			rows[r][c] = ""
		}
	}

	return rows
}

//
// extend formulas/formats after refill and clear padding formats.
//

func (s *refiller) extend() error {
	requests := []google.Request{}
	remoteDataRows := s.remoteDataHeight()
	if len(s.sheet.rows) > remoteDataRows && remoteDataRows >= 2 {
		// extend formats
		requests = append(requests, s.extendRowsRequests(s.allColumns(), remoteDataRows, "PASTE_FORMAT")...)

		// extend formulas
		formulaColumns := s.formulaColumns()
		requests = append(requests, s.extendRowsRequests(formulaColumns, remoteDataRows, "PASTE_FORMULA")...)
	}

	requests = append(requests, s.clearStaleValueRequests()...)

	// clear padding row/column formats
	requests = append(requests, s.clearPaddingRequests()...)

	_, err := s.sheet.client.BatchUpdate(s.sheet.ctx, s.sheet.file.ID, requests)
	if err != nil {
		return err
	}

	return nil
}

//
// build CopyPaste requests from the final remote row into refilled rows.
//

func (s *refiller) extendRowsRequests(columns []int, remoteRows int, pasteType string) []google.Request {
	requests := make([]google.Request, 0, len(columns))
	sourceRow := remoteRows - 1
	for _, c := range columns {
		requests = append(requests, google.Request{
			CopyPaste: &google.CopyPasteRequest{
				Source: google.GridRange{
					SheetID:          s.sheet.id,
					StartRowIndex:    sourceRow,
					EndRowIndex:      sourceRow + 1,
					StartColumnIndex: c,
					EndColumnIndex:   c + 1,
				},
				Destination: google.GridRange{
					SheetID:          s.sheet.id,
					StartRowIndex:    remoteRows,
					EndRowIndex:      len(s.sheet.rows),
					StartColumnIndex: c,
					EndColumnIndex:   c + 1,
				},
				PasteType:        pasteType,
				PasteOrientation: "NORMAL",
			},
		})
	}
	return requests
}

func (s *refiller) clearStaleValueRequests() []google.Request {
	rowCount, colCount := len(s.sheet.rows), len(s.sheet.rows[0])
	if rowCount >= s.remoteHeight() || !s.canShrink() {
		return nil
	}
	return []google.Request{{
		UpdateCells: &google.UpdateCellsRequest{
			Range: google.GridRange{
				SheetID:          s.sheet.id,
				StartRowIndex:    rowCount,
				EndRowIndex:      rowCount + gridPadding,
				StartColumnIndex: 0,
				EndColumnIndex:   colCount + gridPadding,
			},
			Fields: "userEnteredValue",
		},
	}}
}

//
// clears formatting outside the refilled data area.
//

func (s *refiller) clearPaddingRequests() []google.Request {
	rowCount, colCount := len(s.sheet.rows), len(s.sheet.rows[0])
	return []google.Request{
		{
			RepeatCell: &google.RepeatCellRequest{
				Range: google.GridRange{
					SheetID:          s.sheet.id,
					StartRowIndex:    rowCount,
					EndRowIndex:      rowCount + gridPadding,
					StartColumnIndex: 0,
					EndColumnIndex:   colCount + gridPadding,
				},
				Cell:   google.CellData{UserEnteredFormat: &google.CellFormat{}},
				Fields: "userEnteredFormat",
			},
		},
		{
			RepeatCell: &google.RepeatCellRequest{
				Range: google.GridRange{
					SheetID:          s.sheet.id,
					StartRowIndex:    0,
					EndRowIndex:      rowCount,
					StartColumnIndex: colCount,
					EndColumnIndex:   colCount + gridPadding,
				},
				Cell:   google.CellData{UserEnteredFormat: &google.CellFormat{}},
				Fields: "userEnteredFormat",
			},
		},
	}
}

// formulaColumns returns remote-only columns that contain formulas.
func (s *refiller) formulaColumns() []int {
	ignore := map[int]bool{}
	for _, c := range s.sharedCols {
		ignore[c] = true
	}

	columns := []int{}
	for c := range s.remoteRows[0] {
		if ignore[c] {
			continue
		}
		if s.hasFormula(c) {
			columns = append(columns, c)
		}
	}
	return columns
}

// allColumns returns every column in the paste payload.
func (s *refiller) allColumns() []int {
	columns := make([]int, len(s.sheet.rows[0]))
	for c := range columns {
		columns[c] = c
	}
	return columns
}

// remoteHeight includes grid-only rows so formulas are preserved.
func (s *refiller) remoteHeight() int {
	return max(len(s.remoteRows), len(s.remoteSheetData.Rows))
}

// canShrink reports whether remote-only columns are blank in stale rows.
func (s *refiller) canShrink() bool {
	csvHeight := len(s.localRows)
	if csvHeight >= s.remoteHeight() {
		return true
	}
	for _, c := range s.remoteCols {
		for r := csvHeight; r < s.remoteHeight(); r++ {
			if s.remoteOnlyStaleValue(r, c) {
				return false
			}
		}
	}
	return true
}

func (s *refiller) remoteOnlyStaleValue(r, c int) bool {
	if r < len(s.remoteSheetData.Rows) && c < len(s.remoteSheetData.Rows[r].Values) {
		return s.remoteSheetData.Rows[r].Values[c].UserEnteredValue != nil
	}
	return r < len(s.remoteRows) && c < len(s.remoteRows[r]) && s.remoteRows[r][c] != ""
}

// hasFormula reports whether a remote-only column should be formula-extended.
func (s *refiller) hasFormula(c int) bool {
	sawFormula := false
	for r := 1; r < s.remoteDataHeight(); r++ {
		if r >= len(s.remoteSheetData.Rows) || c >= len(s.remoteSheetData.Rows[r].Values) {
			return false
		}
		cell := s.remoteSheetData.Rows[r].Values[c]
		if cell.UserEnteredValue == nil {
			continue
		}
		if cell.UserEnteredValue.FormulaValue == nil || *cell.UserEnteredValue.FormulaValue == "" {
			return false
		}
		sawFormula = true
	}
	return sawFormula
}

// remoteDataHeight returns remote rows covered by the filter or data.
func (s *refiller) remoteDataHeight() int {
	count := len(s.remoteRows)
	if s.remoteSheetData.BasicFilter != nil && s.remoteSheetData.BasicFilter.Range.EndRowIndex > 0 {
		count = s.remoteSheetData.BasicFilter.Range.EndRowIndex
	}
	return min(count, len(s.remoteRows))
}

//
// helpers
//

// gridRows extracts strings and formulas from grid data.
func gridRows(data *google.SheetData) google.Rows {
	rows := make(google.Rows, 0, len(data.Rows))
	for _, row := range data.Rows {
		values := make([]string, 0, len(row.Values))
		for _, cell := range row.Values {
			values = append(values, cell.UserEnteredString())
		}
		rows = append(rows, values)
	}
	return google.Rows(util.CSVRectangularize(rows))
}

// validateHeaders rejects duplicate headers.
func validateHeaders(headers []string, label string) error {
	counts := map[string]int{}
	for _, header := range headers {
		if header != "" {
			counts[header]++
		}
	}
	if len(counts) == 0 {
		return fmt.Errorf("%s has no headers", label)
	}
	duplicates := []string{}
	for header, count := range counts {
		if count > 1 {
			duplicates = append(duplicates, header)
		}
	}
	if len(duplicates) > 0 {
		sort.Strings(duplicates)
		return fmt.Errorf("%s has duplicate headers: %s", label, strings.Join(duplicates, ", "))
	}
	return nil
}
