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

func newSheetRefill(sheet *uploadSheet) (*sheetRefill, error) {
	// values
	remoteRows, err := sheet.client.GetRows(sheet.ctx, sheet.fileID, sheet.title)
	if err != nil {
		return nil, err
	}

	// formulas, filters, formats, etc
	spreadsheet, err := sheet.client.GetSpreadsheetWithGridData(sheet.ctx, sheet.fileID, sheet.title)
	if err != nil {
		return nil, err
	}
	remoteSheetData := spreadsheet.Data[sheet.id]

	return &sheetRefill{
		sheet:           sheet,
		localRows:       sheet.rows,
		remoteRows:      remoteRows,
		remoteUserRows:  userEnteredRows(remoteSheetData),
		remoteSheetData: remoteSheetData,
	}, nil
}

//
// merge!
//

func (r *sheetRefill) mergedRows() (google.Rows, error) {
	if len(r.remoteRows) == 0 {
		return r.localRows, nil
	}

	// build final array of headers
	localHeaders, remoteHeaders := r.localRows[0], r.remoteRows[0]
	if err := r.validateHeaders(localHeaders, "csv"); err != nil {
		return nil, err
	}
	if err := r.validateHeaders(remoteHeaders, "existing sheet"); err != nil {
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

	merged := make(google.Rows, max(len(r.remoteRows), len(r.localRows)))
	for r := range merged {
		merged[r] = make([]string, len(headers))
	}

	// append remote
	for ii, row := range r.remoteUserRows {
		copy(merged[ii], row)
	}

	// append local (w/ header remap)
	for lc, header := range localHeaders {
		c := util.IndexOfString(headers, header)
		for r, local := range r.localRows {
			merged[r][c] = local[lc]
		}
	}

	return merged, nil
}

//
// extend formats, formulas, and clears padding formats.
//

func (r *sheetRefill) extend() error {
	requests := []google.Request{}
	nrows := r.remoteDataRowCount()
	if len(r.sheet.rows) > nrows && nrows >= 2 {
		// copy FORMATS
		requests = append(requests, r.copyRequests(r.remoteCSVColumns(), 2, "PASTE_FORMAT")...)

		// copy FORMULAS
		formulaColumns, err := r.formulaColumns(nrows)
		if err != nil {
			return err
		}
		requests = append(requests, r.copyRequests(formulaColumns, nrows, "PASTE_FORMULA")...)
	}

	// clear out the padding rows & cols
	requests = append(requests, r.clearPaddingRequests()...)

	// do it
	_, err := r.sheet.client.BatchUpdate(r.sheet.ctx, r.sheet.fileID, requests)
	if err != nil {
		return err
	}

	return nil
}

//
// build CopyPaste Requests for extending cols
//

func (r *sheetRefill) copyRequests(columns []int, endRow int, pasteType string) []google.Request {
	requests := make([]google.Request, 0, len(columns))
	for _, c := range columns {
		requests = append(requests, google.Request{
			CopyPaste: &google.CopyPasteRequest{
				Source: google.GridRange{
					SheetID:          r.sheet.id,
					StartRowIndex:    1,
					EndRowIndex:      endRow,
					StartColumnIndex: c,
					EndColumnIndex:   c + 1,
				},
				Destination: google.GridRange{
					SheetID:          r.sheet.id,
					StartRowIndex:    1,
					EndRowIndex:      len(r.sheet.rows),
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

func (r *sheetRefill) clearPaddingRequests() []google.Request {
	w, h := len(r.sheet.rows[0]), len(r.sheet.rows)
	return []google.Request{
		{
			RepeatCell: &google.RepeatCellRequest{
				Range: google.GridRange{
					SheetID:          r.sheet.id,
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
					SheetID:          r.sheet.id,
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
func (r *sheetRefill) formulaColumns(remoteRows int) ([]int, error) {
	remoteCSV := map[int]bool{}
	for _, c := range r.remoteCSVColumns() {
		remoteCSV[c] = true
	}
	columns := []int{}
	for c := range r.remoteRows[0] {
		if remoteCSV[c] {
			continue
		}
		formulaColumn, err := r.formulaColumn(c, remoteRows)
		if err != nil {
			return nil, err
		}
		if formulaColumn {
			columns = append(columns, c)
		}
	}
	return columns, nil
}

// formulaColumn reports whether a non-CSV column should be formula-extended.
func (r *sheetRefill) formulaColumn(columnIndex int, remoteRows int) (bool, error) {
	sawFormula := false
	for ii := 1; ii < remoteRows; ii++ {
		value := r.remoteRows[ii][columnIndex]
		if value == "" {
			continue
		}
		if ii >= len(r.remoteSheetData.Rows) {
			return false, fmt.Errorf("missing grid data for sheet %s row %d", r.sheet.title, ii+1)
		}
		if columnIndex >= len(r.remoteSheetData.Rows[ii].Values) {
			return false, fmt.Errorf("missing grid data for sheet %s row %d column %d", r.sheet.title, ii+1, columnIndex+1)
		}
		cell := r.remoteSheetData.Rows[ii].Values[columnIndex]
		if cell.UserEnteredValue == nil || cell.UserEnteredValue.FormulaValue == nil || *cell.UserEnteredValue.FormulaValue == "" {
			return false, nil
		}
		sawFormula = true
	}
	return sawFormula, nil
}

// remoteDataRowCount returns remote rows covered by the filter or data.
func (r *sheetRefill) remoteDataRowCount() int {
	count := len(r.remoteRows)
	if r.remoteSheetData.BasicFilter != nil && r.remoteSheetData.BasicFilter.Range.EndRowIndex > 0 {
		count = r.remoteSheetData.BasicFilter.Range.EndRowIndex
	}
	return min(count, len(r.remoteRows))
}

// remoteCSVColumns returns remote columns that also appear in the CSV.
func (r *sheetRefill) remoteCSVColumns() []int {
	csvHeaders := map[string]bool{}
	for _, header := range r.localRows[0] {
		csvHeaders[header] = true
	}
	columns := []int{}
	for c, header := range r.remoteRows[0] {
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
func (r *sheetRefill) validateHeaders(headers []string, label string) error {
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
