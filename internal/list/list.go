package list

import (
	"context"
	"time"
)

// Spreadsheet contains listing data for one spreadsheet.
type Spreadsheet struct {
	ID           string
	Name         string
	ModifiedTime time.Time
}

// DriveSpreadsheet is the minimal Drive-side spreadsheet metadata.
type DriveSpreadsheet struct {
	ID           string
	Name         string
	ModifiedTime time.Time
}

// Client provides list operations for spreadsheets.
type Client interface {
	ListSpreadsheets(ctx context.Context, limit int) ([]DriveSpreadsheet, error)
}

// Service lists recent spreadsheets.
type Service struct {
	client Client
}

// NewService creates a listing service.
func NewService(client Client) Service {
	return Service{client: client}
}

// ListRecent returns recent spreadsheets.
func (s Service) ListRecent(ctx context.Context, limit int) ([]Spreadsheet, error) {
	driveItems, err := s.client.ListSpreadsheets(ctx, limit)
	if err != nil {
		return nil, err
	}

	items := make([]Spreadsheet, 0, len(driveItems))
	for _, item := range driveItems {
		items = append(items, Spreadsheet{
			ID:           item.ID,
			Name:         item.Name,
			ModifiedTime: item.ModifiedTime,
		})
	}

	return items, nil
}
