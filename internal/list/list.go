package list

import (
	"context"
	"fmt"
	"time"

	"github.com/gurgeous/gshoot/internal/google"
	"google.golang.org/api/drive/v3"
)

// Recent returns recent spreadsheets ordered by modified time.
func Recent(ctx context.Context, client *google.Client, limit int) ([]*drive.File, time.Duration, error) {
	start := time.Now()
	res, err := client.Drive.Files.List().
		Context(ctx).
		Q("mimeType='application/vnd.google-apps.spreadsheet' and trashed=false").
		OrderBy("modifiedTime desc,name").
		PageSize(int64(limit)).
		Fields("files(id,name,modifiedTime)").
		Do()
	if err != nil {
		return nil, 0, fmt.Errorf("list spreadsheets: %w", err)
	}
	return res.Files, time.Since(start), nil
}
