package commands

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gurgeous/gshoot/gog"
)

//
// Plan a plaintext-column transformation into HYPERLINK formulas.
//

type hyperlinkCell struct {
	row     int
	formula string
}

type hyperlinker struct {
	backupColumn int
	backupHeader string
	cells        []hyperlinkCell
	textColumn   int
}

func newHyperlinker(rows gog.Rows, textHeader, linkHeader string) (*hyperlinker, error) {
	if err := rows.ValidateHeaders("sheet"); err != nil {
		return nil, err
	}
	indexes := rows.ColumnIndexes()
	textColumn, ok := indexes[textHeader]
	if !ok {
		return nil, fmt.Errorf("sheet has no %q column", textHeader)
	}
	linkColumn, originalLinkColumn := -1, -1
	if linkHeader == "" {
		if textHeader != "asin" {
			return nil, fmt.Errorf("cannot infer links for column %q; provide a link column", textHeader)
		}
	} else {
		linkColumn, ok = indexes[linkHeader]
		if !ok {
			return nil, fmt.Errorf("sheet has no %q column", linkHeader)
		}
		if textColumn == linkColumn {
			return nil, errors.New("plaintext and link columns must differ")
		}
		originalLinkColumn = linkColumn
	}

	backupHeader := textHeader + "2"
	if _, ok := indexes[backupHeader]; ok {
		return nil, fmt.Errorf("column %q already exists", backupHeader)
	}
	backupColumn := textColumn + 1
	if originalLinkColumn > textColumn {
		linkColumn++
	}

	h := &hyperlinker{
		backupColumn: backupColumn,
		backupHeader: backupHeader,
		textColumn:   textColumn,
	}
	for row := 1; row < len(rows); row++ {
		text := rows[row][textColumn]
		if text == "" {
			continue
		}
		text = strings.ReplaceAll(text, `"`, `""`)
		var link string
		if originalLinkColumn >= 0 {
			if rows[row][originalLinkColumn] == "" {
				continue
			}
			link = fmt.Sprintf("%s%d", gog.ColumnName(linkColumn), row+1)
		} else {
			link = fmt.Sprintf(`"https://www.amazon.com/dp/%s"`, text)
		}
		h.cells = append(h.cells, hyperlinkCell{
			row: row,
			formula: fmt.Sprintf(
				`=HYPERLINK(%s,"%s")`, link, text,
			),
		})
	}
	return h, nil
}

func (h *hyperlinker) prepare(sheetID int64) []gog.Operation {
	if len(h.cells) == 0 {
		return nil
	}
	return []gog.Operation{
		{InsertDimension: &gog.InsertDimensionOperation{
			SheetID: sheetID, Dimension: columnsDimension, Start: h.textColumn + 1, Count: 1, After: true,
		}},
		{CopyCells: &gog.CopyCellsOperation{
			Source: gog.GridRange{
				SheetID: sheetID, StartColumnIndex: h.textColumn, EndColumnIndex: h.textColumn + 1,
			},
			Destination: gog.GridRange{
				SheetID: sheetID, StartColumnIndex: h.backupColumn, EndColumnIndex: h.backupColumn + 1,
			},
			Type: "NORMAL",
		}},
	}
}

func (h *hyperlinker) values(sheetID int64) []gog.Operation {
	if len(h.cells) == 0 {
		return nil
	}
	operations := []gog.Operation{{PasteRows: &gog.PasteRowsOperation{
		SheetID: sheetID, ColumnIndex: h.backupColumn, Rows: gog.Rows{{h.backupHeader}},
	}}}
	for i := 0; i < len(h.cells); {
		start := h.cells[i].row
		rows := gog.Rows{}
		for i < len(h.cells) && h.cells[i].row == start+len(rows) {
			rows = append(rows, []string{h.cells[i].formula})
			i++
		}
		operations = append(operations, gog.Operation{PasteRows: &gog.PasteRowsOperation{
			SheetID: sheetID, RowIndex: start, ColumnIndex: h.textColumn, Rows: rows,
		}})
	}
	return operations
}
