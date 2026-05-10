package google

import (
	"context"
	"fmt"

	"github.com/gurgeous/gshoot/internal/auth"
	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

// Client holds shared Google API services.
type Client struct {
	Drive  *drive.Service
	Sheets *sheets.Service
}

// NewClient creates a Google API client with auth for a command's scopes.
func NewClient(ctx context.Context, cmd auth.Command) (*Client, error) {
	// auth
	resolved, err := auth.Resolve(auth.Options{Command: cmd})
	if err != nil {
		return nil, err
	}
	tokenSource, err := auth.NewTokenSource(ctx, resolved)
	if err != nil {
		return nil, err
	}

	// services
	httpClient := oauth2.NewClient(ctx, tokenSource)
	drive, err := drive.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("create drive service: %w", err)
	}
	sheets, err := sheets.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("create sheets service: %w", err)
	}

	return &Client{Drive: drive, Sheets: sheets}, nil
}
