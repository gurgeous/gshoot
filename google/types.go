package google

import "fmt"

//
// Drive payloads
//

// These are the tiny subset of Google Drive fields needed to find or create
// spreadsheet files. Keep these payloads narrow so the custom client stays small.

// File is a Google Drive file.
type File struct {
	ID               string `json:"id,omitempty"`
	Name             string `json:"name"`
	ModifiedByMeTime string `json:"modifiedByMeTime,omitempty"`
	MimeType         string `json:"mimeType,omitempty"`
}

//
// Spreadsheet payloads
//

// Spreadsheet and Sheet are the domain values the rest of the app consumes.
// spreadsheetResponse below handles the awkward API response shape.

// Sheet is one tab from a Google spreadsheet.
type Sheet struct {
	ID    int64  `json:"sheetId"`
	Title string `json:"title"`
}

// Spreadsheet is sheet metadata plus optional grid data.
type Spreadsheet struct {
	ID     string
	Sheets []*Sheet
	Data   map[int64]*SheetData
}

type spreadsheetResponse struct {
	SpreadsheetID string          `json:"spreadsheetId"`
	Sheets        []sheetResponse `json:"sheets"`
}

type sheetResponse struct {
	Properties  *SheetProperties    `json:"properties"`
	BasicFilter *BasicFilter        `json:"basicFilter"`
	Data        []sheetDataResponse `json:"data"`
}

type sheetDataResponse struct {
	RowData        []RowData             `json:"rowData"`
	ColumnMetadata []DimensionProperties `json:"columnMetadata"`
}

func (r spreadsheetResponse) spreadsheet() *Spreadsheet {
	spreadsheet := &Spreadsheet{
		ID:   r.SpreadsheetID,
		Data: map[int64]*SheetData{},
	}
	for _, item := range r.Sheets {
		if item.Properties != nil {
			sheet := item.Properties.sheet()
			spreadsheet.Sheets = append(spreadsheet.Sheets, sheet)
			spreadsheet.Data[sheet.ID] = item.sheetData()
		}
	}
	return spreadsheet
}

func (p SheetProperties) sheet() *Sheet {
	var id int64
	if p.SheetID != nil {
		id = *p.SheetID
	}
	return &Sheet{ID: id, Title: p.Title}
}

func (r sheetResponse) sheetData() *SheetData {
	data := &SheetData{BasicFilter: r.BasicFilter}
	if len(r.Data) > 0 {
		data.Rows = r.Data[0].RowData
		data.ColumnMetadata = r.Data[0].ColumnMetadata
	}
	return data
}

//
// Grid data payloads
//

// These represent user-entered cell values and formatting. They are only fetched
// for upload/refill flows that need formulas, filters, or column metadata.

// SheetData is the subset of grid data needed for upload/refill.
type SheetData struct {
	BasicFilter    *BasicFilter
	Rows           []RowData
	ColumnMetadata []DimensionProperties
}

// RowData contains user-entered cells for one row.
type RowData struct {
	Values []CellData `json:"values,omitempty"`
}

// CellData is a minimal Sheets cell payload.
type CellData struct {
	UserEnteredValue  *ExtendedValue `json:"userEnteredValue,omitempty"`
	UserEnteredFormat *CellFormat    `json:"userEnteredFormat,omitempty"`
}

// UserEnteredString returns the user-entered value as a string.
func (c CellData) UserEnteredString() string {
	value := c.UserEnteredValue
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

// ExtendedValue is a minimal Sheets extended value.
type ExtendedValue struct {
	StringValue  *string     `json:"stringValue,omitempty"`
	NumberValue  *float64    `json:"numberValue,omitempty"`
	BoolValue    *bool       `json:"boolValue,omitempty"`
	FormulaValue *string     `json:"formulaValue,omitempty"`
	ErrorValue   *ErrorValue `json:"errorValue,omitempty"`
}

// ErrorValue is a Sheets error cell value.
type ErrorValue struct {
	Type    string `json:"type,omitempty"`
	Message string `json:"message,omitempty"`
}

//
// Format payloads
//

// Formatting payloads are intentionally partial. We only model number formats
// and column widths because those are the only formatting mutations we send.

// CellFormat is a minimal Sheets cell format.
type CellFormat struct {
	NumberFormat *NumberFormat `json:"numberFormat,omitempty"`
}

// NumberFormat is a Sheets number format.
type NumberFormat struct {
	Type    string `json:"type,omitempty"`
	Pattern string `json:"pattern,omitempty"`
}

// DimensionProperties is a minimal Sheets dimension payload.
type DimensionProperties struct {
	PixelSize int `json:"pixelSize,omitempty"`
}

// GridProperties is a minimal Sheets grid payload.
type GridProperties struct {
	RowCount    int `json:"rowCount,omitempty"`
	ColumnCount int `json:"columnCount,omitempty"`
}

//
// Range payloads
//

// Sheets batchUpdate requests use grid, dimension, and coordinate ranges. These
// structs share names with the API docs so request construction stays readable.

// GridRange identifies a rectangular sheet range.
type GridRange struct {
	SheetID          int64 `json:"sheetId"`
	StartRowIndex    int   `json:"startRowIndex,omitempty"`
	EndRowIndex      int   `json:"endRowIndex,omitempty"`
	StartColumnIndex int   `json:"startColumnIndex,omitempty"`
	EndColumnIndex   int   `json:"endColumnIndex,omitempty"`
}

// DimensionRange identifies sheet rows or columns.
type DimensionRange struct {
	SheetID    int64  `json:"sheetId"`
	Dimension  string `json:"dimension"`
	StartIndex int    `json:"startIndex,omitempty"`
	EndIndex   int    `json:"endIndex,omitempty"`
}

// GridCoordinate identifies one sheet cell.
type GridCoordinate struct {
	SheetID     int64 `json:"sheetId"`
	RowIndex    int   `json:"rowIndex,omitempty"`
	ColumnIndex int   `json:"columnIndex,omitempty"`
}

// BasicFilter is a Sheets basic filter.
type BasicFilter struct {
	Range GridRange `json:"range"`
}

//
// Batch update request payloads
//

// Request is a union of the Sheets batchUpdate mutations supported by gshoot.
// Each field maps directly to one Google Sheets request type.

// Request is one Sheets batchUpdate request.
type Request struct {
	AddSheet                  *AddSheetRequest                  `json:"addSheet,omitempty"`
	UpdateSheetProperties     *UpdateSheetPropertiesRequest     `json:"updateSheetProperties,omitempty"`
	UpdateCells               *UpdateCellsRequest               `json:"updateCells,omitempty"`
	PasteData                 *PasteDataRequest                 `json:"pasteData,omitempty"`
	SetBasicFilter            *SetBasicFilterRequest            `json:"setBasicFilter,omitempty"`
	RepeatCell                *RepeatCellRequest                `json:"repeatCell,omitempty"`
	AutoResizeDimensions      *AutoResizeDimensionsRequest      `json:"autoResizeDimensions,omitempty"`
	UpdateDimensionProperties *UpdateDimensionPropertiesRequest `json:"updateDimensionProperties,omitempty"`
	CopyPaste                 *CopyPasteRequest                 `json:"copyPaste,omitempty"`
}

// AddSheetRequest adds a sheet.
type AddSheetRequest struct {
	Properties SheetProperties `json:"properties"`
}

// UpdateSheetPropertiesRequest updates sheet properties.
type UpdateSheetPropertiesRequest struct {
	Properties SheetProperties `json:"properties"`
	Fields     string          `json:"fields"`
}

// UpdateCellsRequest updates or clears cells.
type UpdateCellsRequest struct {
	Range  GridRange `json:"range"`
	Fields string    `json:"fields"`
}

// PasteDataRequest pastes delimited text into a sheet.
type PasteDataRequest struct {
	Coordinate GridCoordinate `json:"coordinate"`
	Data       string         `json:"data"`
	Delimiter  string         `json:"delimiter"`
	Type       string         `json:"type"`
}

// SetBasicFilterRequest applies a basic filter.
type SetBasicFilterRequest struct {
	Filter BasicFilter `json:"filter"`
}

// RepeatCellRequest applies cell data over a range.
type RepeatCellRequest struct {
	Range  GridRange `json:"range"`
	Cell   CellData  `json:"cell"`
	Fields string    `json:"fields"`
}

// AutoResizeDimensionsRequest asks Sheets to autosize dimensions.
type AutoResizeDimensionsRequest struct {
	Dimensions DimensionRange `json:"dimensions"`
}

// UpdateDimensionPropertiesRequest updates dimension properties.
type UpdateDimensionPropertiesRequest struct {
	Range      DimensionRange      `json:"range"`
	Properties DimensionProperties `json:"properties"`
	Fields     string              `json:"fields"`
}

// CopyPasteRequest copies cells from one range to another.
type CopyPasteRequest struct {
	Source           GridRange `json:"source"`
	Destination      GridRange `json:"destination"`
	PasteType        string    `json:"pasteType"`
	PasteOrientation string    `json:"pasteOrientation"`
}

// SheetProperties is a minimal Sheets sheet properties payload.
type SheetProperties struct {
	SheetID        *int64          `json:"sheetId,omitempty"`
	Title          string          `json:"title,omitempty"`
	Index          *int            `json:"index,omitempty"`
	GridProperties *GridProperties `json:"gridProperties,omitempty"`
}

//
// Batch update response payloads
//

// Only addSheet replies are consumed today; other mutation replies are ignored.

// BatchUpdateResponse contains the replies from a Sheets batchUpdate call.
type BatchUpdateResponse struct {
	Replies []Reply `json:"replies"`
}

// Reply is one Sheets batchUpdate reply.
type Reply struct {
	AddSheet *AddSheetReply `json:"addSheet,omitempty"`
}

// AddSheetReply contains the created sheet properties.
type AddSheetReply struct {
	Properties Sheet `json:"properties"`
}

//
// data from a sheet
//

// Rows is rectangular spreadsheet data.
type Rows [][]string
