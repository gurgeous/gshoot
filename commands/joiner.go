package commands

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/gurgeous/gshoot/gog"
)

//
// Plan a LEFT sheet / RIGHT CSV join without mutating either input.
//

const (
	columnsDimension = "cols"
	joinColumnIndex  = 1
	noneLabel        = "(none)"
)

type columnMatch struct {
	left   int
	right  int
	source string
	target string
}

type columnCopy struct {
	source int
	target int
}

type rowCounts struct {
	left  int
	right int
	match int
}

type joiner struct {
	key          string
	left         gog.Rows
	leftHeight   int
	leftKey      int
	leftKeys     map[string]int
	leftWidth    int
	leftColumns  []string
	right        gog.Rows
	rightKey     int
	rightKeys    map[string]int
	rightColumns []string
	matchColumns []columnMatch
	rowCounts    rowCounts
	rows         gog.Rows
}

// ctor
func newJoiner(left, right gog.Rows, key string, columns []string) (*joiner, error) {
	// validate
	if err := left.ValidateHeaders("sheet"); err != nil {
		return nil, err
	}
	if err := right.ValidateHeaders("csv"); err != nil {
		return nil, err
	}

	//
	// resolve key col
	//

	leftHeaders, rightHeaders := left[0], right[0]
	leftIndexes, rightIndexes := left.ColumnIndexes(), right.ColumnIndexes()
	leftKey, leftHasKey := leftIndexes[key]
	if !leftHasKey {
		return nil, fmt.Errorf("sheet has no %q column", key)
	}
	rightKey, rightHasKey := rightIndexes[key]
	if !rightHasKey {
		return nil, fmt.Errorf("csv has no %q column", key)
	}
	if _, ok := leftIndexes["join"]; ok {
		return nil, fmt.Errorf("column %q already exists", "join")
	}

	//
	// resolve selected and build row indices
	//

	selected, err := selectedColumnIndexes(right, key, columns)
	if err != nil {
		return nil, err
	}
	leftKeys, err := left.UniqueRowIndexes(leftKey, "google sheet")
	if err != nil {
		return nil, err
	}
	rightKeys, err := right.UniqueRowIndexes(rightKey, "csv")
	if err != nil {
		return nil, err
	}

	join := &joiner{
		key: key, left: left, leftHeight: len(left), leftKey: leftKey,
		leftKeys: leftKeys, leftWidth: len(leftHeaders), right: right,
		rightKey: rightKey, rightKeys: rightKeys,
	}

	//
	// classify columns into left/match/right
	//

	for i, header := range leftHeaders {
		if header == key || header == "" {
			continue
		}
		rightIndex, matched := selected[header]
		if !matched {
			join.leftColumns = append(join.leftColumns, header)
		} else {
			join.matchColumns = append(join.matchColumns, columnMatch{
				left: i, right: rightIndex, source: header, target: header + "2",
			})
		}
	}
	for _, header := range rightHeaders {
		_, isSelected := selected[header]
		_, matched := leftIndexes[header]
		if isSelected && !matched {
			join.rightColumns = append(join.rightColumns, header)
		}
	}
	if err := join.validateOutputColumns(leftHeaders); err != nil {
		return nil, err
	}

	//
	// build rows and tally for reports
	//

	join.rows = join.joinRows()
	for _, row := range join.rows[1:] {
		switch row[joinColumnIndex] {
		case "left":
			join.rowCounts.left++
		case "right":
			join.rowCounts.right++
		case "match":
			join.rowCounts.match++
		}
	}
	if join.rowCounts.match == 0 {
		return nil, fmt.Errorf("no rows match join key %q", key)
	}
	return join, nil
}

// preview prints column provenance and row counts before mutation.
func (j *joiner) preview(w io.Writer) {
	left := strings.Join(j.leftColumns, ", ")
	right := strings.Join(j.rightColumns, ", ")
	matches := make([]string, 0, len(j.matchColumns))
	for _, column := range j.matchColumns {
		matches = append(matches, column.source)
	}
	if left == "" {
		left = noneLabel
	}
	if right == "" {
		right = noneLabel
	}
	match := strings.Join(matches, ", ")
	if match == "" {
		match = noneLabel
	}
	fmt.Fprintf(w, "Column plan\n  key:   %s\n  left:  %s\n  match: %s\n  right: %s\n", j.key, left, match, right)
	fmt.Fprintf(w, "Row counts\n  left:  %d\n  match: %d\n  right: %d\n", j.rowCounts.left, j.rowCounts.match, j.rowCounts.right)
	fmt.Fprintln(w, "\nWorkaround: \"join\" is inserted after the first column until gog can insert before column A.")
}

// operations translates the plan into ordered, non-overwriting mutations.
func (j *joiner) operations(sheetID int64, hasFilter bool) []gog.Operation {
	// insert columns
	operations := []gog.Operation{}
	matches := append([]columnMatch(nil), j.matchColumns...)
	// Insert right-to-left so earlier indexes remain valid.
	sort.Slice(matches, func(i, k int) bool { return matches[i].left > matches[k].left })
	for _, column := range matches {
		operations = append(operations, gog.Operation{InsertDimension: &gog.InsertDimensionOperation{
			SheetID: sheetID, Dimension: columnsDimension, Start: column.left + 1, Count: 1, After: true,
		}})
	}
	if len(j.rightColumns) > 0 {
		operations = append(operations, gog.Operation{InsertDimension: &gog.InsertDimensionOperation{
			SheetID: sheetID, Dimension: columnsDimension, Start: j.leftWidth + len(j.matchColumns), Count: len(j.rightColumns), After: true,
		}})
	}
	// WORKAROUND: gog v0.37 cannot insert before column A. It converts column 1
	// to startIndex 0, but the Google client omits that zero from the request and
	// Sheets rejects the missing field. Keep "join" after the first column until
	// gog force-sends startIndex 0; then restore it as the first column.
	operations = append(operations, gog.Operation{InsertDimension: &gog.InsertDimensionOperation{
		SheetID: sheetID, Dimension: columnsDimension, Start: 1, Count: 1, After: true,
	}})

	// insert right-only rows
	if j.rowCounts.right > 0 {
		inherit := true
		operations = append(operations, gog.Operation{InsertDimension: &gog.InsertDimensionOperation{
			SheetID: sheetID, Dimension: "rows", Start: j.leftHeight, Count: j.rowCounts.right,
			After: true, InheritFromBefore: &inherit,
		}})
	}

	// clear and populate inserted cells
	inserted := j.insertedColumns()
	// Remove formatting inherited by inserted columns before populating them.
	for _, column := range inserted {
		operations = append(operations, gog.Operation{FormatCells: &gog.FormatCellsOperation{
			Range: gog.GridRange{SheetID: sheetID, StartColumnIndex: column, EndColumnIndex: column + 1},
		}})
	}
	// Existing rows receive values only in newly inserted columns.
	for _, column := range inserted {
		values := make(gog.Rows, j.leftHeight)
		for row := range values {
			values[row] = []string{j.rows[row][column]}
		}
		operations = append(operations, gog.Operation{PasteRows: &gog.PasteRowsOperation{
			SheetID: sheetID, ColumnIndex: column, Rows: values,
		}})
	}
	if j.rowCounts.right > 0 {
		operations = append(operations, gog.Operation{PasteRows: &gog.PasteRowsOperation{
			SheetID: sheetID, RowIndex: j.leftHeight, Rows: j.rows[j.leftHeight:],
		}})
	}

	// filter and layout
	// Reapplying expands the range but discards filter criteria and sorting.
	if hasFilter {
		operations = append(operations, gog.Operation{SetFilter: &gog.GridRange{
			SheetID: sheetID, EndRowIndex: len(j.rows), EndColumnIndex: len(j.rows[0]),
		}})
	}
	for _, column := range inserted {
		operations = append(operations, gog.Operation{AutoResizeColumns: &gog.ColumnRange{
			SheetID: sheetID, StartIndex: column, EndIndex: column + 1,
		}})
	}
	return operations
}

// insertedColumns returns the final indexes of every newly inserted column.
func (j *joiner) insertedColumns() []int {
	headers := j.rows.ColumnIndexes()
	columns := make([]int, 1, 1+len(j.matchColumns)+len(j.rightColumns))
	columns[0] = headers["join"]
	for _, match := range j.matchColumns {
		columns = append(columns, headers[match.target])
	}
	for _, header := range j.rightColumns {
		columns = append(columns, headers[header])
	}
	sort.Ints(columns)
	return columns
}

// selectedColumnIndexes validates selected columns and returns RIGHT indexes.
func selectedColumnIndexes(rows gog.Rows, key string, columns []string) (map[string]int, error) {
	headers, indexes := rows[0], rows.ColumnIndexes()
	selected := map[string]int{}
	if columns == nil {
		// default to every named non-key column
		for i, header := range headers {
			if header != key && header != "" {
				selected[header] = i
			}
		}
		return selected, nil
	}

	// validate explicit columns
	for _, column := range columns {
		if column == "" {
			return nil, fmt.Errorf("--columns contains an empty column")
		}
		if column == key {
			return nil, fmt.Errorf("--columns must not include join key %q", key)
		}
		if _, ok := selected[column]; ok {
			return nil, fmt.Errorf("duplicate --columns value %q", column)
		}
		index, ok := indexes[column]
		if !ok {
			return nil, fmt.Errorf("csv has no %q column", column)
		}
		selected[column] = index
	}
	return selected, nil
}

// validateOutputColumns rejects names that would overwrite planned output.
func (j *joiner) validateOutputColumns(leftHeaders []string) error {
	output := map[string]bool{"join": true}
	for _, header := range leftHeaders {
		output[header] = true
	}
	for _, header := range j.rightColumns {
		if output[header] {
			return fmt.Errorf("column %q already exists", header)
		}
		output[header] = true
	}
	for _, match := range j.matchColumns {
		if output[match.target] {
			return fmt.Errorf("column %q already exists", match.target)
		}
		output[match.target] = true
	}
	return nil
}

// joinRows builds output values while preserving LEFT and RIGHT row order.
func (j *joiner) joinRows() gog.Rows {
	// build headers and source-to-output column mappings
	headers := []string{}
	matchByLeft := map[int]columnMatch{}
	for _, match := range j.matchColumns {
		matchByLeft[match.left] = match
	}
	leftOutput := make([]int, len(j.left[0]))
	rightCopies := make([]columnCopy, 0, len(j.matchColumns)+len(j.rightColumns))
	for i, header := range j.left[0] {
		leftOutput[i] = len(headers)
		headers = append(headers, header)
		if i == 0 {
			headers = append(headers, "join")
		}
		if match, ok := matchByLeft[i]; ok {
			rightCopies = append(rightCopies, columnCopy{source: match.right, target: len(headers)})
			headers = append(headers, match.target)
		}
	}
	rightIndexes := j.right.ColumnIndexes()
	for _, header := range j.rightColumns {
		rightIndex := rightIndexes[header]
		rightCopies = append(rightCopies, columnCopy{source: rightIndex, target: len(headers)})
		headers = append(headers, header)
	}

	// emit left rows, mixing in matching right values
	rows := gog.Rows{headers}
	for i := 1; i < len(j.left); i++ {
		out := make([]string, len(headers))
		for c, value := range j.left[i] {
			out[leftOutput[c]] = value
		}
		rightRow, matched := j.rightKeys[j.left[i][j.leftKey]]
		out[joinColumnIndex] = "left"
		if matched {
			out[joinColumnIndex] = "match"
			copyColumns(out, j.right[rightRow], rightCopies)
		}
		rows = append(rows, out)
	}

	// append right-only rows
	for i := 1; i < len(j.right); i++ {
		key := j.right[i][j.rightKey]
		if _, matched := j.leftKeys[key]; matched {
			continue
		}
		out := make([]string, len(headers))
		out[joinColumnIndex] = "right"
		out[leftOutput[j.leftKey]] = key
		copyColumns(out, j.right[i], rightCopies)
		rows = append(rows, out)
	}
	return rows
}

// copyColumns places selected source values into their output columns.
func copyColumns(out, source []string, columns []columnCopy) {
	for _, column := range columns {
		out[column.target] = source[column.source]
	}
}
