package gog

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

//
// Batch-compatible sheet operations through gog's Sheets API commands.
//

type spreadsheetBatch struct {
	Requests []spreadsheetRequest `json:"requests"`
}

type spreadsheetRequest struct {
	AutoResizeDimensions *autoResizeDimensionsRequest `json:"autoResizeDimensions,omitempty"`
	CopyPaste            *copyPasteRequest            `json:"copyPaste,omitempty"`
	InsertDimension      *insertDimensionRequest      `json:"insertDimension,omitempty"`
	RepeatCell           *repeatCellRequest           `json:"repeatCell,omitempty"`
	SetBasicFilter       *setBasicFilterRequest       `json:"setBasicFilter,omitempty"`
}

type dimensionRange struct {
	SheetID   int64  `json:"sheetId"`
	Dimension string `json:"dimension"`
	Start     int    `json:"startIndex"`
	End       int    `json:"endIndex"`
}

type apiGridRange struct {
	SheetID          int64 `json:"sheetId"`
	StartRowIndex    int   `json:"startRowIndex,omitempty"`
	EndRowIndex      int   `json:"endRowIndex,omitempty"`
	StartColumnIndex int   `json:"startColumnIndex,omitempty"`
	EndColumnIndex   int   `json:"endColumnIndex,omitempty"`
}

type insertDimensionRequest struct {
	Range             dimensionRange `json:"range"`
	InheritFromBefore bool           `json:"inheritFromBefore"`
}

type copyPasteRequest struct {
	Source      apiGridRange `json:"source"`
	Destination apiGridRange `json:"destination"`
	PasteType   string       `json:"pasteType"`
}

type repeatCellRequest struct {
	Range  apiGridRange `json:"range"`
	Cell   cellData     `json:"cell"`
	Fields string       `json:"fields"`
}

type cellData struct {
	UserEnteredFormat map[string]any `json:"userEnteredFormat"`
}

type setBasicFilterRequest struct {
	Filter basicFilter `json:"filter"`
}

type basicFilter struct {
	Range apiGridRange `json:"range"`
}

type autoResizeDimensionsRequest struct {
	Dimensions dimensionRange `json:"dimensions"`
}

type valueRange struct {
	Range  string `json:"range"`
	Values Rows   `json:"values"`
}

// ApplySheetBatch applies one sheet phase with a single Sheets request.
func (c *Client) ApplySheetBatch(ctx context.Context, id string, sheet *Sheet, operations []Operation) error {
	if len(operations) == 0 {
		return nil
	}
	if operations[0].PasteRows != nil {
		return c.applyValueBatch(ctx, id, sheet, operations)
	}
	return c.applySpreadsheetBatch(ctx, id, operations)
}

func (c *Client) applyValueBatch(ctx context.Context, id string, sheet *Sheet, operations []Operation) error {
	ranges := make([]valueRange, 0, len(operations))
	for _, operation := range operations {
		paste := operation.PasteRows
		if paste == nil {
			return errors.New("value batch contains a non-value operation")
		}
		cell := fmt.Sprintf("%s!%s%d", quoteSheet(sheet.Title), ColumnName(paste.ColumnIndex), paste.RowIndex+1)
		ranges = append(ranges, valueRange{Range: cell, Values: paste.Rows})
	}

	body, err := json.Marshal(ranges)
	if err != nil {
		return err
	}
	debugBatch("values", len(ranges), len(body))
	return c.runJSON(ctx, bytes.NewReader(body), nil, sheetsCommand, "batch-update", id,
		"--data-json", "@-", "--input", "USER_ENTERED")
}

func (c *Client) applySpreadsheetBatch(ctx context.Context, id string, operations []Operation) error {
	requests := make([]spreadsheetRequest, 0, len(operations))
	for _, operation := range operations {
		request, err := operation.spreadsheetRequest()
		if err != nil {
			return err
		}
		requests = append(requests, request)
	}

	body, err := json.Marshal(spreadsheetBatch{Requests: requests})
	if err != nil {
		return err
	}
	debugBatch("spreadsheet", len(requests), len(body))
	return c.spreadsheetBatchUpdate(ctx, id, body)
}

func (o Operation) spreadsheetRequest() (spreadsheetRequest, error) {
	switch {
	case o.InsertDimension != nil:
		insert := o.InsertDimension
		dimension := "ROWS"
		if insert.Dimension == columnsDimension {
			dimension = sheetsColumns
		} else if insert.Dimension != "rows" {
			return spreadsheetRequest{}, fmt.Errorf("unsupported dimension %q", insert.Dimension)
		}
		start := insert.Start - 1
		if insert.After {
			start = insert.Start
		}
		inherit := insert.After
		if insert.InheritFromBefore != nil {
			inherit = *insert.InheritFromBefore
		}
		return spreadsheetRequest{InsertDimension: &insertDimensionRequest{
			Range: dimensionRange{
				SheetID: insert.SheetID, Dimension: dimension,
				Start: start, End: start + insert.Count,
			},
			InheritFromBefore: inherit,
		}}, nil

	case o.FormatCells != nil:
		format := o.FormatCells
		if format.NumberFormat != nil {
			return spreadsheetRequest{}, errors.New("number formats are not supported in sheet batches")
		}
		return spreadsheetRequest{RepeatCell: &repeatCellRequest{
			Range:  apiRange(format.Range),
			Cell:   cellData{UserEnteredFormat: map[string]any{}},
			Fields: "userEnteredFormat",
		}}, nil

	case o.CopyCells != nil:
		copyOp := o.CopyCells
		return spreadsheetRequest{CopyPaste: &copyPasteRequest{
			Source: apiRange(copyOp.Source), Destination: apiRange(copyOp.Destination),
			PasteType: "PASTE_" + copyOp.Type,
		}}, nil

	case o.SetFilter != nil:
		return spreadsheetRequest{SetBasicFilter: &setBasicFilterRequest{
			Filter: basicFilter{Range: apiRange(*o.SetFilter)},
		}}, nil

	case o.AutoResizeColumns != nil:
		resize := o.AutoResizeColumns
		return spreadsheetRequest{AutoResizeDimensions: &autoResizeDimensionsRequest{
			Dimensions: dimensionRange{
				SheetID: resize.SheetID, Dimension: sheetsColumns,
				Start: resize.StartIndex, End: resize.EndIndex,
			},
		}}, nil
	}
	return spreadsheetRequest{}, errors.New("unsupported sheet batch operation")
}

func apiRange(r GridRange) apiGridRange {
	return apiGridRange{
		SheetID: r.SheetID, StartRowIndex: r.StartRowIndex, EndRowIndex: r.EndRowIndex,
		StartColumnIndex: r.StartColumnIndex, EndColumnIndex: r.EndColumnIndex,
	}
}

func (c *Client) spreadsheetBatchUpdate(ctx context.Context, id string, body []byte) error {
	file, err := os.CreateTemp("", "gshoot-api-*.json")
	if err != nil {
		return fmt.Errorf("create gog API body: %w", err)
	}
	path := file.Name()
	defer func() { _ = os.Remove(path) }()

	if _, err := file.Write(body); err != nil {
		_ = file.Close()
		return fmt.Errorf("write gog API body: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close gog API body: %w", err)
	}

	params, err := json.Marshal(map[string]string{"spreadsheetId": id})
	if err != nil {
		return err
	}
	return c.runJSON(ctx, nil, nil, "api", "call", "sheets", "v4", "sheets.spreadsheets.batchUpdate",
		"--params", string(params), "--body", "@"+path,
		"--scope", "https://www.googleapis.com/auth/spreadsheets", "--allow-write", "--force")
}

func debugBatch(method string, count, size int) {
	debug := strings.TrimSpace(os.Getenv("GSHOOT_DEBUG"))
	if debug == "" || debug == "0" || strings.EqualFold(debug, "false") {
		return
	}
	_, _ = fmt.Fprintf(os.Stderr, "%s batch %s count=%d bytes=%d\n",
		time.Now().UTC().Format("2006-01-02T15:04:05.000Z"), method, count, size)
}
