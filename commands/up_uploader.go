package commands

// This file owns the target worksheet for `up`: choose/create/rename it,
// paste rows into it, and apply optional sheet-level formatting.

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/gurgeous/gshoot/google"
	"github.com/gurgeous/gshoot/util"
)

const (
	defaultUploadSheet = "gsheet_up" // default sheet name for refill/replace
	gridPadding        = 2           // empty rows/columns kept around uploaded data
	sheetPrefix        = "gsheet_"   // prefix for generated upload sheet names
	layoutPadding      = 20          // extra pixels added after auto-sizing columns
)

var (
	integerRE     = regexp.MustCompile(`\A-?\d+\z`)           // whole-number detector for numeric formatting
	decimalRE     = regexp.MustCompile(`\A-?\d+(?:\.\d+)?\z`) // decimal detector for numeric formatting
	leadingZeroRE = regexp.MustCompile(`\A-?0\d`)             // numeric-looking value that should remain text
)

//
// this does most of the work
//

type uploader struct {
	ctx         context.Context     // request context for Google calls
	client      *google.Client      // Google API client
	file        *google.File        // spreadsheet Drive file
	spreadsheet *google.Spreadsheet // current spreadsheet metadata
	cmd         *UpCmd              // upload command options
	title       string              // target sheet title
	id          int64               // target sheet ID after ensure
	rows        google.Rows         // rows to paste into the target sheet
}

func newUploader(
	ctx context.Context,
	client *google.Client,
	file *google.File,
	spreadsheet *google.Spreadsheet,
	cmd *UpCmd,
	rows google.Rows,
) *uploader {
	return &uploader{
		ctx:         ctx,
		client:      client,
		file:        file,
		spreadsheet: spreadsheet,
		cmd:         cmd,
		title:       sheetTitle(cmd, spreadsheet),
		rows:        rows,
	}
}

// resolveTargetSheet selects, creates, or renames the target sheet.
func (s *uploader) resolveTargetSheet() error {
	// already present?
	for _, sheet := range s.spreadsheet.Sheets {
		if strings.EqualFold(sheet.Title, s.title) {
			s.id = sheet.ID
			return nil
		}
	}

	// is this an empty file?
	isEmpty, err := s.isEmpty()
	if err != nil {
		return err
	}

	if s.cmd.Refill || s.cmd.Replace {
		if isEmpty {
			return s.renameSheet(s.spreadsheet.Sheets[0].ID)
		}
		return s.addSheet()
	}
	if s.cmd.Sheet != "" || !isEmpty {
		return s.addSheet()
	}
	return s.renameSheet(s.spreadsheet.Sheets[0].ID)
}

// addSheet creates the target sheet and records its ID.
func (s *uploader) addSheet() error {
	res, err := s.client.BatchUpdate(s.ctx, s.file.ID, []google.Request{{
		AddSheet: &google.AddSheetRequest{
			Properties: google.SheetProperties{
				Title: s.title,
				Index: new(0),
				GridProperties: &google.GridProperties{
					RowCount:    len(s.rows) + gridPadding,
					ColumnCount: len(s.rows[0]) + gridPadding,
				},
			},
		},
	}})
	if err != nil {
		return err
	}
	s.id = res.Replies[0].AddSheet.Properties.ID
	return nil
}

// renameSheet renames an existing sheet and records it as the target.
func (s *uploader) renameSheet(sheetID int64) error {
	_, err := s.client.BatchUpdate(s.ctx, s.file.ID, []google.Request{{
		UpdateSheetProperties: &google.UpdateSheetPropertiesRequest{
			Properties: google.SheetProperties{
				SheetID: new(sheetID),
				Title:   s.title,
			},
			Fields: "title",
		},
	}})
	s.id = sheetID
	return err
}

// clearSheet clears every cell in the target sheet.
func (s *uploader) clearSheet() error {
	_, err := s.client.BatchUpdate(s.ctx, s.file.ID, []google.Request{{
		UpdateCells: &google.UpdateCellsRequest{
			Range:  google.GridRange{SheetID: s.id},
			Fields: "*",
		},
	}})
	return err
}

// growSheet expands the target grid to fit data plus padding.
func (s *uploader) growSheet() error {
	_, err := s.client.BatchUpdate(s.ctx, s.file.ID, []google.Request{{
		UpdateSheetProperties: &google.UpdateSheetPropertiesRequest{
			Properties: google.SheetProperties{
				SheetID: new(s.id),
				GridProperties: &google.GridProperties{
					RowCount:    len(s.rows) + gridPadding,
					ColumnCount: len(s.rows[0]) + gridPadding,
				},
			},
			Fields: "gridProperties.rowCount,gridProperties.columnCount",
		},
	}})
	return err
}

// pasteCSV pastes CSV data into the target sheet.
func (s *uploader) pasteCSV() error {
	pasteType := "PASTE_NORMAL"
	if s.cmd.Refill {
		pasteType = "PASTE_VALUES"
	}
	_, err := s.client.BatchUpdate(s.ctx, s.file.ID, []google.Request{{
		PasteData: &google.PasteDataRequest{
			Coordinate: google.GridCoordinate{SheetID: s.id},
			Data:       util.CSVString(s.rows),
			Delimiter:  ",",
			Type:       pasteType,
		},
	}})
	return err
}

// prepareRefiller loads remote refill data and replaces rows with merged rows.
func (s *uploader) prepareRefiller() (*refiller, error) {
	refill, err := newRefiller(s)
	if err != nil {
		return nil, err
	}
	s.rows, err = refill.mergedRows()
	if err != nil {
		return nil, err
	}
	return refill, nil
}

// applyFilter adds a standard filter over uploaded data.
func (s *uploader) applyFilter() error {
	_, err := s.client.BatchUpdate(s.ctx, s.file.ID, []google.Request{{
		SetBasicFilter: &google.SetBasicFilterRequest{
			Filter: google.BasicFilter{
				Range: google.GridRange{
					SheetID:          s.id,
					EndRowIndex:      len(s.rows),
					EndColumnIndex:   len(s.rows[0]),
					StartRowIndex:    0,
					StartColumnIndex: 0,
				},
			},
		},
	}})
	return err
}

// applyNumeric formats obvious numeric CSV columns.
func (s *uploader) applyNumeric() error {
	formats := s.numericFormats()
	requests := make([]google.Request, 0, len(formats))
	if len(formats) == 0 {
		return nil
	}
	for c, pattern := range formats {
		requests = append(requests, google.Request{
			RepeatCell: &google.RepeatCellRequest{
				Range: google.GridRange{
					SheetID:          s.id,
					StartRowIndex:    1,
					EndRowIndex:      len(s.rows),
					StartColumnIndex: c,
					EndColumnIndex:   c + 1,
				},
				Cell: google.CellData{
					UserEnteredFormat: &google.CellFormat{
						NumberFormat: &google.NumberFormat{Type: "NUMBER", Pattern: pattern},
					},
				},
				Fields: "userEnteredFormat.numberFormat",
			},
		})
	}
	_, err := s.client.BatchUpdate(s.ctx, s.file.ID, requests)
	if err != nil {
		return err
	}
	return nil
}

// applyLayout auto-sizes columns and adds padding.
func (s *uploader) applyLayout() error {
	_, err := s.client.BatchUpdate(s.ctx, s.file.ID, []google.Request{{
		AutoResizeDimensions: &google.AutoResizeDimensionsRequest{
			Dimensions: google.DimensionRange{
				SheetID:    s.id,
				Dimension:  "COLUMNS",
				StartIndex: 0,
				EndIndex:   len(s.rows[0]),
			},
		},
	}})
	if err != nil {
		return err
	}

	requests, err := s.layoutWidthRequests()
	if err != nil {
		return err
	}
	if len(requests) == 0 {
		return nil
	}
	_, err = s.client.BatchUpdate(s.ctx, s.file.ID, requests)
	return err
}

// layoutWidthRequests builds padding requests from autosized column widths.
func (s *uploader) layoutWidthRequests() ([]google.Request, error) {
	spreadsheet, err := s.client.GetSpreadsheetWithGridData(s.ctx, s.file.ID, s.title)
	if err != nil {
		return nil, err
	}

	data := spreadsheet.Data[s.id]
	requests := []google.Request{}
	for columnIndex := range s.rows[0] {
		meta := data.ColumnMetadata[columnIndex]
		pixelSize := meta.PixelSize
		if pixelSize == 0 {
			pixelSize = 100
		}
		requests = append(requests, google.Request{
			UpdateDimensionProperties: &google.UpdateDimensionPropertiesRequest{
				Range: google.DimensionRange{
					SheetID:    s.id,
					Dimension:  "COLUMNS",
					StartIndex: columnIndex,
					EndIndex:   columnIndex + 1,
				},
				Properties: google.DimensionProperties{PixelSize: pixelSize + layoutPadding},
				Fields:     "pixelSize",
			},
		})
	}
	return requests, nil
}

// numericFormats returns target column indexes and Sheets number patterns.
func (s *uploader) numericFormats() map[int]string {
	formats := map[int]string{}
	if len(s.rows) < 2 {
		return formats
	}

	for c := range s.rows[0] {
		values := []string{}
		for _, row := range s.rows[1:] {
			value := row[c]
			if value != "" {
				values = append(values, value)
			}
		}
		if len(values) == 0 || util.AnyContains(values, ",") {
			continue
		}
		if hasLeadingZeroNumber(values) {
			continue
		}
		if util.AllMatch(values, integerRE) {
			formats[c] = "#,##0"
			continue
		}
		if !util.AllMatch(values, decimalRE) || !util.AnyContains(values, ".") {
			continue
		}
		formats[c] = "#,##0." + strings.Repeat("0", util.DecimalPrecision(values))
	}
	return formats
}

// isEmpty reports whether the only existing sheet has no values.
func (s *uploader) isEmpty() (bool, error) {
	if len(s.spreadsheet.Sheets) != 1 {
		return false, nil
	}
	rows, err := s.client.GetRows(s.ctx, s.file.ID, s.spreadsheet.Sheets[0].Title)
	if err != nil {
		return false, err
	}
	return len(rows) == 0, nil
}

//
// helpers
//

func hasLeadingZeroNumber(values []string) bool {
	return slices.ContainsFunc(values, leadingZeroRE.MatchString)
}

// sheetTitle returns the requested or generated destination sheet name.
func sheetTitle(cmd *UpCmd, spreadsheet *google.Spreadsheet) string {
	if cmd.Sheet != "" {
		return cmd.Sheet
	}
	if cmd.Refill || cmd.Replace {
		return defaultUploadSheet
	}

	next := 1
	for _, sheet := range spreadsheet.Sheets {
		if suffix, ok := strings.CutPrefix(sheet.Title, sheetPrefix); ok {
			n, err := strconv.Atoi(suffix)
			if err == nil {
				next = max(next, n+1)
			}
		}
	}
	return fmt.Sprintf("%s%d", sheetPrefix, next)
}
