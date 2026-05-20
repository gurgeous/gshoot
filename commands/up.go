package commands

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/gurgeous/gshoot/google"
	"github.com/gurgeous/gshoot/util"
	"github.com/gurgeous/gshoot/ux"
)

const (
	defaultUploadSheet = "gsheet_up"
	sheetPrefix        = "gsheet_up_"
	layoutPadding      = 20
)

var (
	integerRE = regexp.MustCompile(`\A-?\d+\z`)
	decimalRE = regexp.MustCompile(`\A-?\d+(?:\.\d+)?\z`)
)

// UpCmd uploads a CSV to Google Sheets.
type UpCmd struct {
	Filter      bool   `help:"Add a standard Google Sheets filter."`
	Layout      bool   `help:"Auto-size columns to fit cells."`
	Numeric     bool   `help:"Format obvious numeric columns."`
	Open        bool   `help:"Open the sheet URL when done."`
	Refill      bool   `help:"Merge CSV data into the destination sheet."`
	Replace     bool   `help:"Create or overwrite the destination sheet."`
	Sheet       string `help:"Destination sheet name."`
	Spreadsheet string `arg:"" name:"spreadsheet" help:"Spreadsheet name."`
	CSVPath     string `arg:"" name:"csv" type:"path" help:"CSV file to upload."`
}

// Run uploads the configured CSV.
func (c *UpCmd) Run() error {
	if c.Refill && c.Replace {
		return errors.New("use either --refill or --replace")
	}

	rows, err := readUploadCSV(c.CSVPath)
	if err != nil {
		return err
	}

	ctx := context.Background()
	dots := ux.StartDots(os.Stderr, "connecting to Google Sheets...")

	client, err := google.NewClient(ctx, google.ReadWriteScopes())
	if err != nil {
		return err
	}

	runner := &upRunner{
		cmd:    c,
		ctx:    ctx,
		client: client,
		rows:   rows,
	}
	if err := runner.upload(dots); err != nil {
		return err
	}

	dots.Stop()
	url := util.SpreadsheetURL(runner.file.ID) + "/edit"
	fmt.Fprintln(os.Stdout, url)
	if c.Open {
		return util.OpenBrowserURL(url)
	}
	return nil
}

// REVIEW: add a suffix comment for each ivar
type upRunner struct {
	cmd               *UpCmd
	ctx               context.Context
	client            *google.Client
	file              *google.File
	spreadsheet       *google.Spreadsheet
	rows              google.Rows
	uploadRows        google.Rows
	existingRows      google.Rows
	existingUserRows  google.Rows
	existingSheetData *google.SheetData
	targetTitle       string
	targetSheetID     int64
}

// upload runs the complete upload workflow.
func (u *upRunner) upload(dots *ux.Dots) error {
	if err := u.findOrCreateFile(dots); err != nil {
		return err
	}
	if err := u.loadSpreadsheet(); err != nil {
		return err
	}
	u.uploadRows = u.rows
	u.targetTitle = u.targetSheetTitle()

	dots.SetDescription(fmt.Sprintf("uploading %d rows to '%s' / '%s'...", len(u.rows), u.file.Name, u.targetTitle))
	if err := u.ensureSheet(); err != nil {
		return err
	}
	if u.cmd.Refill {
		if err := u.loadExistingSheetData(); err != nil {
			return err
		}
	}

	if u.cmd.Refill {
		merged, err := u.refillRows()
		if err != nil {
			return err
		}
		u.uploadRows = merged
	}

	if u.cmd.Replace {
		if err := u.clearSheet(); err != nil {
			return err
		}
	}
	if err := u.resizeSheet(); err != nil {
		return err
	}
	if err := u.pasteSheet(); err != nil {
		return err
	}
	if err := u.applyOptions(); err != nil {
		return err
	}
	return nil
}

// REVIEW: move this to google client
// findOrCreateFile finds the target spreadsheet or creates it.
func (u *upRunner) findOrCreateFile(dots *ux.Dots) error {
	dots.SetDescription("finding spreadsheet...")
	file, err := u.client.FindSpreadsheet(u.ctx, u.cmd.Spreadsheet)
	if err != nil {
		return err
	}
	if file != nil {
		u.file = file
		return nil
	}

	dots.SetDescription(fmt.Sprintf("creating '%s'...", u.cmd.Spreadsheet))
	u.file, err = u.client.CreateSpreadsheet(u.ctx, u.cmd.Spreadsheet)
	return err
}

// REVIEW: why dos this exist?
// loadSpreadsheet fetches sheet metadata for the selected file.
func (u *upRunner) loadSpreadsheet() error {
	spreadsheet, err := u.client.GetSpreadsheet(u.ctx, u.file.ID)
	if err != nil {
		return err
	}
	u.spreadsheet = spreadsheet
	return nil
}

// ensureSheet selects, creates, or renames the target sheet.
func (u *upRunner) ensureSheet() error {
	if existing := u.existingSheetID(); existing != nil {
		u.targetSheetID = *existing
		return nil
	}
	blankDefault, err := u.blankDefaultSheet()
	if err != nil {
		return err
	}
	if u.cmd.Refill || u.cmd.Replace {
		return u.renameOrAddSheet(blankDefault)
	}
	if u.cmd.Sheet != "" || !blankDefault {
		return u.addSheet()
	}
	return u.renameSheet(u.spreadsheet.Sheets[0].ID)
}

// renameOrAddSheet reuses a blank default sheet or creates a new one.
func (u *upRunner) renameOrAddSheet(blankDefault bool) error {
	if blankDefault {
		return u.renameSheet(u.spreadsheet.Sheets[0].ID)
	}
	return u.addSheet()
}

// addSheet creates the target sheet and records its ID.
func (u *upRunner) addSheet() error {
	res, err := u.client.BatchUpdate(u.ctx, u.file.ID, []google.Request{{
		AddSheet: &google.AddSheetRequest{
			Properties: google.SheetProperties{
				Title: u.targetTitle,
				Index: new(0),
				GridProperties: &google.GridProperties{
					RowCount:    u.targetRowCount(),
					ColumnCount: u.targetColumnCount(),
				},
			},
		},
	}})
	if err != nil {
		return err
	}
	if len(res.Replies) == 0 || res.Replies[0].AddSheet == nil {
		return errors.New("Google Sheets did not return the created sheet ID")
	}
	u.targetSheetID = res.Replies[0].AddSheet.Properties.ID
	return nil
}

// renameSheet renames an existing sheet and records it as the target.
func (u *upRunner) renameSheet(sheetID int64) error {
	_, err := u.client.BatchUpdate(u.ctx, u.file.ID, []google.Request{{
		UpdateSheetProperties: &google.UpdateSheetPropertiesRequest{
			Properties: google.SheetProperties{
				SheetID: new(sheetID),
				Title:   u.targetTitle,
			},
			Fields: "title",
		},
	}})
	u.targetSheetID = sheetID
	return err
}

// clearSheet clears every cell in the target sheet.
func (u *upRunner) clearSheet() error {
	_, err := u.client.BatchUpdate(u.ctx, u.file.ID, []google.Request{{
		UpdateCells: &google.UpdateCellsRequest{
			Range:  google.GridRange{SheetID: u.targetSheetID},
			Fields: "*",
		},
	}})
	return err
}

// resizeSheet expands the target grid to fit data plus padding.
func (u *upRunner) resizeSheet() error {
	_, err := u.client.BatchUpdate(u.ctx, u.file.ID, []google.Request{{
		UpdateSheetProperties: &google.UpdateSheetPropertiesRequest{
			Properties: google.SheetProperties{
				SheetID: new(u.targetSheetID),
				GridProperties: &google.GridProperties{
					RowCount:    u.targetRowCount(),
					ColumnCount: u.targetColumnCount(),
				},
			},
			Fields: "gridProperties.rowCount,gridProperties.columnCount",
		},
	}})
	return err
}

// pasteSheet pastes CSV data into the target sheet.
func (u *upRunner) pasteSheet() error {
	pasteType := "PASTE_NORMAL"
	if u.cmd.Refill {
		pasteType = "PASTE_VALUES"
	}
	_, err := u.client.BatchUpdate(u.ctx, u.file.ID, []google.Request{{
		PasteData: &google.PasteDataRequest{
			Coordinate: google.GridCoordinate{SheetID: u.targetSheetID},
			Data:       util.CSVString(u.uploadRows),
			Delimiter:  ",",
			Type:       pasteType,
		},
	}})
	return err
}

// applyOptions applies refill, filter, numeric, and layout options.
func (u *upRunner) applyOptions() error {
	if u.cmd.Refill {
		if err := u.applyRefillFormats(); err != nil {
			return err
		}
		if err := u.applyRefillFormulas(); err != nil {
			return err
		}
		if err := u.clearPaddingFormats(); err != nil {
			return err
		}
	}
	if u.cmd.Filter {
		if err := u.applyFilter(); err != nil {
			return err
		}
	}
	if u.cmd.Numeric {
		if err := u.applyNumeric(); err != nil {
			return err
		}
	}
	if u.cmd.Layout {
		return u.applyLayout()
	}
	return nil
}

// applyFilter adds a standard filter over uploaded data.
func (u *upRunner) applyFilter() error {
	_, err := u.client.BatchUpdate(u.ctx, u.file.ID, []google.Request{{
		SetBasicFilter: &google.SetBasicFilterRequest{
			Filter: google.BasicFilter{
				Range: google.GridRange{
					SheetID:          u.targetSheetID,
					EndRowIndex:      u.rowCount(),
					EndColumnIndex:   u.columnCount(),
					StartRowIndex:    0,
					StartColumnIndex: 0,
				},
			},
		},
	}})
	return err
}

// applyNumeric formats obvious numeric CSV columns.
func (u *upRunner) applyNumeric() error {
	formats := u.numericFormats()
	requests := make([]google.Request, 0, len(formats))
	for columnIndex, pattern := range formats {
		requests = append(requests, google.Request{
			RepeatCell: &google.RepeatCellRequest{
				Range: google.GridRange{
					SheetID:          u.targetSheetID,
					StartRowIndex:    1,
					EndRowIndex:      u.rowCount(),
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
	_, err := u.client.BatchUpdate(u.ctx, u.file.ID, requests)
	return err
}

// applyLayout auto-sizes columns and adds padding.
func (u *upRunner) applyLayout() error {
	_, err := u.client.BatchUpdate(u.ctx, u.file.ID, []google.Request{{
		AutoResizeDimensions: &google.AutoResizeDimensionsRequest{
			Dimensions: google.DimensionRange{
				SheetID:    u.targetSheetID,
				Dimension:  "COLUMNS",
				StartIndex: 0,
				EndIndex:   u.columnCount(),
			},
		},
	}})
	if err != nil {
		return err
	}

	requests, err := u.layoutWidthRequests()
	if err != nil {
		return err
	}
	if len(requests) == 0 {
		return nil
	}
	_, err = u.client.BatchUpdate(u.ctx, u.file.ID, requests)
	return err
}

// applyRefillFormats copies existing data-row formats across refilled columns.
func (u *upRunner) applyRefillFormats() error {
	if len(u.existingRows) < 2 || u.rowCount() <= 1 {
		return nil
	}
	requests := []google.Request{}
	for _, columnIndex := range u.existingCSVColumns() {
		requests = append(requests, google.Request{
			CopyPaste: &google.CopyPasteRequest{
				Source: google.GridRange{
					SheetID:          u.targetSheetID,
					StartRowIndex:    1,
					EndRowIndex:      2,
					StartColumnIndex: columnIndex,
					EndColumnIndex:   columnIndex + 1,
				},
				Destination: google.GridRange{
					SheetID:          u.targetSheetID,
					StartRowIndex:    1,
					EndRowIndex:      u.rowCount(),
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
	_, err := u.client.BatchUpdate(u.ctx, u.file.ID, requests)
	return err
}

// applyRefillFormulas extends non-CSV formula columns during refill.
func (u *upRunner) applyRefillFormulas() error {
	if u.rowCount() <= u.existingDataRowCount() || u.existingDataRowCount() < 2 {
		return nil
	}
	requests := []google.Request{}
	for _, columnIndex := range u.refillFormulaColumns() {
		requests = append(requests, google.Request{
			CopyPaste: &google.CopyPasteRequest{
				Source: google.GridRange{
					SheetID:          u.targetSheetID,
					StartRowIndex:    1,
					EndRowIndex:      u.existingDataRowCount(),
					StartColumnIndex: columnIndex,
					EndColumnIndex:   columnIndex + 1,
				},
				Destination: google.GridRange{
					SheetID:          u.targetSheetID,
					StartRowIndex:    1,
					EndRowIndex:      u.rowCount(),
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
	_, err := u.client.BatchUpdate(u.ctx, u.file.ID, requests)
	return err
}

// clearPaddingFormats clears formatting outside the refilled data area.
func (u *upRunner) clearPaddingFormats() error {
	_, err := u.client.BatchUpdate(u.ctx, u.file.ID, []google.Request{
		{
			RepeatCell: &google.RepeatCellRequest{
				Range: google.GridRange{
					SheetID:          u.targetSheetID,
					StartRowIndex:    u.rowCount(),
					EndRowIndex:      u.targetRowCount(),
					StartColumnIndex: 0,
					EndColumnIndex:   u.targetColumnCount(),
				},
				Cell:   google.CellData{UserEnteredFormat: &google.CellFormat{}},
				Fields: "userEnteredFormat",
			},
		},
		{
			RepeatCell: &google.RepeatCellRequest{
				Range: google.GridRange{
					SheetID:          u.targetSheetID,
					StartRowIndex:    0,
					EndRowIndex:      u.rowCount(),
					StartColumnIndex: u.columnCount(),
					EndColumnIndex:   u.targetColumnCount(),
				},
				Cell:   google.CellData{UserEnteredFormat: &google.CellFormat{}},
				Fields: "userEnteredFormat",
			},
		},
	})
	return err
}

// layoutWidthRequests builds padding requests from autosized column widths.
func (u *upRunner) layoutWidthRequests() ([]google.Request, error) {
	refreshed, err := u.client.GetSpreadsheetWithGridData(u.ctx, u.file.ID, u.targetTitle)
	if err != nil {
		return nil, err
	}

	data := refreshed.Data[u.targetSheetID]
	if data == nil {
		return nil, nil
	}
	requests := []google.Request{}
	for columnIndex, meta := range data.ColumnMetadata[:min(len(data.ColumnMetadata), u.columnCount())] {
		pixelSize := meta.PixelSize
		if pixelSize == 0 {
			pixelSize = 100
		}
		requests = append(requests, google.Request{
			UpdateDimensionProperties: &google.UpdateDimensionPropertiesRequest{
				Range: google.DimensionRange{
					SheetID:    u.targetSheetID,
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

// loadExistingSheetData loads display values and user-entered grid data for refill.
func (u *upRunner) loadExistingSheetData() error {
	rows, err := u.client.GetRows(u.ctx, u.file.ID, u.targetTitle)
	if err != nil {
		return err
	}
	u.existingRows = rows

	refreshed, err := u.client.GetSpreadsheetWithGridData(u.ctx, u.file.ID, u.targetTitle)
	if err != nil {
		return err
	}
	u.existingSheetData = refreshed.Data[u.targetSheetID]
	u.existingUserRows = userEnteredRows(u.existingSheetData)
	return nil
}

// refillRows merges CSV rows into the existing sheet shape.
func (u *upRunner) refillRows() (google.Rows, error) {
	if len(u.existingRows) == 0 {
		return u.rows, nil
	}

	existingHeaders := u.existingRows[0]
	csvHeaders := u.rows[0]
	if err := validateHeaders(existingHeaders, "existing sheet"); err != nil {
		return nil, err
	}
	if err := validateHeaders(csvHeaders, "csv"); err != nil {
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

	finalRowCount := max(len(u.existingRows), len(u.rows))
	merged := make(google.Rows, finalRowCount)
	for rowIndex := range merged {
		merged[rowIndex] = make([]string, len(finalHeaders))
	}
	for rowIndex, row := range u.existingUserRows {
		for columnIndex, value := range row {
			merged[rowIndex][columnIndex] = value
		}
	}
	for csvColumnIndex, header := range csvHeaders {
		targetColumnIndex := byHeader[header]
		for rowIndex, row := range u.rows {
			merged[rowIndex][targetColumnIndex] = row[csvColumnIndex]
		}
	}
	return merged, nil
}

// numericFormats returns target column indexes and Sheets number patterns.
func (u *upRunner) numericFormats() map[int]string {
	formats := map[int]string{}
	if len(u.rows) < 2 {
		return formats
	}

	targetColumns := u.csvTargetColumns()
	for columnIndex := range u.rows[0] {
		values := []string{}
		for _, row := range u.rows[1:] {
			value := row[columnIndex]
			if value != "" {
				values = append(values, value)
			}
		}
		if len(values) == 0 || util.AnyContains(values, ",") {
			continue
		}
		if util.AllMatch(values, integerRE) {
			formats[targetColumns[columnIndex]] = "#,##0"
			continue
		}
		if !util.AllMatch(values, decimalRE) || !util.AnyContains(values, ".") {
			continue
		}
		formats[targetColumns[columnIndex]] = "#,##0." + strings.Repeat("0", util.DecimalPrecision(values))
	}
	return formats
}

// refillFormulaColumns returns non-CSV columns that contain formulas.
func (u *upRunner) refillFormulaColumns() []int {
	existingCSV := map[int]bool{}
	for _, columnIndex := range u.existingCSVColumns() {
		existingCSV[columnIndex] = true
	}
	columns := []int{}
	for columnIndex := range u.existingRows[0] {
		if !existingCSV[columnIndex] && u.formulaColumn(columnIndex) {
			columns = append(columns, columnIndex)
		}
	}
	return columns
}

// formulaColumn reports whether a non-CSV column should be formula-extended.
func (u *upRunner) formulaColumn(columnIndex int) bool {
	sawFormula := false
	for rowIndex := 1; rowIndex < u.existingDataRowCount(); rowIndex++ {
		value := u.existingRows[rowIndex][columnIndex]
		if value == "" {
			continue
		}
		if rowIndex >= len(u.existingSheetData.Rows) || columnIndex >= len(u.existingSheetData.Rows[rowIndex].Values) {
			return false
		}
		cell := u.existingSheetData.Rows[rowIndex].Values[columnIndex]
		if cell.UserEnteredValue == nil || cell.UserEnteredValue.FormulaValue == nil || *cell.UserEnteredValue.FormulaValue == "" {
			return false
		}
		sawFormula = true
	}
	return sawFormula
}

// existingDataRowCount returns the existing rows covered by the filter or data.
func (u *upRunner) existingDataRowCount() int {
	count := len(u.existingRows)
	if u.existingSheetData != nil && u.existingSheetData.BasicFilter != nil && u.existingSheetData.BasicFilter.Range.EndRowIndex > 0 {
		count = u.existingSheetData.BasicFilter.Range.EndRowIndex
	}
	return min(count, len(u.existingRows))
}

// existingCSVColumns returns existing sheet columns that also appear in the CSV.
func (u *upRunner) existingCSVColumns() []int {
	csvHeaders := map[string]bool{}
	for _, header := range u.rows[0] {
		csvHeaders[header] = true
	}
	columns := []int{}
	for columnIndex, header := range u.existingRows[0] {
		if csvHeaders[header] {
			columns = append(columns, columnIndex)
		}
	}
	return columns
}

// csvTargetColumns maps CSV columns to upload target columns.
func (u *upRunner) csvTargetColumns() []int {
	headers := u.uploadRows
	if len(headers) == 0 {
		headers = u.rows
	}
	targetHeaders := headers[0]
	targetColumns := make([]int, 0, len(u.rows[0]))
	for _, header := range u.rows[0] {
		targetColumns = append(targetColumns, util.IndexOfString(targetHeaders, header))
	}
	return targetColumns
}

// targetSheetTitle returns the requested or generated destination sheet name.
func (u *upRunner) targetSheetTitle() string {
	if u.cmd.Sheet != "" {
		return u.cmd.Sheet
	}
	if u.cmd.Refill || u.cmd.Replace {
		return defaultUploadSheet
	}
	return u.nextSheetTitle()
}

// nextSheetTitle returns the next gsheet_up_N sheet title.
func (u *upRunner) nextSheetTitle() string {
	next := 1
	for _, sheet := range u.spreadsheet.Sheets {
		if suffix, ok := strings.CutPrefix(sheet.Title, sheetPrefix); ok {
			n, err := strconv.Atoi(suffix)
			if err == nil {
				next = max(next, n+1)
			}
		}
	}
	return fmt.Sprintf("%s%d", sheetPrefix, next)
}

// existingSheetID returns the ID of the current target sheet, if present.
func (u *upRunner) existingSheetID() *int64 {
	for _, sheet := range u.spreadsheet.Sheets {
		if strings.EqualFold(sheet.Title, u.targetTitle) {
			return &sheet.ID
		}
	}
	return nil
}

// blankDefaultSheet reports whether the only existing sheet has no values.
func (u *upRunner) blankDefaultSheet() (bool, error) {
	if len(u.spreadsheet.Sheets) != 1 {
		return false, nil
	}
	rows, err := u.client.GetRows(u.ctx, u.file.ID, u.spreadsheet.Sheets[0].Title)
	if err != nil {
		return false, err
	}
	return len(rows) == 0, nil
}

// rowCount returns the number of rows to upload.
func (u *upRunner) rowCount() int {
	return len(u.uploadRows)
}

// columnCount returns the number of columns to upload.
func (u *upRunner) columnCount() int {
	return len(u.uploadRows[0])
}

// targetRowCount returns upload rows plus legacy padding.
func (u *upRunner) targetRowCount() int {
	return u.rowCount() + 2
}

// targetColumnCount returns upload columns plus legacy padding.
func (u *upRunner) targetColumnCount() int {
	return u.columnCount() + 2
}

// readUploadCSV reads and rectangularizes the input CSV.
func readUploadCSV(path string) (google.Rows, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("not found: %s", path)
		}
		return nil, fmt.Errorf("open csv: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	rows, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("csv is empty: %s", path)
	}
	return google.Rectangularize(rows), nil
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
			values = append(values, userEnteredCellValue(cell))
		}
		rows = append(rows, values)
	}
	return google.Rectangularize(rows)
}

// userEnteredCellValue stringifies the user-entered value for one cell.
func userEnteredCellValue(cell google.CellData) string {
	value := cell.UserEnteredValue
	if value == nil {
		return ""
	}
	switch {
	case value.FormulaValue != nil:
		return *value.FormulaValue
	case value.StringValue != nil:
		return *value.StringValue
	case value.NumberValue != nil:
		return fmt.Sprint(*value.NumberValue)
	case value.BoolValue != nil:
		return fmt.Sprint(*value.BoolValue)
	case value.ErrorValue != nil:
		return value.ErrorValue.Type
	default:
		return ""
	}
}

// validateHeaders rejects duplicate non-empty headers.
func validateHeaders(headers []string, label string) error {
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
