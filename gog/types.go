package gog

//
// Shared spreadsheet values used by commands and the gog client.
//

// File is a Google Drive spreadsheet file.
type File struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	ModifiedByMeTime string `json:"modifiedByMeTime"`
}

// Spreadsheet contains sheet metadata plus optional grid data.
type Spreadsheet struct {
	Title  string
	Sheets []*Sheet
	Data   map[int64]*SheetData
}

// Sheet is one tab from a spreadsheet.
type Sheet struct {
	ID             int64
	Title          string
	GridProperties *GridProperties
}

// SheetData is grid data needed by refill and layout.
type SheetData struct {
	FilterEndRow   int
	Rows           []RowData
	ColumnMetadata []ColumnMetadata
}

// GridProperties is a sheet's size.
type GridProperties struct {
	RowCount    int
	ColumnCount int
}

// GridRange identifies a rectangular sheet range.
type GridRange struct {
	SheetID          int64
	StartRowIndex    int
	EndRowIndex      int
	StartColumnIndex int
	EndColumnIndex   int
}

// Rows is rectangular spreadsheet data.
type Rows [][]string

//
// Operations
//

// Operation is one spreadsheet mutation translated into a gog command.
type Operation struct {
	AddSheet          *AddSheetOperation
	AutoResizeColumns *ColumnRange
	ClearCells        *ClearCellsOperation
	CopyCells         *CopyCellsOperation
	FormatCells       *FormatCellsOperation
	PasteRows         *PasteRowsOperation
	ResizeColumns     *ResizeColumnsOperation
	SetFilter         *GridRange
	UpdateSheet       *UpdateSheetOperation
}

type AddSheetOperation struct {
	Title          string
	Index          *int
	GridProperties *GridProperties
}

type UpdateSheetOperation struct {
	SheetID        int64
	Title          *string
	GridProperties *GridProperties
}

type ClearCellsOperation struct {
	Range GridRange
	All   bool
}

type PasteRowsOperation struct {
	SheetID int64
	Rows    Rows
}

type FormatCellsOperation struct {
	Range        GridRange
	NumberFormat *NumberFormat
}

type NumberFormat struct {
	Type    string
	Pattern string
}

type ColumnRange struct {
	SheetID    int64
	StartIndex int
	EndIndex   int
}

type ResizeColumnsOperation struct {
	Range     ColumnRange
	PixelSize int
}

type CopyCellsOperation struct {
	Source      GridRange
	Destination GridRange
	Type        string
}

// OperationResult contains values returned by a mutation.
type OperationResult struct {
	AddedSheet *Sheet
}
