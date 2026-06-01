package commands

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

//
// This does most of the work
//

const (
	gridPadding   = 2         // empty rows/columns kept around uploaded data
	layoutPadding = 20        // extra pixels added after auto-sizing columns
	sheetPrefix   = "gsheet_" // prefix for generated upload sheet names
)

var (
	integerRE     = regexp.MustCompile(`\A-?\d+\z`)           // whole-number detector for numeric formatting
	decimalRE     = regexp.MustCompile(`\A-?\d+(?:\.\d+)?\z`) // decimal detector for numeric formatting
	leadingZeroRE = regexp.MustCompile(`\A-?0\d`)             // numeric-looking value that should remain text
)

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

//
// resolveTargetSheet and friends
//

func (s *uploader) resolveTargetSheet() (int64, error) {
	// already present?
	for _, sheet := range s.spreadsheet.Sheets {
		if strings.EqualFold(sheet.Title, s.title) {
			return sheet.ID, nil
		}
	}

	// is this an empty file?
	isEmpty, err := s.isEmpty()
	if err != nil {
		return 0, err
	}

	reuseEmptySheet := isEmpty && (s.cmd.Refill || s.cmd.Replace || s.cmd.Sheet == "")
	if reuseEmptySheet {
		return s.renameSheet(s.spreadsheet.Sheets[0].ID)
	}
	return s.addSheet()
}

func (s *uploader) addSheet() (int64, error) {
	nrows, ncols := len(s.rows), len(s.rows[0])
	res, err := s.client.BatchUpdate(s.ctx, s.file.ID, []google.Request{{
		AddSheet: &google.AddSheetRequest{
			Properties: google.SheetProperties{
				Title: s.title,
				Index: new(0),
				GridProperties: &google.GridProperties{
					RowCount:    nrows + gridPadding,
					ColumnCount: ncols + gridPadding,
				},
			},
		},
	}})
	if err != nil {
		return 0, err
	}
	return res.Replies[0].AddSheet.Properties.ID, nil
}

func (s *uploader) renameSheet(sheetID int64) (int64, error) {
	_, err := s.client.BatchUpdate(s.ctx, s.file.ID, []google.Request{{
		UpdateSheetProperties: &google.UpdateSheetPropertiesRequest{
			Properties: google.SheetProperties{
				SheetID: new(sheetID),
				Title:   s.title,
			},
			Fields: "title",
		},
	}})
	return sheetID, err
}

//
// sheet ops
//

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
	nrows, ncols := len(s.rows), len(s.rows[0])
	_, err := s.client.BatchUpdate(s.ctx, s.file.ID, []google.Request{{
		UpdateSheetProperties: &google.UpdateSheetPropertiesRequest{
			Properties: google.SheetProperties{
				SheetID: new(s.id),
				GridProperties: &google.GridProperties{
					RowCount:    nrows + gridPadding,
					ColumnCount: ncols + gridPadding,
				},
			},
			Fields: "gridProperties.rowCount,gridProperties.columnCount",
		},
	}})
	return err
}

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

//
// options
//

func (s *uploader) applyFilter() error {
	nrows, ncols := len(s.rows), len(s.rows[0])
	_, err := s.client.BatchUpdate(s.ctx, s.file.ID, []google.Request{{
		SetBasicFilter: &google.SetBasicFilterRequest{
			Filter: google.BasicFilter{
				Range: google.GridRange{
					SheetID:          s.id,
					EndRowIndex:      nrows,
					EndColumnIndex:   ncols,
					StartRowIndex:    0,
					StartColumnIndex: 0,
				},
			},
		},
	}})
	return err
}

func (s *uploader) applyNumeric() error {
	nrows := len(s.rows)
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
					EndRowIndex:      nrows,
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

func (s *uploader) applyLayout() error {
	ncols := len(s.rows[0])
	_, err := s.client.BatchUpdate(s.ctx, s.file.ID, []google.Request{{
		AutoResizeDimensions: &google.AutoResizeDimensionsRequest{
			Dimensions: google.DimensionRange{
				SheetID:    s.id,
				Dimension:  "COLUMNS",
				StartIndex: 0,
				EndIndex:   ncols,
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
	ncols := len(s.rows[0])
	spreadsheet, err := s.client.GetSpreadsheetWithGridData(s.ctx, s.file.ID, s.title)
	if err != nil {
		return nil, err
	}

	data := spreadsheet.Data[s.id]
	requests := []google.Request{}
	for c := range ncols {
		meta := data.ColumnMetadata[c]
		pixelSize := meta.PixelSize
		if pixelSize == 0 {
			pixelSize = 100
		}
		requests = append(requests, google.Request{
			UpdateDimensionProperties: &google.UpdateDimensionPropertiesRequest{
				Range: google.DimensionRange{
					SheetID:    s.id,
					Dimension:  "COLUMNS",
					StartIndex: c,
					EndIndex:   c + 1,
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
	nrows, ncols := len(s.rows), len(s.rows[0])
	formats := map[int]string{}
	if nrows < 2 {
		return formats
	}

	for c := range ncols {
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
	// protects against things like zip codes that start with zeroes, `00234`
	return slices.ContainsFunc(values, leadingZeroRE.MatchString)
}

// sheetTitle returns the requested or generated destination sheet name.
func sheetTitle(cmd *UpCmd, spreadsheet *google.Spreadsheet) string {
	if cmd.Sheet != "" {
		return cmd.Sheet
	}
	if cmd.Refill || cmd.Replace {
		return "gsheet"
	}

	nxt := 1
	for _, sheet := range spreadsheet.Sheets {
		if suffix, ok := strings.CutPrefix(sheet.Title, sheetPrefix); ok {
			n, err := strconv.Atoi(suffix)
			if err == nil {
				nxt = max(nxt, n+1)
			}
		}
	}
	return fmt.Sprintf("%s%d", sheetPrefix, nxt)
}
