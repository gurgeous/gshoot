package googletest

import (
	"context"
	"net/http"
	"testing"

	"github.com/gurgeous/gshoot/google"
	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

// NewClient creates a Google client with Drive and Sheets services pointed at serverURL.
func NewClient(t *testing.T, serverURL string) *google.Client {
	t.Helper()

	httpClient := &http.Client{
		Transport: &oauth2.Transport{
			Source: oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "token"}),
			Base:   http.DefaultTransport,
		},
	}
	driveService, err := drive.NewService(
		context.Background(),
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(serverURL+"/drive/v3/"),
	)
	if err != nil {
		t.Fatalf("drive.NewService() error = %v", err)
	}
	sheetsService, err := sheets.NewService(
		context.Background(),
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(serverURL+"/"),
	)
	if err != nil {
		t.Fatalf("sheets.NewService() error = %v", err)
	}
	return &google.Client{Drive: driveService, Sheets: sheetsService}
}
