package gog

import "fmt"

//
// Decode gog metadata and raw Sheets responses into gshoot values.
//

type spreadsheetResponse struct {
	Title  string          `json:"title"`
	Sheets []sheetResponse `json:"sheets"`
}

type sheetResponse struct {
	Properties  *sheetPropertiesResponse `json:"properties"`
	BasicFilter *basicFilterResponse     `json:"basicFilter"`
	Data        []sheetDataResponse      `json:"data"`
}

type sheetPropertiesResponse struct {
	SheetID        int64                  `json:"sheetId"`
	Title          string                 `json:"title"`
	GridProperties gridPropertiesResponse `json:"gridProperties"`
}

type gridPropertiesResponse struct {
	RowCount    int `json:"rowCount"`
	ColumnCount int `json:"columnCount"`
}

type basicFilterResponse struct {
	Range struct {
		SheetID          int64 `json:"sheetId"`
		StartRowIndex    int   `json:"startRowIndex"`
		EndRowIndex      int   `json:"endRowIndex"`
		StartColumnIndex int   `json:"startColumnIndex"`
		EndColumnIndex   int   `json:"endColumnIndex"`
	} `json:"range"`
}

type sheetDataResponse struct {
	RowData        []RowData        `json:"rowData"`
	ColumnMetadata []ColumnMetadata `json:"columnMetadata"`
}

// RowData contains user-entered cells for one row.
type RowData struct {
	Values []CellData `json:"values"`
}

// CellData is the user-entered value of one cell.
type CellData struct {
	UserEnteredValue *ExtendedValue `json:"userEnteredValue"`
}

// ExtendedValue is a user-entered Sheets value.
type ExtendedValue struct {
	StringValue  *string  `json:"stringValue"`
	NumberValue  *float64 `json:"numberValue"`
	BoolValue    *bool    `json:"boolValue"`
	FormulaValue *string  `json:"formulaValue"`
}

// UserEnteredString returns the value as a string, preserving formulas.
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
	default:
		return ""
	}
}

// ColumnMetadata contains column metadata returned by gog.
type ColumnMetadata struct {
	PixelSize int `json:"pixelSize"`
}

func (r spreadsheetResponse) spreadsheet() *Spreadsheet {
	spreadsheet := &Spreadsheet{Title: r.Title, Data: map[int64]*SheetData{}}
	for _, item := range r.Sheets {
		if item.Properties == nil {
			continue
		}
		properties := item.Properties
		sheet := &Sheet{
			ID:    properties.SheetID,
			Title: properties.Title,
			GridProperties: &GridProperties{
				RowCount:    properties.GridProperties.RowCount,
				ColumnCount: properties.GridProperties.ColumnCount,
			},
		}
		data := &SheetData{}
		if item.BasicFilter != nil {
			rng := item.BasicFilter.Range
			data.FilterRange = &GridRange{
				SheetID: rng.SheetID, StartRowIndex: rng.StartRowIndex,
				EndRowIndex: rng.EndRowIndex, StartColumnIndex: rng.StartColumnIndex,
				EndColumnIndex: rng.EndColumnIndex,
			}
		}
		if len(item.Data) > 0 {
			data.Rows = item.Data[0].RowData
			data.ColumnMetadata = item.Data[0].ColumnMetadata
		}
		spreadsheet.Sheets = append(spreadsheet.Sheets, sheet)
		spreadsheet.Data[sheet.ID] = data
	}
	return spreadsheet
}
