package commands

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/gurgeous/gshoot/gog"
	"github.com/gurgeous/gshoot/util"
)

//
// This does most of the work
//

const (
	gridPadding    = 2   // empty rows/columns kept around uploaded data
	layoutPadding  = 20  // extra pixels added after auto-sizing columns
	layoutMaxWidth = 300 // maximum column width after auto-sizing
)

var (
	integerRE     = regexp.MustCompile(`\A-?\d+\z`)           // whole-number detector for numeric formatting
	decimalRE     = regexp.MustCompile(`\A-?\d+(?:\.\d+)?\z`) // decimal detector for numeric formatting
	leadingZeroRE = regexp.MustCompile(`\A-?0\d`)             // numeric-looking value that should remain text
)

type uploader struct {
	ctx         context.Context  // request context for Google calls
	client      *gog.Client      // Google API client
	file        *gog.File        // spreadsheet Drive file
	spreadsheet *gog.Spreadsheet // current spreadsheet metadata
	cmd         *UpCmd           // upload command options
	title       string           // target sheet title
	id          int64            // target sheet ID after ensure
	rows        gog.Rows         // rows to paste into the target sheet
}

func newUploader(
	ctx context.Context,
	client *gog.Client,
	file *gog.File,
	spreadsheet *gog.Spreadsheet,
	cmd *UpCmd,
	rows gog.Rows,
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
	if sheet := findSheet(s.spreadsheet, s.title); sheet != nil {
		if s.cmd.Refill || s.cmd.Replace {
			return sheet.ID, nil
		}
		s.title = nextAvailableSheetTitle(s.spreadsheet, s.title)
		return s.addSheet()
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
	results, err := s.client.Apply(s.ctx, s.file.ID, []gog.Operation{{
		AddSheet: &gog.AddSheetOperation{
			Title: s.title,
			Index: new(0),
			GridProperties: &gog.GridProperties{
				RowCount:    nrows + gridPadding,
				ColumnCount: ncols + gridPadding,
			},
		},
	}})
	if err != nil {
		return 0, err
	}
	return results[0].AddedSheet.ID, nil
}

func (s *uploader) renameSheet(sheetID int64) (int64, error) {
	_, err := s.client.Apply(s.ctx, s.file.ID, []gog.Operation{{
		UpdateSheet: &gog.UpdateSheetOperation{
			SheetID: sheetID,
			Title:   &s.title,
		},
	}})
	return sheetID, err
}

//
// sheet ops
//

func (s *uploader) clearSheet() error {
	_, err := s.client.Apply(s.ctx, s.file.ID, []gog.Operation{{
		ClearCells: &gog.ClearCellsOperation{
			Range: gog.GridRange{SheetID: s.id},
			All:   true,
		},
	}})
	return err
}

// growSheet expands the target grid to fit data plus padding.
func (s *uploader) growSheet() error {
	nrows, ncols := len(s.rows), len(s.rows[0])
	_, err := s.client.Apply(s.ctx, s.file.ID, []gog.Operation{{
		UpdateSheet: &gog.UpdateSheetOperation{
			SheetID: s.id,
			GridProperties: &gog.GridProperties{
				RowCount:    nrows + gridPadding,
				ColumnCount: ncols + gridPadding,
			},
		},
	}})
	return err
}

func (s *uploader) pasteCSV() error {
	_, err := s.client.Apply(s.ctx, s.file.ID, []gog.Operation{{
		PasteRows: &gog.PasteRowsOperation{
			SheetID: s.id,
			Rows:    s.rows,
		},
	}})
	return err
}

func (s *uploader) prepareRefiller() (*refiller, error) {
	refill, err := newRefiller(s)
	if err != nil {
		return nil, err
	}
	s.rows = refill.pasteRows()
	return refill, nil
}

//
// options
//

func (s *uploader) applyFilter() error {
	nrows, ncols := len(s.rows), len(s.rows[0])
	_, err := s.client.Apply(s.ctx, s.file.ID, []gog.Operation{{
		SetFilter: &gog.GridRange{
			SheetID:        s.id,
			EndRowIndex:    nrows,
			EndColumnIndex: ncols,
		},
	}})
	return err
}

func (s *uploader) applyNumeric() error {
	nrows := len(s.rows)
	formats := s.numericFormats()
	operations := make([]gog.Operation, 0, len(formats))
	if len(formats) == 0 {
		return nil
	}
	for c, pattern := range formats {
		operations = append(operations, gog.Operation{
			FormatCells: &gog.FormatCellsOperation{
				Range: gog.GridRange{
					SheetID:          s.id,
					StartRowIndex:    1,
					EndRowIndex:      nrows,
					StartColumnIndex: c,
					EndColumnIndex:   c + 1,
				},
				NumberFormat: &gog.NumberFormat{Type: "NUMBER", Pattern: pattern},
			},
		})
	}
	_, err := s.client.Apply(s.ctx, s.file.ID, operations)
	if err != nil {
		return err
	}
	return nil
}

func (s *uploader) applyLayout() error {
	ncols := len(s.rows[0])
	_, err := s.client.Apply(s.ctx, s.file.ID, []gog.Operation{{
		AutoResizeColumns: &gog.ColumnRange{
			SheetID:    s.id,
			StartIndex: 0,
			EndIndex:   ncols,
		},
	}})
	if err != nil {
		return err
	}

	operations, err := s.layoutWidthOperations()
	if err != nil {
		return err
	}
	if len(operations) == 0 {
		return nil
	}
	_, err = s.client.Apply(s.ctx, s.file.ID, operations)
	return err
}

// layoutWidthOperations builds padding operations from autosized column widths.
func (s *uploader) layoutWidthOperations() ([]gog.Operation, error) {
	ncols := len(s.rows[0])
	spreadsheet, err := s.client.GetSpreadsheetWithGridData(s.ctx, s.file.ID)
	if err != nil {
		return nil, err
	}

	data := spreadsheet.Data[s.id]
	if data == nil {
		return nil, fmt.Errorf("sheet %q has no grid data", s.title)
	}
	if len(data.ColumnMetadata) < ncols {
		return nil, fmt.Errorf("sheet %q has column metadata for %d of %d columns", s.title, len(data.ColumnMetadata), ncols)
	}
	operations := []gog.Operation{}
	for c := range ncols {
		meta := data.ColumnMetadata[c]
		pixelSize := meta.PixelSize
		if pixelSize == 0 {
			pixelSize = 100
		}
		operations = append(operations, gog.Operation{
			ResizeColumns: &gog.ResizeColumnsOperation{
				Range: gog.ColumnRange{
					SheetID:    s.id,
					StartIndex: c,
					EndIndex:   c + 1,
				},
				PixelSize: util.Clamp(pixelSize+layoutPadding, 0, layoutMaxWidth),
			},
		})
	}
	return operations, nil
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
func sheetTitle(cmd *UpCmd, spreadsheet *gog.Spreadsheet) string {
	title := cmd.Sheet
	if title == "" {
		title = csvSheetTitle(cmd.CSVPath)
	}
	if cmd.Refill || cmd.Replace {
		return title
	}
	return nextAvailableSheetTitle(spreadsheet, title)
}

func csvSheetTitle(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

func nextAvailableSheetTitle(spreadsheet *gog.Spreadsheet, title string) string {
	if findSheet(spreadsheet, title) == nil {
		return title
	}

	for ii := 2; ; ii++ {
		nxt := fmt.Sprintf("%s_%d", title, ii)
		if findSheet(spreadsheet, nxt) == nil {
			return nxt
		}
	}
}

func findSheet(spreadsheet *gog.Spreadsheet, title string) *gog.Sheet {
	for _, sheet := range spreadsheet.Sheets {
		if strings.EqualFold(sheet.Title, title) {
			return sheet
		}
	}
	return nil
}
