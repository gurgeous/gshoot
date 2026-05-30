package commands

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/gurgeous/gshoot/google"
	"github.com/gurgeous/gshoot/util"
)

type refillUpload struct {
	ctx               context.Context
	client            *google.Client
	fileID            string
	sheetID           int64
	sheetTitle        string
	csvRows           google.Rows
	existingRows      google.Rows
	existingUserRows  google.Rows
	existingSheetData *google.SheetData
}

// newRefillUpload loads existing sheet data needed for refill.
func newRefillUpload(
	ctx context.Context,
	client *google.Client,
	fileID string,
	sheetID int64,
	sheetTitle string,
	csvRows google.Rows,
) (*refillUpload, error) {
	existingRows, err := client.GetRows(ctx, fileID, sheetTitle)
	if err != nil {
		return nil, err
	}

	refreshed, err := client.GetSpreadsheetWithGridData(ctx, fileID, sheetTitle)
	if err != nil {
		return nil, err
	}
	existingSheetData := refreshed.Data[sheetID]
	return &refillUpload{
		ctx:               ctx,
		client:            client,
		fileID:            fileID,
		sheetID:           sheetID,
		sheetTitle:        sheetTitle,
		csvRows:           csvRows,
		existingRows:      existingRows,
		existingUserRows:  userEnteredRows(existingSheetData),
		existingSheetData: existingSheetData,
	}, nil
}

// rows merges CSV rows into the existing sheet shape.
func (r *refillUpload) rows() (google.Rows, error) {
	if len(r.existingRows) == 0 {
		return r.csvRows, nil
	}

	existingHeaders := r.existingRows[0]
	csvHeaders := r.csvRows[0]
	if err := r.validateHeaders(existingHeaders, "existing sheet"); err != nil {
		return nil, err
	}
	if err := r.validateHeaders(csvHeaders, "csv"); err != nil {
		return nil, err
	}

	finalHeaders := append([]string(nil), existingHeaders...)
	for _, header := range csvHeaders {
		if !util.ContainsString(finalHeaders, header) {
			finalHeaders = append(finalHeaders, header)
		}
	}
	byHeader := map[string]int{}
	for i, header := range finalHeaders {
		byHeader[header] = i
	}

	finalRowCount := max(len(r.existingRows), len(r.csvRows))
	merged := make(google.Rows, finalRowCount)
	for ii := range merged {
		merged[ii] = make([]string, len(finalHeaders))
	}
	for ii, row := range r.existingUserRows {
		copy(merged[ii], row)
	}
	for csvColumnIndex, header := range csvHeaders {
		targetColumnIndex := byHeader[header]
		for ii, row := range r.csvRows {
			merged[ii][targetColumnIndex] = row[csvColumnIndex]
		}
	}
	return merged, nil
}

// apply copies refill formats, formulas, and clears padding formats.
func (r *refillUpload) apply(sheet *uploadSheet) error {
	if err := r.applyFormats(sheet); err != nil {
		return err
	}
	if err := r.applyFormulas(sheet); err != nil {
		return err
	}
	return r.clearPaddingFormats(sheet)
}

// applyFormats copies existing data-row formats across refilled columns.
func (r *refillUpload) applyFormats(sheet *uploadSheet) error {
	if len(r.existingRows) < 2 || len(sheet.rows) <= 1 {
		return nil
	}
	requests := []google.Request{}
	for _, columnIndex := range r.existingCSVColumns() {
		requests = append(requests, google.Request{
			CopyPaste: &google.CopyPasteRequest{
				Source: google.GridRange{
					SheetID:          sheet.id,
					StartRowIndex:    1,
					EndRowIndex:      2,
					StartColumnIndex: columnIndex,
					EndColumnIndex:   columnIndex + 1,
				},
				Destination: google.GridRange{
					SheetID:          sheet.id,
					StartRowIndex:    1,
					EndRowIndex:      len(sheet.rows),
					StartColumnIndex: columnIndex,
					EndColumnIndex:   columnIndex + 1,
				},
				PasteType:        "PASTE_FORMAT",
				PasteOrientation: "NORMAL",
			},
		})
	}
	if len(requests) == 0 {
		return nil
	}
	_, err := r.client.BatchUpdate(r.ctx, r.fileID, requests)
	return err
}

// applyFormulas extends non-CSV formula columns during refill.
func (r *refillUpload) applyFormulas(sheet *uploadSheet) error {
	existingRows := r.existingDataRowCount()
	if len(sheet.rows) <= existingRows || existingRows < 2 {
		return nil
	}
	formulaColumns, err := r.formulaColumns(existingRows)
	if err != nil {
		return err
	}
	requests := []google.Request{}
	for _, columnIndex := range formulaColumns {
		requests = append(requests, google.Request{
			CopyPaste: &google.CopyPasteRequest{
				Source: google.GridRange{
					SheetID:          sheet.id,
					StartRowIndex:    1,
					EndRowIndex:      existingRows,
					StartColumnIndex: columnIndex,
					EndColumnIndex:   columnIndex + 1,
				},
				Destination: google.GridRange{
					SheetID:          sheet.id,
					StartRowIndex:    1,
					EndRowIndex:      len(sheet.rows),
					StartColumnIndex: columnIndex,
					EndColumnIndex:   columnIndex + 1,
				},
				PasteType:        "PASTE_FORMULA",
				PasteOrientation: "NORMAL",
			},
		})
	}
	if len(requests) == 0 {
		return nil
	}
	_, err = r.client.BatchUpdate(r.ctx, r.fileID, requests)
	return err
}

// clearPaddingFormats clears formatting outside the refilled data area.
func (r *refillUpload) clearPaddingFormats(sheet *uploadSheet) error {
	_, err := r.client.BatchUpdate(r.ctx, r.fileID, []google.Request{
		{
			RepeatCell: &google.RepeatCellRequest{
				Range: google.GridRange{
					SheetID:          sheet.id,
					StartRowIndex:    len(sheet.rows),
					EndRowIndex:      len(sheet.rows) + gridPadding,
					StartColumnIndex: 0,
					EndColumnIndex:   len(sheet.rows[0]) + gridPadding,
				},
				Cell:   google.CellData{UserEnteredFormat: &google.CellFormat{}},
				Fields: "userEnteredFormat",
			},
		},
		{
			RepeatCell: &google.RepeatCellRequest{
				Range: google.GridRange{
					SheetID:          sheet.id,
					StartRowIndex:    0,
					EndRowIndex:      len(sheet.rows),
					StartColumnIndex: len(sheet.rows[0]),
					EndColumnIndex:   len(sheet.rows[0]) + gridPadding,
				},
				Cell:   google.CellData{UserEnteredFormat: &google.CellFormat{}},
				Fields: "userEnteredFormat",
			},
		},
	})
	return err
}

// formulaColumns returns non-CSV columns that contain formulas.
func (r *refillUpload) formulaColumns(existingRows int) ([]int, error) {
	existingCSV := map[int]bool{}
	for _, columnIndex := range r.existingCSVColumns() {
		existingCSV[columnIndex] = true
	}
	columns := []int{}
	for columnIndex := range r.existingRows[0] {
		if existingCSV[columnIndex] {
			continue
		}
		formulaColumn, err := r.formulaColumn(columnIndex, existingRows)
		if err != nil {
			return nil, err
		}
		if formulaColumn {
			columns = append(columns, columnIndex)
		}
	}
	return columns, nil
}

// formulaColumn reports whether a non-CSV column should be formula-extended.
func (r *refillUpload) formulaColumn(columnIndex int, existingRows int) (bool, error) {
	sawFormula := false
	for ii := 1; ii < existingRows; ii++ {
		value := r.existingRows[ii][columnIndex]
		if value == "" {
			continue
		}
		if r.existingSheetData == nil {
			return false, fmt.Errorf("missing grid data for sheet %s", r.sheetTitle)
		}
		if ii >= len(r.existingSheetData.Rows) {
			return false, fmt.Errorf("missing grid data for sheet %s row %d", r.sheetTitle, ii+1)
		}
		if columnIndex >= len(r.existingSheetData.Rows[ii].Values) {
			return false, fmt.Errorf("missing grid data for sheet %s row %d column %d", r.sheetTitle, ii+1, columnIndex+1)
		}
		cell := r.existingSheetData.Rows[ii].Values[columnIndex]
		if cell.UserEnteredValue == nil || cell.UserEnteredValue.FormulaValue == nil || *cell.UserEnteredValue.FormulaValue == "" {
			return false, nil
		}
		sawFormula = true
	}
	return sawFormula, nil
}

// existingDataRowCount returns the existing rows covered by the filter or data.
func (r *refillUpload) existingDataRowCount() int {
	count := len(r.existingRows)
	if r.existingSheetData != nil && r.existingSheetData.BasicFilter != nil && r.existingSheetData.BasicFilter.Range.EndRowIndex > 0 {
		count = r.existingSheetData.BasicFilter.Range.EndRowIndex
	}
	return min(count, len(r.existingRows))
}

// existingCSVColumns returns existing sheet columns that also appear in the CSV.
func (r *refillUpload) existingCSVColumns() []int {
	csvHeaders := map[string]bool{}
	for _, header := range r.csvRows[0] {
		csvHeaders[header] = true
	}
	columns := []int{}
	for columnIndex, header := range r.existingRows[0] {
		if csvHeaders[header] {
			columns = append(columns, columnIndex)
		}
	}
	return columns
}

// userEnteredRows extracts user-entered strings from grid data.
func userEnteredRows(data *google.SheetData) google.Rows {
	if data == nil {
		return nil
	}
	rows := google.Rows{}
	for _, row := range data.Rows {
		values := []string{}
		for _, cell := range row.Values {
			values = append(values, cell.UserEnteredString())
		}
		rows = append(rows, values)
	}
	return google.Rows(util.CSVRectangularize(rows))
}

// validateHeaders rejects duplicate non-empty headers.
func (r *refillUpload) validateHeaders(headers []string, label string) error {
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
