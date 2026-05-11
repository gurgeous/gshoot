package smoke

import (
	"context"
	"fmt"
	"strings"

	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	gsheets "google.golang.org/api/sheets/v4"
)

// NewGoogleClient constructs a Google-backed smoke client.
func NewGoogleClient(ctx context.Context, tokenSource oauth2.TokenSource) (*GoogleClient, error) {
	httpClient := oauth2.NewClient(ctx, tokenSource)

	driveService, err := drive.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("create drive service: %w", err)
	}

	sheetsService, err := gsheets.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("create sheets service: %w", err)
	}

	return &GoogleClient{
		drive:  driveService,
		sheets: sheetsService,
	}, nil
}

// GoogleClient manages smoke fixtures through the Google APIs.
type GoogleClient struct {
	drive  *drive.Service
	sheets *gsheets.Service
}

// ResetDownFixture creates or resets one sheet with a known download fixture.
func (c *GoogleClient) ResetDownFixture(ctx context.Context, spreadsheetName, sheetName string, values [][]string) (string, error) {
	spreadsheet, err := c.findOrCreateSpreadsheet(ctx, spreadsheetName)
	if err != nil {
		return "", err
	}

	if err := c.resetSheets(ctx, spreadsheet.SpreadsheetId, sheetName, spreadsheet.Sheets); err != nil {
		return "", err
	}
	if err := c.clearSheet(ctx, spreadsheet.SpreadsheetId, sheetName); err != nil {
		return "", err
	}
	if err := c.writeValues(ctx, spreadsheet.SpreadsheetId, sheetName, values); err != nil {
		return "", err
	}

	return spreadsheet.SpreadsheetId, nil
}

func (c *GoogleClient) findOrCreateSpreadsheet(ctx context.Context, name string) (*gsheets.Spreadsheet, error) {
	pageToken := ""
	for {
		call := c.drive.Files.List().
			Context(ctx).
			Q("mimeType='application/vnd.google-apps.spreadsheet' and trashed=false").
			OrderBy("modifiedByMeTime desc, name").
			PageSize(1000).
			Fields("nextPageToken,files(id,name)")
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		files, err := call.Do()
		if err != nil {
			return nil, fmt.Errorf("list spreadsheets: %w", err)
		}

		for _, file := range files.Files {
			if file.Name == name {
				return c.sheets.Spreadsheets.Get(file.Id).
					Context(ctx).
					Fields("spreadsheetId,sheets(properties(sheetId,title))").
					Do()
			}
		}

		if files.NextPageToken == "" {
			break
		}
		pageToken = files.NextPageToken
	}
	spreadsheet, err := c.sheets.Spreadsheets.Create(&gsheets.Spreadsheet{
		Properties: &gsheets.SpreadsheetProperties{Title: name},
	}).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("create spreadsheet %q: %w", name, err)
	}
	return spreadsheet, nil
}

func (c *GoogleClient) resetSheets(ctx context.Context, spreadsheetID, sheetName string, current []*gsheets.Sheet) error {
	if len(current) == 0 {
		_, err := c.sheets.Spreadsheets.BatchUpdate(spreadsheetID, &gsheets.BatchUpdateSpreadsheetRequest{
			Requests: []*gsheets.Request{
				{
					AddSheet: &gsheets.AddSheetRequest{
						Properties: &gsheets.SheetProperties{Title: sheetName},
					},
				},
			},
		}).Context(ctx).Do()
		if err != nil {
			return fmt.Errorf("add smoke sheet: %w", err)
		}
		return nil
	}

	requests := make([]*gsheets.Request, 0, len(current))
	first := current[0].Properties
	if first != nil && first.Title != sheetName {
		requests = append(requests, &gsheets.Request{
			UpdateSheetProperties: &gsheets.UpdateSheetPropertiesRequest{
				Fields: "title",
				Properties: &gsheets.SheetProperties{
					SheetId: first.SheetId,
					Title:   sheetName,
				},
			},
		})
	}
	for _, sheet := range current[1:] {
		if sheet.Properties == nil {
			continue
		}
		requests = append(requests, &gsheets.Request{
			DeleteSheet: &gsheets.DeleteSheetRequest{SheetId: sheet.Properties.SheetId},
		})
	}
	if len(requests) == 0 {
		return nil
	}

	_, err := c.sheets.Spreadsheets.BatchUpdate(spreadsheetID, &gsheets.BatchUpdateSpreadsheetRequest{
		Requests: requests,
	}).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("reset smoke sheets: %w", err)
	}
	return nil
}

func (c *GoogleClient) clearSheet(ctx context.Context, spreadsheetID, sheetName string) error {
	_, err := c.sheets.Spreadsheets.Values.Clear(spreadsheetID, quotedSheetName(sheetName), &gsheets.ClearValuesRequest{}).
		Context(ctx).
		Do()
	if err != nil {
		return fmt.Errorf("clear smoke sheet: %w", err)
	}
	return nil
}

func (c *GoogleClient) writeValues(ctx context.Context, spreadsheetID, sheetName string, values [][]string) error {
	rows := make([][]any, 0, len(values))
	for _, row := range values {
		cells := make([]any, 0, len(row))
		for _, cell := range row {
			cells = append(cells, cell)
		}
		rows = append(rows, cells)
	}

	_, err := c.sheets.Spreadsheets.Values.Update(spreadsheetID, quotedSheetName(sheetName), &gsheets.ValueRange{
		Values: rows,
	}).
		ValueInputOption("RAW").
		Context(ctx).
		Do()
	if err != nil {
		return fmt.Errorf("write smoke values: %w", err)
	}
	return nil
}

func quotedSheetName(sheetName string) string {
	return "'" + strings.ReplaceAll(sheetName, "'", "''") + "'"
}
