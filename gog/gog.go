package gog

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/gurgeous/gshoot/util"
)

//
// Google Sheets operations implemented through the gog command-line client.
//

const (
	sheetsCommand       = "sheets"
	spreadsheetMimeType = "application/vnd.google-apps.spreadsheet"
)

type Client struct {
	gog string
}

// NewClient locates gog. gog owns authentication and account selection.
func NewClient(_ context.Context) (*Client, error) {
	path, err := exec.LookPath("gog")
	if err != nil {
		return nil, errors.New("gog is required; install it with `brew install openclaw/tap/gogcli`")
	}
	return &Client{gog: path}, nil
}

func (c *Client) CreateSpreadsheetFile(ctx context.Context, name string) (*File, error) {
	var out struct {
		ID   string `json:"spreadsheetId"`
		Name string `json:"title"`
	}
	if err := c.runJSON(ctx, nil, &out, sheetsCommand, "create", name); err != nil {
		return nil, err
	}
	return &File{ID: out.ID, Name: out.Name}, nil
}

func (c *Client) CopySpreadsheet(ctx context.Context, id, title string) (*File, error) {
	var out struct {
		File *File `json:"file"`
	}
	if err := c.runJSON(ctx, nil, &out, sheetsCommand, "copy", id, title, "--parent", "root"); err != nil {
		return nil, err
	}
	if out.File == nil {
		return nil, errors.New("gog returned no copied spreadsheet")
	}
	return out.File, nil
}

// FindSpreadsheetFile accepts a spreadsheet name, ID, or URL.
func (c *Client) FindSpreadsheetFile(ctx context.Context, ref string) (*File, error) {
	if strings.Contains(ref, "://") {
		id := spreadsheetID(ref)
		if id == "" {
			return nil, fmt.Errorf("invalid Google Sheets URL %q", ref)
		}
		spreadsheet, err := c.GetSpreadsheet(ctx, id)
		if err != nil {
			return nil, err
		}
		return &File{ID: id, Name: spreadsheet.Title}, nil
	}

	files, err := c.listFiles(ctx, "name = '"+driveQueryString(ref)+"'", 1000)
	if err != nil || len(files) == 0 {
		if err != nil {
			return nil, err
		}
		id := spreadsheetID(ref)
		if id == "" {
			return nil, nil
		}
		spreadsheet, err := c.GetSpreadsheet(ctx, id)
		if err != nil {
			return nil, err
		}
		return &File{ID: id, Name: spreadsheet.Title}, nil
	}
	return files[0], nil
}

func (c *Client) FindOrCreateSpreadsheetFile(ctx context.Context, ref string) (*File, error) {
	file, err := c.FindSpreadsheetFile(ctx, ref)
	if err != nil || file != nil {
		return file, err
	}
	return c.CreateSpreadsheetFile(ctx, ref)
}

func (c *Client) ListSpreadsheetFiles(ctx context.Context, limit int) ([]*File, error) {
	return c.listFiles(ctx, "", limit)
}

func (c *Client) listFiles(ctx context.Context, condition string, limit int) ([]*File, error) {
	query := fmt.Sprintf("mimeType='%s' and trashed=false", spreadsheetMimeType)
	if condition != "" {
		query += " and " + condition
	}

	var out struct {
		Files []*File `json:"files"`
	}
	args := []string{
		"drive", "ls", "--all", "--max", strconv.Itoa(limit), "--query", query,
		"--fields", "files(id,name,mimeType,modifiedByMeTime)",
	}
	if err := c.runJSON(ctx, nil, &out, args...); err != nil {
		return nil, err
	}
	return out.Files, nil
}

func driveQueryString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	return strings.ReplaceAll(s, `'`, `\'`)
}

func spreadsheetID(ref string) string {
	ref = strings.TrimSpace(ref)
	if u, err := url.Parse(ref); err == nil && strings.EqualFold(u.Host, "docs.google.com") {
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		for i := 0; i+1 < len(parts); i++ {
			if parts[i] == "d" {
				return parts[i+1]
			}
		}
	}
	if len(ref) >= 20 && !strings.ContainsAny(ref, " ./") {
		return ref
	}
	return ""
}

func (c *Client) GetSpreadsheet(ctx context.Context, id string) (*Spreadsheet, error) {
	return c.getSpreadsheet(ctx, id, false)
}

func (c *Client) GetSpreadsheetWithGridData(ctx context.Context, id string) (*Spreadsheet, error) {
	return c.getSpreadsheet(ctx, id, true)
}

func (c *Client) getSpreadsheet(ctx context.Context, id string, grid bool) (*Spreadsheet, error) {
	var out spreadsheetResponse
	args := []string{sheetsCommand, "metadata", id}
	if grid {
		args = []string{sheetsCommand, "raw", id, "--include-grid-data"}
	}
	if err := c.runJSON(ctx, nil, &out, args...); err != nil {
		return nil, err
	}
	return out.spreadsheet(), nil
}

func (c *Client) WipeSpreadsheet(ctx context.Context, id string) error {
	spreadsheet, err := c.GetSpreadsheet(ctx, id)
	if err != nil {
		return err
	}
	wipeTitle := ""
	if sheet := findSheet(spreadsheet.Sheets, "Sheet1"); sheet != nil {
		wipeTitle = "gshoot-wipe"
		for findSheet(spreadsheet.Sheets, wipeTitle) != nil {
			wipeTitle += "-x"
		}
		if err := c.runJSON(ctx, nil, nil, sheetsCommand, "rename-tab", id, sheet.Title, wipeTitle); err != nil {
			return err
		}
	}
	if err := c.runJSON(ctx, nil, nil, sheetsCommand, "add-tab", id, "Sheet1", "--index", "0"); err != nil {
		return err
	}
	for _, sheet := range spreadsheet.Sheets {
		name := sheet.Title
		if wipeTitle != "" && strings.EqualFold(name, "Sheet1") {
			name = wipeTitle
		}
		if err := c.runJSON(ctx, nil, nil, sheetsCommand, "delete-tab", id, name, "--force"); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) FindSheet(ctx context.Context, id, name string) (*Sheet, error) {
	sheets, err := c.GetSheets(ctx, id)
	if err != nil {
		return nil, err
	}
	if name == "" {
		if len(sheets) == 0 {
			return nil, nil
		}
		return sheets[0], nil
	}
	return findSheet(sheets, name), nil
}

func findSheet(sheets []*Sheet, name string) *Sheet {
	for _, sheet := range sheets {
		if strings.EqualFold(sheet.Title, name) {
			return sheet
		}
	}
	return nil
}

func (c *Client) GetSheets(ctx context.Context, id string) ([]*Sheet, error) {
	spreadsheet, err := c.GetSpreadsheet(ctx, id)
	if err != nil {
		return nil, err
	}
	return spreadsheet.Sheets, nil
}

func (c *Client) GetRows(ctx context.Context, id, title string) (Rows, error) {
	var out struct {
		Values [][]any `json:"values"`
	}
	if err := c.runJSON(ctx, nil, &out, sheetsCommand, "get", id, quoteSheet(title)); err != nil {
		return nil, err
	}
	rows := make([][]string, 0, len(out.Values))
	for _, row := range out.Values {
		cells := make([]string, 0, len(row))
		for _, cell := range row {
			cells = append(cells, fmt.Sprint(cell))
		}
		rows = append(rows, cells)
	}
	return Rows(util.CSVRectangularize(rows)), nil
}

// Apply translates gshoot operations to named gog commands.
func (c *Client) Apply(ctx context.Context, id string, operations []Operation) ([]OperationResult, error) {
	results := make([]OperationResult, 0, len(operations))
	for _, operation := range operations {
		result, err := c.applyOperation(ctx, id, operation)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, nil
}

func (c *Client) applyOperation(ctx context.Context, id string, operation Operation) (OperationResult, error) {
	spreadsheet, err := c.GetSpreadsheet(ctx, id)
	if err != nil {
		return OperationResult{}, err
	}
	sheetByID := func(sheetID int64) (*Sheet, error) {
		for _, sheet := range spreadsheet.Sheets {
			if sheet.ID == sheetID {
				return sheet, nil
			}
		}
		return nil, fmt.Errorf("sheet id %d not found", sheetID)
	}

	switch {
	case operation.AddSheet != nil:
		add := operation.AddSheet
		args := []string{sheetsCommand, "add-tab", id, add.Title}
		if add.Index != nil {
			args = append(args, "--index", strconv.Itoa(*add.Index))
		}
		var out struct {
			SheetID int64 `json:"sheetId"`
		}
		if err := c.runJSON(ctx, nil, &out, args...); err != nil {
			return OperationResult{}, err
		}
		if add.GridProperties != nil {
			// Google creates regular sheets with this default grid size.
			current := &Sheet{
				ID:             out.SheetID,
				Title:          add.Title,
				GridProperties: &GridProperties{RowCount: 1000, ColumnCount: 26},
			}
			if err := c.resizeGrid(ctx, id, add.Title, add.GridProperties, current); err != nil {
				return OperationResult{}, err
			}
		}
		return OperationResult{AddedSheet: &Sheet{ID: out.SheetID, Title: add.Title}}, nil

	case operation.UpdateSheet != nil:
		update := operation.UpdateSheet
		sheet, err := sheetByID(update.SheetID)
		if err != nil {
			return OperationResult{}, err
		}
		if update.Title != nil {
			if err := c.runJSON(ctx, nil, nil, sheetsCommand, "rename-tab", id, sheet.Title, *update.Title); err != nil {
				return OperationResult{}, err
			}
			sheet.Title = *update.Title
		}
		if update.GridProperties != nil {
			return OperationResult{}, c.resizeGrid(ctx, id, sheet.Title, update.GridProperties, sheet)
		}
		return OperationResult{}, nil

	case operation.ClearCells != nil:
		clearOp := operation.ClearCells
		sheet, err := sheetByID(clearOp.Range.SheetID)
		if err != nil {
			return OperationResult{}, err
		}
		rng := gridRange(sheet, clearOp.Range)
		if err := c.runJSON(ctx, nil, nil, sheetsCommand, "clear", id, rng); err != nil {
			return OperationResult{}, err
		}
		if clearOp.All {
			for _, args := range [][]string{
				{sheetsCommand, "format", id, rng, "--format-json", "{}", "--format-fields", "userEnteredFormat"},
				{sheetsCommand, "validation", "clear", id, rng, "--filtered-rows-included"},
				{sheetsCommand, "update-note", id, rng, "--note", ""},
			} {
				if err := c.runJSON(ctx, nil, nil, args...); err != nil {
					return OperationResult{}, err
				}
			}
		}
		return OperationResult{}, nil

	case operation.PasteRows != nil:
		paste := operation.PasteRows
		sheet, err := sheetByID(paste.SheetID)
		if err != nil {
			return OperationResult{}, err
		}
		data, err := json.Marshal(paste.Rows)
		if err != nil {
			return OperationResult{}, err
		}
		cell := fmt.Sprintf("%s!%s%d", quoteSheet(sheet.Title), columnName(paste.ColumnIndex), paste.RowIndex+1)
		return OperationResult{}, c.runJSON(ctx, bytes.NewReader(data), nil, sheetsCommand, "update", id, cell, "--values-json", "@-", "--input", "USER_ENTERED")

	case operation.InsertDimension != nil:
		insert := operation.InsertDimension
		sheet, err := sheetByID(insert.SheetID)
		if err != nil {
			return OperationResult{}, err
		}
		args := []string{sheetsCommand, "insert", id, sheet.Title, insert.Dimension, strconv.Itoa(insert.Start), "--count", strconv.Itoa(insert.Count)}
		if insert.After {
			args = append(args, "--after")
		}
		if insert.InheritFromBefore != nil {
			args = append(args, fmt.Sprintf("--inherit-from-before=%t", *insert.InheritFromBefore))
		}
		return OperationResult{}, c.runJSON(ctx, nil, nil, args...)

	case operation.SetFilter != nil:
		sheet, err := sheetByID(operation.SetFilter.SheetID)
		if err != nil {
			return OperationResult{}, err
		}
		return OperationResult{}, c.runJSON(ctx, nil, nil, sheetsCommand, "filter", "set", id, gridRange(sheet, *operation.SetFilter), "--force")

	case operation.FormatCells != nil:
		format := operation.FormatCells
		sheet, err := sheetByID(format.Range.SheetID)
		if err != nil {
			return OperationResult{}, err
		}
		rng := gridRange(sheet, format.Range)
		if format.NumberFormat != nil {
			return OperationResult{}, c.runJSON(ctx, nil, nil, sheetsCommand, "number-format", id, rng,
				"--type", format.NumberFormat.Type, "--pattern", format.NumberFormat.Pattern)
		}
		return OperationResult{}, c.runJSON(ctx, nil, nil, sheetsCommand, "format", id, rng, "--format-json", "{}", "--format-fields", "userEnteredFormat")

	case operation.AutoResizeColumns != nil:
		sheet, err := sheetByID(operation.AutoResizeColumns.SheetID)
		if err != nil {
			return OperationResult{}, err
		}
		return OperationResult{}, c.runJSON(ctx, nil, nil, sheetsCommand, "resize-columns", id, columnRange(sheet.Title, *operation.AutoResizeColumns), "--auto")

	case operation.ResizeColumns != nil:
		update := operation.ResizeColumns
		sheet, err := sheetByID(update.Range.SheetID)
		if err != nil {
			return OperationResult{}, err
		}
		return OperationResult{}, c.runJSON(ctx, nil, nil, sheetsCommand, "resize-columns", id, columnRange(sheet.Title, update.Range), "--width", strconv.Itoa(update.PixelSize))

	case operation.CopyCells != nil:
		copyOp := operation.CopyCells
		source, err := sheetByID(copyOp.Source.SheetID)
		if err != nil {
			return OperationResult{}, err
		}
		destination, err := sheetByID(copyOp.Destination.SheetID)
		if err != nil {
			return OperationResult{}, err
		}
		return OperationResult{}, c.runJSON(ctx, nil, nil, sheetsCommand, "copy-paste", id,
			gridRange(source, copyOp.Source), gridRange(destination, copyOp.Destination),
			"--type", copyOp.Type)
	default:
		return OperationResult{}, errors.New("unsupported Sheets operation")
	}
}

func (c *Client) resizeGrid(ctx context.Context, id, name string, want *GridProperties, current *Sheet) error {
	if current == nil {
		spreadsheet, err := c.GetSpreadsheet(ctx, id)
		if err != nil {
			return err
		}
		current = findSheet(spreadsheet.Sheets, name)
	}
	if current == nil {
		return fmt.Errorf("sheet %q not found", name)
	}
	for _, dim := range []struct {
		label string
		have  int
		want  int
	}{
		{label: "rows", have: current.GridProperties.RowCount, want: want.RowCount},
		{label: "cols", have: current.GridProperties.ColumnCount, want: want.ColumnCount},
	} {
		if dim.want == 0 || dim.have == dim.want {
			continue
		}
		if dim.have < dim.want {
			if err := c.runJSON(ctx, nil, nil, sheetsCommand, "insert", id, name, dim.label, strconv.Itoa(dim.have), "--after", "--count", strconv.Itoa(dim.want-dim.have)); err != nil {
				return err
			}
			continue
		}
		apiDim := "ROWS"
		if dim.label == "cols" {
			apiDim = "COLUMNS"
		}
		if err := c.runJSON(ctx, nil, nil, sheetsCommand, "delete-dimension", id, name, "--dimension", apiDim,
			"--start", strconv.Itoa(dim.want+1), "--end", strconv.Itoa(dim.have), "--force"); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) runJSON(ctx context.Context, stdin io.Reader, dst any, args ...string) error {
	base := make([]string, 0, 3+len(args))
	base = append(base, "--json", "--no-input", "--color=never")
	base = append(base, args...)
	debugGogCall(base)
	// #nosec G204 -- arguments are passed directly without a shell.
	cmd := exec.CommandContext(ctx, c.gog, base...)
	cmd.Stdin = stdin
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("gog: %s", msg)
	}
	if dst == nil || stdout.Len() == 0 {
		return nil
	}
	if err := json.Unmarshal(stdout.Bytes(), dst); err != nil {
		return fmt.Errorf("decode gog output: %w", err)
	}
	return nil
}

// debugGogCall logs wrapped gog commands when GSHOOT_DEBUG is enabled.
func debugGogCall(args []string) {
	debug := strings.TrimSpace(os.Getenv("GSHOOT_DEBUG"))
	if debug == "" || debug == "0" || strings.EqualFold(debug, "false") {
		return
	}

	quoted := make([]string, len(args))
	for i, arg := range args {
		quoted[i] = arg
		if arg == "" || strings.ContainsAny(arg, " \t\r\n\"\\") {
			quoted[i] = strconv.Quote(arg)
		}
	}
	line := time.Now().UTC().Format("2006-01-02T15:04:05.000Z") + " gog " + strings.Join(quoted, " ")
	for len([]rune(line)) >= 80 {
		runes := []rune(line)
		cut := 77
		for i := cut; i > 1; i-- {
			if unicode.IsSpace(runes[i]) {
				cut = i
				break
			}
		}
		fmt.Fprintln(os.Stderr, strings.TrimRightFunc(string(runes[:cut]), unicode.IsSpace)+" \\")
		line = "  " + strings.TrimLeftFunc(string(runes[cut:]), unicode.IsSpace)
	}
	fmt.Fprintln(os.Stderr, line)
}

func quoteSheet(title string) string {
	return "'" + strings.ReplaceAll(title, "'", "''") + "'"
}

func gridRange(sheet *Sheet, rng GridRange) string {
	rows := sheet.GridProperties.RowCount
	cols := sheet.GridProperties.ColumnCount
	endRow, endCol := rng.EndRowIndex, rng.EndColumnIndex
	if endRow == 0 {
		endRow = rows
	}
	if endCol == 0 {
		endCol = cols
	}
	return fmt.Sprintf("%s!%s%d:%s%d", quoteSheet(sheet.Title), columnName(rng.StartColumnIndex), rng.StartRowIndex+1, columnName(endCol-1), endRow)
}

func columnRange(title string, rng ColumnRange) string {
	return fmt.Sprintf("%s!%s:%s", quoteSheet(title), columnName(rng.StartIndex), columnName(rng.EndIndex-1))
}

func columnName(index int) string {
	name := ""
	for index >= 0 {
		name = string(rune('A'+index%26)) + name
		index = index/26 - 1
	}
	return name
}
