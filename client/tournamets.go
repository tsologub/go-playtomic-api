package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/rafa-garcia/go-playtomic-api/models"
)

func (c *Client) GetTournaments(ctx context.Context, params *models.SearchTournamentsParams) ([]models.Tournament, error) {
	var tournaments []models.Tournament
	err := c.sendRequest(ctx, http.MethodGet, "/tournaments", params.ToURLValues().Encode(), nil, &tournaments)
	if err != nil {
		return nil, fmt.Errorf("fetching tournaments: %w", err)
	}
	return tournaments, nil
}
