package commands

// This file owns `up --refill`: merge CSV rows into a remote sheet while
// preserving remote user-entered values, formats, and formula columns.

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gurgeous/gshoot/google"
	"github.com/gurgeous/gshoot/util"
)

type sheetRefill struct {
	sheet           *uploadSheet      // target sheet being refilled
	localRows       google.Rows       // csv rows from disk
	remoteRows      google.Rows       // remote displayed values
	remoteUserRows  google.Rows       // remote user-entered values
	remoteSheetData *google.SheetData // remote grid data for formats/formulas
}

//
// loads remote sheet data before we refill
//

func newSheetRefill(u *uploadSheet) (*sheetRefill, error) {
	s := &sheetRefill{sheet: u, localRows: u.rows}

	// fetch values
	var err error
	s.remoteRows, err = u.client.GetRows(u.ctx, u.fileID, u.title)
	if err != nil {
		return nil, err
	}

	// fetch "grid data" - formulas, filters, formats, etc
	spreadsheet, err := u.client.GetSpreadsheetWithGridData(u.ctx, u.fileID, u.title)
	if err != nil {
		return nil, err
	}
	s.remoteSheetData = spreadsheet.Data[u.id]
	s.remoteUserRows = userEnteredRows(s.remoteSheetData)

	return s, nil
}

//
// merge!
//

func (s *sheetRefill) mergedRows() (google.Rows, error) {
	if len(s.remoteRows) == 0 {
		return s.localRows, nil
	}

	// build final array of headers
	localHeaders, remoteHeaders := s.localRows[0], s.remoteRows[0]
	if err := s.validateHeaders(localHeaders, "csv"); err != nil {
		return nil, err
	}
	if err := s.validateHeaders(remoteHeaders, "existing sheet"); err != nil {
		return nil, err
	}
	headers := append([]string(nil), remoteHeaders...)
	for _, header := range localHeaders {
		if !util.ContainsString(headers, header) {
			headers = append(headers, header)
		}
	}

	//
	// now merge rows
	//

	merged := make(google.Rows, max(len(s.remoteRows), len(s.localRows)))
	for r := range merged {
		merged[r] = make([]string, len(headers))
	}

	// append remote
	for ii, row := range s.remoteUserRows {
		copy(merged[ii], row)
	}

	// append local (w/ header remap)
	for lc, header := range localHeaders {
		c := util.IndexOfString(headers, header)
		for r, local := range s.localRows {
			merged[r][c] = local[lc]
		}
	}

	return merged, nil
}

//
// extend formats, formulas, and clears padding formats.
//

func (s *sheetRefill) extend() error {
	requests := []google.Request{}
	nrows := s.remoteDataRowCount()
	if len(s.sheet.rows) > nrows && nrows >= 2 {
		// copy FORMATS
		requests = append(requests, s.copyRequests(s.sharedRemoteColumns(), 2, "PASTE_FORMAT")...)

		// copy FORMULAS
		formulaColumns := s.formulaColumns(nrows)
		requests = append(requests, s.copyRequests(formulaColumns, nrows, "PASTE_FORMULA")...)
	}

	// clear out the padding rows & cols
	requests = append(requests, s.clearPaddingRequests()...)

	// do it
	_, err := s.sheet.client.BatchUpdate(s.sheet.ctx, s.sheet.fileID, requests)
	if err != nil {
		return err
	}

	return nil
}

//
// build CopyPaste Requests for extending cols
//

func (s *sheetRefill) copyRequests(columns []int, endRow int, pasteType string) []google.Request {
	requests := make([]google.Request, 0, len(columns))
	for _, c := range columns {
		requests = append(requests, google.Request{
			CopyPaste: &google.CopyPasteRequest{
				Source: google.GridRange{
					SheetID:          s.sheet.id,
					StartRowIndex:    1,
					EndRowIndex:      endRow,
					StartColumnIndex: c,
					EndColumnIndex:   c + 1,
				},
				Destination: google.GridRange{
					SheetID:          s.sheet.id,
					StartRowIndex:    1,
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

//
// clears formatting outside the refilled data area.
//

func (s *sheetRefill) clearPaddingRequests() []google.Request {
	w, h := len(s.sheet.rows[0]), len(s.sheet.rows)
	return []google.Request{
		{
			RepeatCell: &google.RepeatCellRequest{
				Range: google.GridRange{
					SheetID:          s.sheet.id,
					StartRowIndex:    h,
					EndRowIndex:      h + gridPadding,
					StartColumnIndex: 0,
					EndColumnIndex:   w + gridPadding,
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
					EndRowIndex:      h,
					StartColumnIndex: w,
					EndColumnIndex:   w + gridPadding,
				},
				Cell:   google.CellData{UserEnteredFormat: &google.CellFormat{}},
				Fields: "userEnteredFormat",
			},
		},
	}
}

// formulaColumns returns non-CSV columns that contain formulas.
func (s *sheetRefill) formulaColumns(remoteRows int) []int {
	set := map[int]bool{}
	for _, c := range s.sharedRemoteColumns() {
		set[c] = true
	}

	columns := []int{}
	for c := range s.remoteRows[0] {
		if set[c] {
			continue
		}
		if s.formulaColumn(c, remoteRows) {
			columns = append(columns, c)
		}
	}
	return columns
}

// formulaColumn reports whether a non-CSV column should be formula-extended.
func (s *sheetRefill) formulaColumn(c int, remoteRows int) bool {
	sawFormula := false
	for r := 1; r < remoteRows; r++ {
		value := s.remoteRows[r][c]
		if value == "" {
			continue
		}
		if r >= len(s.remoteSheetData.Rows) || c >= len(s.remoteSheetData.Rows[r].Values) {
			return false
		}
		cell := s.remoteSheetData.Rows[r].Values[c]
		if cell.UserEnteredValue == nil || cell.UserEnteredValue.FormulaValue == nil || *cell.UserEnteredValue.FormulaValue == "" {
			return false
		}
		sawFormula = true
	}
	return sawFormula
}

// remoteDataRowCount returns remote rows covered by the filter or data.
func (s *sheetRefill) remoteDataRowCount() int {
	count := len(s.remoteRows)
	if s.remoteSheetData.BasicFilter != nil && s.remoteSheetData.BasicFilter.Range.EndRowIndex > 0 {
		count = s.remoteSheetData.BasicFilter.Range.EndRowIndex
	}
	return min(count, len(s.remoteRows))
}

// returns columns that are both local and remote
func (s *sheetRefill) sharedRemoteColumns() []int {
	csvHeaders := map[string]bool{}
	for _, header := range s.localRows[0] {
		csvHeaders[header] = true
	}
	columns := []int{}
	for c, header := range s.remoteRows[0] {
		if csvHeaders[header] {
			columns = append(columns, c)
		}
	}
	return columns
}

//
// helpers
//

// userEnteredRows extracts user-entered strings from grid data.
func userEnteredRows(data *google.SheetData) google.Rows {
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
func (s *sheetRefill) validateHeaders(headers []string, label string) error {
	counts := map[string]int{}
	for _, header := range headers {
		if header != "" {
			counts[header]++
		}
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
