package list

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

// NewGoogleClient constructs a Google-backed listing client.
func NewGoogleClient(ctx context.Context, tokenSource oauth2.TokenSource) (*GoogleClient, error) {
	httpClient := oauth2.NewClient(ctx, tokenSource)

	driveService, err := drive.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("create drive service: %w", err)
	}

	return &GoogleClient{
		drive: driveService,
	}, nil
}

// GoogleClient lists spreadsheets through the Google APIs.
type GoogleClient struct {
	drive *drive.Service
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
