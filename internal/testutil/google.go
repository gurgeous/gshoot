package testutil

import (
	"context"
	"net/http"

	"github.com/gurgeous/gshoot/internal/google"
	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

// NewDriveTestClient creates a Google client with a Drive service pointed at serverURL.
func NewDriveTestClient(t TestingT, serverURL string) *google.Client {
	t.Helper()

	httpClient := &http.Client{
		Transport: &oauth2.Transport{
			Source: oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "token"}),
			Base:   http.DefaultTransport,
		},
	}
	ctx := context.Background()
	driveService, err := drive.NewService(
		context.Background(),
		option.WithHTTPClient(httpClient),
		option.WithEndpoint(serverURL+"/drive/v3/"),
	)
	if err != nil {
		t.Fatalf("drive.NewService() error = %v", err)
	}
	return &google.Client{Ctx: ctx, Drive: driveService}
}
