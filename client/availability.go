package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/rafa-garcia/go-playtomic-api/models"
)

// GetAvailability retrieves court availability from the Playtomic API.
// The API enforces a maximum window of 25 hours between StartMin and StartMax.
func (c *Client) GetAvailability(ctx context.Context, params *models.SearchAvailabilityParams) ([]models.CourtAvailability, error) {
	var availability []models.CourtAvailability
	err := c.sendRequest(ctx, http.MethodGet, "/availability", params.ToURLValues().Encode(), nil, &availability)
	if err != nil {
		return nil, fmt.Errorf("fetching availability: %w", err)
	}
	return availability, nil
}
