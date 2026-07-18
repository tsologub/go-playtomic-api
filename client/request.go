package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// sendRequest sends a request to the Playtomic API and decodes the response
func (c *Client) sendRequest(ctx context.Context, method, endpoint string, queryParams string, body io.Reader, result interface{}) error {
	respBody, statusCode, err := c.doAuthenticated(ctx, method, endpoint, queryParams, body, false)
	if err != nil {
		return err
	}

	if statusCode != http.StatusOK {
		return parseAPIError(statusCode, respBody)
	}

	if err := json.Unmarshal(respBody, result); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}

	return nil
}

// doAuthenticated attaches a Bearer access token and performs the request,
// retrying network failures per c.maxRetries. On a 401, it invalidates the
// cached access token and retries the whole request once with a fresh one -
// unless this is already a retried call, in which case it returns the 401
// as-is so the caller doesn't loop forever on a bad refresh token.
func (c *Client) doAuthenticated(ctx context.Context, method, endpoint, queryParams string, body io.Reader, retriedAuth bool) ([]byte, int, error) {
	token, err := c.accessTokenFor(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("getting access token: %w", err)
	}

	reqURL := fmt.Sprintf("%s%s?%s", c.baseURL, endpoint, queryParams)

	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return nil, 0, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Authorization", "Bearer "+token)

	var resp *http.Response
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		resp, err = c.httpClient.Do(req)
		if err == nil {
			break
		}

		if attempt == c.maxRetries {
			return nil, 0, fmt.Errorf("sending request after %d attempts: %w", c.maxRetries, err)
		}

		select {
		case <-time.After(time.Duration(attempt) * 500 * time.Millisecond):
		case <-ctx.Done():
			return nil, 0, ctx.Err()
		}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized && !retriedAuth {
		c.invalidateAccessToken()
		return c.doAuthenticated(ctx, method, endpoint, queryParams, body, true)
	}

	return respBody, resp.StatusCode, nil
}
