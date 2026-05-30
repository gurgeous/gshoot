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

type uploadSheet struct {
	ctx         context.Context     // request context for Google calls
	client      *google.Client      // Google API client
	fileID      string              // spreadsheet file ID
	spreadsheet *google.Spreadsheet // current spreadsheet metadata
	title       string              // target sheet title
	id          int64               // target sheet ID after ensure
	rows        google.Rows         // rows to paste into the target sheet
	refill      bool                // paste values only and preserve formulas
	replace     bool                // clear or create the destination sheet
	sheetName   string              // explicit destination sheet name
	filter      bool                // apply a basic filter after upload
	numeric     bool                // format obvious numeric columns
	layout      bool                // auto-size columns with padding
}

// newUploadSheet creates a target sheet mutator.
func newUploadSheet(
	ctx context.Context,
	client *google.Client,
	fileID string,
	spreadsheet *google.Spreadsheet,
	cmd *UpCmd,
	rows google.Rows,
) *uploadSheet {
	return &uploadSheet{
		ctx:         ctx,
		client:      client,
		fileID:      fileID,
		spreadsheet: spreadsheet,
		title:       sheetTitle(cmd, spreadsheet),
		rows:        rows,
		refill:      cmd.Refill,
		replace:     cmd.Replace,
		sheetName:   cmd.Sheet,
		filter:      cmd.Filter,
		numeric:     cmd.Numeric,
		layout:      cmd.Layout,
	}
}

// ensure selects, creates, or renames the target sheet.
func (s *uploadSheet) ensure() error {
	if existingID, ok := s.findExistingID(); ok {
		s.id = existingID
		return nil
	}
	blankDefault, err := s.blankDefault()
	if err != nil {
		return err
	}
	if s.refill || s.replace {
		if blankDefault {
			return s.rename(s.spreadsheet.Sheets[0].ID)
		}
		return s.add()
	}
	if s.sheetName != "" || !blankDefault {
		return s.add()
	}
	return s.rename(s.spreadsheet.Sheets[0].ID)
}

// add creates the target sheet and records its ID.
func (s *uploadSheet) add() error {
	res, err := s.client.BatchUpdate(s.ctx, s.fileID, []google.Request{{
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

// rename renames an existing sheet and records it as the target.
func (s *uploadSheet) rename(sheetID int64) error {
	_, err := s.client.BatchUpdate(s.ctx, s.fileID, []google.Request{{
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

// clear clears every cell in the target sheet.
func (s *uploadSheet) clear() error {
	_, err := s.client.BatchUpdate(s.ctx, s.fileID, []google.Request{{
		UpdateCells: &google.UpdateCellsRequest{
			Range:  google.GridRange{SheetID: s.id},
			Fields: "*",
		},
	}})
	return err
}

// resize expands the target grid to fit data plus padding.
func (s *uploadSheet) resize() error {
	_, err := s.client.BatchUpdate(s.ctx, s.fileID, []google.Request{{
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

// paste pastes CSV data into the target sheet.
func (s *uploadSheet) paste() error {
	pasteType := "PASTE_NORMAL"
	if s.refill {
		pasteType = "PASTE_VALUES"
	}
	_, err := s.client.BatchUpdate(s.ctx, s.fileID, []google.Request{{
		PasteData: &google.PasteDataRequest{
			Coordinate: google.GridCoordinate{SheetID: s.id},
			Data:       util.CSVString(s.rows),
			Delimiter:  ",",
			Type:       pasteType,
		},
	}})
	return err
}

// applyOptions applies filter, numeric, and layout options.
func (s *uploadSheet) applyOptions() error {
	if s.filter {
		if err := s.applyFilter(); err != nil {
			return err
		}
	}
	if s.numeric {
		if err := s.applyNumeric(); err != nil {
			return err
		}
	}
	if s.layout {
		return s.applyLayout()
	}
	return nil
}

// applyFilter adds a standard filter over uploaded data.
func (s *uploadSheet) applyFilter() error {
	_, err := s.client.BatchUpdate(s.ctx, s.fileID, []google.Request{{
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
func (s *uploadSheet) applyNumeric() error {
	formats := s.numericFormats()
	requests := make([]google.Request, 0, len(formats))
	for columnIndex, pattern := range formats {
		requests = append(requests, google.Request{
			RepeatCell: &google.RepeatCellRequest{
				Range: google.GridRange{
					SheetID:          s.id,
					StartRowIndex:    1,
					EndRowIndex:      len(s.rows),
					StartColumnIndex: columnIndex,
					EndColumnIndex:   columnIndex + 1,
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
	if len(requests) == 0 {
		return nil
	}
	_, err := s.client.BatchUpdate(s.ctx, s.fileID, requests)
	return err
}

// applyLayout auto-sizes columns and adds padding.
func (s *uploadSheet) applyLayout() error {
	_, err := s.client.BatchUpdate(s.ctx, s.fileID, []google.Request{{
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
	_, err = s.client.BatchUpdate(s.ctx, s.fileID, requests)
	return err
}

// layoutWidthRequests builds padding requests from autosized column widths.
func (s *uploadSheet) layoutWidthRequests() ([]google.Request, error) {
	refreshed, err := s.client.GetSpreadsheetWithGridData(s.ctx, s.fileID, s.title)
	if err != nil {
		return nil, err
	}

	data := refreshed.Data[s.id]
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
func (s *uploadSheet) numericFormats() map[int]string {
	formats := map[int]string{}
	if len(s.rows) < 2 {
		return formats
	}

	for columnIndex := range s.rows[0] {
		values := []string{}
		for _, row := range s.rows[1:] {
			value := row[columnIndex]
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
			formats[columnIndex] = "#,##0"
			continue
		}
		if !util.AllMatch(values, decimalRE) || !util.AnyContains(values, ".") {
			continue
		}
		formats[columnIndex] = "#,##0." + strings.Repeat("0", util.DecimalPrecision(values))
	}
	return formats
}

func hasLeadingZeroNumber(values []string) bool {
	return slices.ContainsFunc(values, leadingZeroRE.MatchString)
}

// findExistingID returns the ID of the current target sheet, if present.
func (s *uploadSheet) findExistingID() (int64, bool) {
	for _, sheet := range s.spreadsheet.Sheets {
		if strings.EqualFold(sheet.Title, s.title) {
			return sheet.ID, true
		}
	}
	return 0, false
}

// blankDefault reports whether the only existing sheet has no values.
func (s *uploadSheet) blankDefault() (bool, error) {
	if len(s.spreadsheet.Sheets) != 1 {
		return false, nil
	}
	rows, err := s.client.GetRows(s.ctx, s.fileID, s.spreadsheet.Sheets[0].Title)
	if err != nil {
		return false, err
	}
	return len(rows) == 0, nil
}

// sheetTitle returns the requested or generated destination sheet name.
func sheetTitle(cmd *UpCmd, spreadsheet *google.Spreadsheet) string {
	if cmd.Sheet != "" {
		return cmd.Sheet
	}
	if cmd.Refill || cmd.Replace {
		return defaultUploadSheet
	}
	return nextSheetTitle(spreadsheet)
}

// nextSheetTitle returns the next gsheet_N sheet title.
func nextSheetTitle(spreadsheet *google.Spreadsheet) string {
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
