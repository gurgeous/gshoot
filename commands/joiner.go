package commands

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/gurgeous/gshoot/gog"
	"github.com/gurgeous/gshoot/util"
)

//
// Plan a LEFT sheet / RIGHT CSV join without mutating either input.
//

const (
	columnsDimension = "cols"
	noneLabel        = "(none)"
)

type columnMatch struct {
	left   int
	right  int
	source string
	target string
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

func newJoiner(left, right gog.Rows, key string, columns []string) (*joiner, error) {
	if len(left) == 0 || len(right) == 0 {
		return nil, fmt.Errorf("join inputs must have headers")
	}
	if err := validateHeaders(left[0], "sheet"); err != nil {
		return nil, err
	}
	if err := validateHeaders(right[0], "csv"); err != nil {
		return nil, err
	}

	leftHeaders, rightHeaders := left[0], right[0]
	leftKey := util.IndexOfString(leftHeaders, key)
	if leftKey < 0 {
		return nil, fmt.Errorf("sheet has no %q column", key)
	}
	rightKey := util.IndexOfString(rightHeaders, key)
	if rightKey < 0 {
		return nil, fmt.Errorf("csv has no %q column", key)
	}
	if util.ContainsString(leftHeaders, "join") {
		return nil, fmt.Errorf("column %q already exists", "join")
	}

	selected, err := selectedColumns(rightHeaders, key, columns)
	if err != nil {
		return nil, err
	}
	leftKeys, err := keyedRows(left, leftKey, "sheet")
	if err != nil {
		return nil, err
	}
	rightKeys, err := keyedRows(right, rightKey, "csv")
	if err != nil {
		return nil, err
	}

	join := &joiner{
		key: key, left: left, leftHeight: len(left), leftKey: leftKey,
		leftKeys: leftKeys, leftWidth: len(leftHeaders), right: right,
		rightKey: rightKey, rightKeys: rightKeys,
	}
	leftHeaderSet := stringIndexes(leftHeaders)
	rightIndexes := stringIndexes(rightHeaders)
	selectedSet := map[string]bool{}
	for _, header := range selected {
		selectedSet[header] = true
	}
	for i, header := range leftHeaders {
		if header == key || header == "" {
			continue
		}
		if !selectedSet[header] {
			join.leftColumns = append(join.leftColumns, header)
			continue
		}
		join.matchColumns = append(join.matchColumns, columnMatch{
			left: i, right: rightIndexes[header], source: header, target: header + "2",
		})
	}
	for _, header := range rightHeaders {
		if !selectedSet[header] {
			continue
		}
		if _, ok := leftHeaderSet[header]; !ok {
			join.rightColumns = append(join.rightColumns, header)
		}
	}
	if err := join.validateOutputColumns(leftHeaders); err != nil {
		return nil, err
	}

	join.rows = join.joinRows()
	for _, row := range join.rows[1:] {
		switch row[0] {
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

func (j *joiner) preview(w io.Writer) {
	left := strings.Join(j.leftColumns, ", ")
	right := strings.Join(j.rightColumns, ", ")
	matches := make([]string, 0, len(j.matchColumns))
	for _, column := range j.matchColumns {
		matches = append(matches, column.source+" -> "+column.target)
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
	fmt.Fprintf(w, "Columns\n  key:   %s\n  left:  %s\n  right: %s\n  match: %s\n", j.key, left, right, match)
	fmt.Fprintf(w, "Rows\n  left:  %d\n  right: %d\n  match: %d\n", j.rowCounts.left, j.rowCounts.right, j.rowCounts.match)
}

func (j *joiner) operations(sheetID int64, hasFilter bool) []gog.Operation {
	operations := []gog.Operation{}
	matches := append([]columnMatch(nil), j.matchColumns...)
	sort.Slice(matches, func(i, k int) bool { return matches[i].left > matches[k].left })
	for _, column := range matches {
		operations = append(operations, gog.Operation{InsertDimension: &gog.InsertDimensionOperation{
			SheetID: sheetID, Dimension: columnsDimension, Start: column.left + 2, Count: 1,
		}})
	}
	if len(j.rightColumns) > 0 {
		operations = append(operations, gog.Operation{InsertDimension: &gog.InsertDimensionOperation{
			SheetID: sheetID, Dimension: columnsDimension, Start: j.leftWidth + len(j.matchColumns), Count: len(j.rightColumns), After: true,
		}})
	}
	operations = append(operations, gog.Operation{InsertDimension: &gog.InsertDimensionOperation{
		SheetID: sheetID, Dimension: columnsDimension, Start: 1, Count: 1,
	}})
	if j.rowCounts.right > 0 {
		inherit := true
		operations = append(operations, gog.Operation{InsertDimension: &gog.InsertDimensionOperation{
			SheetID: sheetID, Dimension: "rows", Start: j.leftHeight, Count: j.rowCounts.right,
			After: true, InheritFromBefore: &inherit,
		}})
	}

	inserted := j.insertedColumns()
	for _, column := range inserted {
		operations = append(operations, gog.Operation{FormatCells: &gog.FormatCellsOperation{
			Range: gog.GridRange{SheetID: sheetID, StartColumnIndex: column, EndColumnIndex: column + 1},
		}})
	}
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

func (j *joiner) insertedColumns() []int {
	headers := stringIndexes(j.rows[0])
	columns := make([]int, 1, 1+len(j.matchColumns)+len(j.rightColumns))
	for _, match := range j.matchColumns {
		columns = append(columns, headers[match.target])
	}
	for _, header := range j.rightColumns {
		columns = append(columns, headers[header])
	}
	sort.Ints(columns)
	return columns
}

func selectedColumns(headers []string, key string, columns []string) ([]string, error) {
	indexes := stringIndexes(headers)
	if columns == nil {
		selected := make([]string, 0, len(headers)-1)
		for _, header := range headers {
			if header != key && header != "" {
				selected = append(selected, header)
			}
		}
		return selected, nil
	}

	seen := map[string]bool{}
	for _, column := range columns {
		if column == "" {
			return nil, fmt.Errorf("--columns contains an empty column")
		}
		if column == key {
			return nil, fmt.Errorf("--columns must not include join key %q", key)
		}
		if seen[column] {
			return nil, fmt.Errorf("duplicate --columns value %q", column)
		}
		if _, ok := indexes[column]; !ok {
			return nil, fmt.Errorf("csv has no %q column", column)
		}
		seen[column] = true
	}

	selected := []string{}
	for _, header := range headers {
		if seen[header] {
			selected = append(selected, header)
		}
	}
	return selected, nil
}

func keyedRows(rows gog.Rows, keyColumn int, label string) (map[string]int, error) {
	keys := map[string]int{}
	for i := 1; i < len(rows); i++ {
		key := rows[i][keyColumn]
		if key == "" {
			continue
		}
		if _, ok := keys[key]; ok {
			return nil, fmt.Errorf("%s has duplicate key %q", label, key)
		}
		keys[key] = i
	}
	return keys, nil
}

func stringIndexes(values []string) map[string]int {
	indexes := make(map[string]int, len(values))
	for i, value := range values {
		indexes[value] = i
	}
	return indexes
}

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

func (j *joiner) joinRows() gog.Rows {
	headers := []string{"join"}
	matchByLeft := map[int]columnMatch{}
	for _, match := range j.matchColumns {
		matchByLeft[match.left] = match
	}
	leftOutput := make([]int, len(j.left[0]))
	matchOutput := map[int]int{}
	for i, header := range j.left[0] {
		leftOutput[i] = len(headers)
		headers = append(headers, header)
		if match, ok := matchByLeft[i]; ok {
			matchOutput[match.right] = len(headers)
			headers = append(headers, match.target)
		}
	}
	rightIndexes := stringIndexes(j.right[0])
	rightOutput := map[int]int{}
	for _, header := range j.rightColumns {
		rightIndex := rightIndexes[header]
		rightOutput[rightIndex] = len(headers)
		headers = append(headers, header)
	}

	rows := gog.Rows{headers}
	for i := 1; i < len(j.left); i++ {
		out := make([]string, len(headers))
		for c, value := range j.left[i] {
			out[leftOutput[c]] = value
		}
		rightRow, matched := j.rightKeys[j.left[i][j.leftKey]]
		out[0] = "left"
		if matched {
			out[0] = "match"
			copyRightValues(out, j.right[rightRow], matchOutput, rightOutput)
		}
		rows = append(rows, out)
	}
	for i := 1; i < len(j.right); i++ {
		key := j.right[i][j.rightKey]
		if _, matched := j.leftKeys[key]; matched {
			continue
		}
		out := make([]string, len(headers))
		out[0] = "right"
		out[leftOutput[j.leftKey]] = key
		copyRightValues(out, j.right[i], matchOutput, rightOutput)
		rows = append(rows, out)
	}
	return rows
}

func copyRightValues(out, right []string, matchOutput, rightOutput map[int]int) {
	for source, target := range matchOutput {
		out[target] = right[source]
	}
	for source, target := range rightOutput {
		out[target] = right[source]
	}
}
