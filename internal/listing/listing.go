package listing

import (
	"context"
	"time"
)

// Spreadsheet contains listing data for one spreadsheet.
type Spreadsheet struct {
	ID           string
	Name         string
	ModifiedTime time.Time
	SheetNames   []string
}

// DriveSpreadsheet is the minimal Drive-side spreadsheet metadata.
type DriveSpreadsheet struct {
	ID           string
	Name         string
	ModifiedTime time.Time
}

// Client provides list operations for spreadsheets and sheets.
type Client interface {
	ListSpreadsheets(ctx context.Context, limit int) ([]DriveSpreadsheet, error)
	ListSheetNames(ctx context.Context, spreadsheetID string) ([]string, error)
}

// Service lists recent spreadsheets and their sheet names.
type Service struct {
	client Client
}

// NewService creates a listing service.
func NewService(client Client) Service {
	return Service{client: client}
}

// ListRecent returns recent spreadsheets with sheet names attached.
func (s Service) ListRecent(ctx context.Context, limit int) ([]Spreadsheet, error) {
	driveItems, err := s.client.ListSpreadsheets(ctx, limit)
	if err != nil {
		return nil, err
	}

	items := make([]Spreadsheet, 0, len(driveItems))
	for _, item := range driveItems {
		sheetNames, err := s.client.ListSheetNames(ctx, item.ID)
		if err != nil {
			return nil, err
		}

		items = append(items, Spreadsheet{
			ID:           item.ID,
			Name:         item.Name,
			ModifiedTime: item.ModifiedTime,
			SheetNames:   sheetNames,
		})
	}

	return items, nil
}
