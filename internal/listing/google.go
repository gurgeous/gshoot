package listing

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

// NewGoogleClient constructs a Google-backed listing client.
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

// GoogleClient lists spreadsheets through the Google APIs.
type GoogleClient struct {
	drive  *drive.Service
	sheets *sheets.Service
}

// ListSpreadsheets returns recent spreadsheets ordered by modified time.
func (c *GoogleClient) ListSpreadsheets(ctx context.Context, limit int) ([]DriveSpreadsheet, error) {
	call := c.drive.Files.List().
		Context(ctx).
		Q("mimeType='application/vnd.google-apps.spreadsheet' and trashed=false").
		OrderBy("modifiedTime desc,name").
		PageSize(int64(limit)).
		Fields("files(id,name,modifiedTime)")

	res, err := call.Do()
	if err != nil {
		return nil, fmt.Errorf("list spreadsheets: %w", err)
	}

	items := make([]DriveSpreadsheet, 0, len(res.Files))
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

	return items, nil
}

// ListSheetNames returns sheet names for one spreadsheet.
func (c *GoogleClient) ListSheetNames(ctx context.Context, spreadsheetID string) ([]string, error) {
	res, err := c.sheets.Spreadsheets.Get(spreadsheetID).
		Context(ctx).
		Fields("sheets(properties(title))").
		Do()
	if err != nil {
		return nil, fmt.Errorf("list sheets for %s: %w", spreadsheetID, err)
	}

	names := make([]string, 0, len(res.Sheets))
	for _, sheet := range res.Sheets {
		if sheet.Properties != nil {
			names = append(names, sheet.Properties.Title)
		}
	}

	return names, nil
}
