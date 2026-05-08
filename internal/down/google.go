package down

import (
	"context"
	"fmt"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

// NewGoogleClient constructs a Google-backed download client.
func NewGoogleClient(ctx context.Context, tokenSource oauth2.TokenSource) (*GoogleClient, error) {
	httpClient := oauth2.NewClient(ctx, tokenSource)

	driveService, err := drive.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("create drive service: %w", err)
	}

	sheetsService, err := sheets.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("create sheets service: %w", err)
	}

	return &GoogleClient{
		drive:  driveService,
		sheets: sheetsService,
	}, nil
}

// GoogleClient reads spreadsheets through the Google APIs.
type GoogleClient struct {
	drive  *drive.Service
	sheets *sheets.Service
}

// ListSpreadsheets returns recent spreadsheets ordered by modified time.
func (c *GoogleClient) ListSpreadsheets(ctx context.Context) ([]DriveSpreadsheet, error) {
	items := make([]DriveSpreadsheet, 0, 64)
	pageToken := ""
	for {
		call := c.drive.Files.List().
			Context(ctx).
			Q("mimeType='application/vnd.google-apps.spreadsheet' and trashed=false").
			OrderBy("modifiedTime desc,name").
			PageSize(1000).
			Fields("nextPageToken,files(id,name,modifiedTime)")
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}

		res, err := call.Do()
		if err != nil {
			return nil, fmt.Errorf("list spreadsheets: %w", err)
		}

		for _, file := range res.Files {
			modifiedTime, err := time.Parse(time.RFC3339, file.ModifiedTime)
			if err != nil {
				return nil, fmt.Errorf("parse modified time for %q: %w", file.Name, err)
			}
			items = append(items, DriveSpreadsheet{
				ID:           file.Id,
				Name:         file.Name,
				ModifiedTime: modifiedTime,
			})
		}

		if res.NextPageToken == "" {
			break
		}
		pageToken = res.NextPageToken
	}
	return items, nil
}

// ListSheets returns sheet metadata for one spreadsheet.
func (c *GoogleClient) ListSheets(ctx context.Context, spreadsheetID string) ([]Sheet, error) {
	res, err := c.sheets.Spreadsheets.Get(spreadsheetID).
		Context(ctx).
		Fields("properties(title),sheets(properties(sheetId,title))").
		Do()
	if err != nil {
		return nil, fmt.Errorf("list sheets for %s: %w", spreadsheetID, err)
	}

	items := make([]Sheet, 0, len(res.Sheets))
	for _, sheet := range res.Sheets {
		if sheet.Properties == nil {
			continue
		}
		items = append(items, Sheet{
			ID:    sheet.Properties.SheetId,
			Title: sheet.Properties.Title,
		})
	}
	return items, nil
}

// GetValues returns all values from one sheet.
func (c *GoogleClient) GetValues(ctx context.Context, spreadsheetID, sheetTitle string) ([][]string, error) {
	res, err := c.sheets.Spreadsheets.Values.Get(spreadsheetID, sheetRange(sheetTitle)).
		Context(ctx).
		Do()
	if err != nil {
		return nil, fmt.Errorf("get values for %s/%s: %w", spreadsheetID, sheetTitle, err)
	}

	rows := make([][]string, 0, len(res.Values))
	for _, row := range res.Values {
		cells := make([]string, 0, len(row))
		for _, cell := range row {
			cells = append(cells, fmt.Sprint(cell))
		}
		rows = append(rows, cells)
	}
	return rows, nil
}

func sheetRange(sheetTitle string) string {
	escaped := strings.ReplaceAll(sheetTitle, "'", "''")
	return fmt.Sprintf("'%s'", escaped)
}
