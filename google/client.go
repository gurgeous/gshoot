package google

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os/exec"
	"slices"
	"strconv"
	"strings"

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
	return &File{ID: out.ID, Name: out.Name, MimeType: spreadsheetMimeType}, nil
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
		return &File{ID: id, Name: spreadsheet.Title, MimeType: spreadsheetMimeType}, nil
	}

	files, err := c.listFiles(ctx, "name = '"+driveQueryString(ref)+"'")
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
		return &File{ID: id, Name: spreadsheet.Title, MimeType: spreadsheetMimeType}, nil
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
	files, err := c.listFiles(ctx, "")
	if err != nil {
		return nil, err
	}
	limit = util.Clamp(limit, 1, 100)
	return files[:min(limit, len(files))], nil
}

func (c *Client) listFiles(ctx context.Context, condition string) ([]*File, error) {
	query := fmt.Sprintf("mimeType='%s' and trashed=false", spreadsheetMimeType)
	if condition != "" {
		query += " and " + condition
	}

	files := []*File{}
	page := ""
	for {
		var out struct {
			Files         []*File `json:"files"`
			NextPageToken string  `json:"nextPageToken"`
		}
		args := []string{
			"drive", "ls", "--all", "--max", "1000", "--query", query,
			"--fields", "nextPageToken,files(id,name,mimeType,modifiedByMeTime)",
		}
		if page != "" {
			args = append(args, "--page", page)
		}
		if err := c.runJSON(ctx, nil, &out, args...); err != nil {
			return nil, err
		}
		files = append(files, out.Files...)
		page = out.NextPageToken
		if page == "" {
			break
		}
	}
	slices.SortStableFunc(files, func(a, b *File) int {
		if n := strings.Compare(b.ModifiedByMeTime, a.ModifiedByMeTime); n != 0 {
			return n
		}
		return strings.Compare(a.Name, b.Name)
	})
	return files, nil
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

func (c *Client) GetSpreadsheetWithGridData(ctx context.Context, id string, _ ...string) (*Spreadsheet, error) {
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
	title := "gshoot-wipe"
	for findSheet(spreadsheet.Sheets, title) != nil {
		title += "-x"
	}
	if sheet := findSheet(spreadsheet.Sheets, "Sheet1"); sheet != nil {
		if err := c.runJSON(ctx, nil, nil, sheetsCommand, "rename-tab", id, sheet.Title, title); err != nil {
			return err
		}
	}
	if err := c.runJSON(ctx, nil, nil, sheetsCommand, "add-tab", id, "Sheet1", "--index", "0"); err != nil {
		return err
	}
	for _, sheet := range spreadsheet.Sheets {
		name := sheet.Title
		if strings.EqualFold(name, "Sheet1") {
			name = title
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
	if err := c.runJSON(ctx, nil, &out, sheetsCommand, "get", id, sheetRange(title)); err != nil {
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

// BatchUpdate translates gshoot's narrow request model to named gog commands.
func (c *Client) BatchUpdate(ctx context.Context, id string, requests []Request) (*BatchUpdateResponse, error) {
	response := &BatchUpdateResponse{}
	for _, request := range requests {
		reply, err := c.applyRequest(ctx, id, request)
		if err != nil {
			return nil, err
		}
		response.Replies = append(response.Replies, reply)
	}
	return response, nil
}

func (c *Client) applyRequest(ctx context.Context, id string, request Request) (Reply, error) {
	spreadsheet, err := c.GetSpreadsheet(ctx, id)
	if err != nil {
		return Reply{}, err
	}
	sheetName := func(sheetID int64) (string, error) {
		for _, sheet := range spreadsheet.Sheets {
			if sheet.ID == sheetID {
				return sheet.Title, nil
			}
		}
		return "", fmt.Errorf("sheet id %d not found", sheetID)
	}

	switch {
	case request.AddSheet != nil:
		p := request.AddSheet.Properties
		args := []string{sheetsCommand, "add-tab", id, p.Title}
		if p.Index != nil {
			args = append(args, "--index", strconv.Itoa(*p.Index))
		}
		var out struct {
			SheetID int64 `json:"sheetId"`
		}
		if err := c.runJSON(ctx, nil, &out, args...); err != nil {
			return Reply{}, err
		}
		if p.GridProperties != nil {
			if err := c.resizeGrid(ctx, id, p.Title, p.GridProperties, nil); err != nil {
				return Reply{}, err
			}
		}
		return Reply{AddSheet: &AddSheetReply{Properties: Sheet{ID: out.SheetID, Title: p.Title}}}, nil

	case request.DeleteSheet != nil:
		name, err := sheetName(request.DeleteSheet.SheetID)
		if err != nil {
			return Reply{}, err
		}
		return Reply{}, c.runJSON(ctx, nil, nil, sheetsCommand, "delete-tab", id, name, "--force")

	case request.UpdateSheetProperties != nil:
		p := request.UpdateSheetProperties.Properties
		name, err := sheetName(*p.SheetID)
		if err != nil {
			return Reply{}, err
		}
		if strings.Contains(request.UpdateSheetProperties.Fields, "title") {
			if err := c.runJSON(ctx, nil, nil, sheetsCommand, "rename-tab", id, name, p.Title); err != nil {
				return Reply{}, err
			}
			name = p.Title
		}
		if p.GridProperties != nil {
			return Reply{}, c.resizeGrid(ctx, id, name, p.GridProperties, findSheet(spreadsheet.Sheets, name))
		}
		return Reply{}, nil

	case request.UpdateCells != nil:
		name, err := sheetName(request.UpdateCells.Range.SheetID)
		if err != nil {
			return Reply{}, err
		}
		rng := gridRange(name, request.UpdateCells.Range, spreadsheet)
		if err := c.runJSON(ctx, nil, nil, sheetsCommand, "clear", id, rng); err != nil {
			return Reply{}, err
		}
		if request.UpdateCells.Fields == "*" {
			for _, args := range [][]string{
				{sheetsCommand, "format", id, rng, "--format-json", "{}", "--format-fields", "userEnteredFormat"},
				{sheetsCommand, "validation", "clear", id, rng, "--filtered-rows-included"},
				{sheetsCommand, "update-note", id, rng, "--note", ""},
			} {
				if err := c.runJSON(ctx, nil, nil, args...); err != nil {
					return Reply{}, err
				}
			}
		}
		return Reply{}, nil

	case request.PasteData != nil:
		name, err := sheetName(request.PasteData.Coordinate.SheetID)
		if err != nil {
			return Reply{}, err
		}
		reader := csv.NewReader(strings.NewReader(request.PasteData.Data))
		rows, err := reader.ReadAll()
		if err != nil {
			return Reply{}, err
		}
		data, err := json.Marshal(rows)
		if err != nil {
			return Reply{}, err
		}
		input := "USER_ENTERED"
		if request.PasteData.Type == "PASTE_VALUES" {
			input = "RAW"
		}
		cell := a1Cell(name, request.PasteData.Coordinate.RowIndex, request.PasteData.Coordinate.ColumnIndex)
		return Reply{}, c.runJSON(ctx, bytes.NewReader(data), nil, sheetsCommand, "update", id, cell, "--values-json", "@-", "--input", input)

	case request.SetBasicFilter != nil:
		filter := request.SetBasicFilter.Filter.Range
		name, err := sheetName(filter.SheetID)
		if err != nil {
			return Reply{}, err
		}
		return Reply{}, c.runJSON(ctx, nil, nil, sheetsCommand, "filter", "set", id, gridRange(name, filter, spreadsheet), "--force")

	case request.RepeatCell != nil:
		repeat := request.RepeatCell
		name, err := sheetName(repeat.Range.SheetID)
		if err != nil {
			return Reply{}, err
		}
		rng := gridRange(name, repeat.Range, spreadsheet)
		if repeat.Cell.UserEnteredFormat != nil && repeat.Cell.UserEnteredFormat.NumberFormat != nil {
			format := repeat.Cell.UserEnteredFormat.NumberFormat
			return Reply{}, c.runJSON(ctx, nil, nil, sheetsCommand, "number-format", id, rng, "--type", format.Type, "--pattern", format.Pattern)
		}
		return Reply{}, c.runJSON(ctx, nil, nil, sheetsCommand, "format", id, rng, "--format-json", "{}", "--format-fields", "userEnteredFormat")

	case request.AutoResizeDimensions != nil:
		dim := request.AutoResizeDimensions.Dimensions
		name, err := sheetName(dim.SheetID)
		if err != nil {
			return Reply{}, err
		}
		return Reply{}, c.runJSON(ctx, nil, nil, sheetsCommand, "resize-columns", id, columnRange(name, dim), "--auto")

	case request.UpdateDimensionProperties != nil:
		update := request.UpdateDimensionProperties
		name, err := sheetName(update.Range.SheetID)
		if err != nil {
			return Reply{}, err
		}
		return Reply{}, c.runJSON(ctx, nil, nil, sheetsCommand, "resize-columns", id, columnRange(name, update.Range), "--width", strconv.Itoa(update.Properties.PixelSize))

	case request.CopyPaste != nil:
		copyReq := request.CopyPaste
		sourceName, err := sheetName(copyReq.Source.SheetID)
		if err != nil {
			return Reply{}, err
		}
		destName, err := sheetName(copyReq.Destination.SheetID)
		if err != nil {
			return Reply{}, err
		}
		return Reply{}, c.runJSON(ctx, nil, nil, sheetsCommand, "copy-paste", id,
			gridRange(sourceName, copyReq.Source, spreadsheet), gridRange(destName, copyReq.Destination, spreadsheet),
			"--type", strings.TrimPrefix(copyReq.PasteType, "PASTE_"))
	default:
		return Reply{}, errors.New("unsupported Sheets update")
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
		apiDim := strings.ToUpper(dim.label)
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

func sheetRange(title string) string {
	return quoteSheet(title)
}

func quoteSheet(title string) string {
	return "'" + strings.ReplaceAll(title, "'", "''") + "'"
}

func a1Cell(title string, row, column int) string {
	return fmt.Sprintf("%s!%s%d", quoteSheet(title), columnName(column), row+1)
}

func gridRange(title string, rng GridRange, spreadsheet *Spreadsheet) string {
	rows, cols := 1, 1
	if sheet := findSheet(spreadsheet.Sheets, title); sheet != nil {
		rows = sheet.GridProperties.RowCount
		cols = sheet.GridProperties.ColumnCount
	}
	endRow, endCol := rng.EndRowIndex, rng.EndColumnIndex
	if endRow == 0 {
		endRow = rows
	}
	if endCol == 0 {
		endCol = cols
	}
	return fmt.Sprintf("%s!%s%d:%s%d", quoteSheet(title), columnName(rng.StartColumnIndex), rng.StartRowIndex+1, columnName(endCol-1), endRow)
}

func columnRange(title string, rng DimensionRange) string {
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
