package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/rafa-garcia/go-playtomic-api/models"
)

// maxClassesPages caps pagination to avoid an unbounded loop if the API ever
// returns full pages indefinitely (100 pages * 50 = 5000 classes).
const maxClassesPages = 100

// GetClasses retrieves classes from the Playtomic API, paging through results.
//
// The endpoint caps page size at 50, so this walks pages 0..N (accumulating
// results) until a page returns fewer than a full page, or the safety cap is
// reached. The caller's params are used as-is; Size and Page are managed here.
func (c *Client) GetClasses(ctx context.Context, params *models.SearchClassesParams) ([]models.Class, error) {
	params.Size = models.MaxClassesPageSize

	var classes []models.Class
	for page := 0; page < maxClassesPages; page++ {
		params.Page = page

		var pageClasses []models.Class
		err := c.sendRequest(ctx, http.MethodGet, "/classes", params.ToURLValues().Encode(), nil, &pageClasses)
		if err != nil {
			return nil, fmt.Errorf("fetching classes: %w", err)
		}

		classes = append(classes, pageClasses...)

		// A short (or empty) page means we've reached the end.
		if len(pageClasses) < models.MaxClassesPageSize {
			break
		}
	}

	return classes, nil
}
