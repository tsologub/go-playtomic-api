package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// tokenExpirationLayout matches the timestamp format returned by the auth
// endpoint, e.g. "2026-07-18T06:04:09" (UTC, no offset).
const tokenExpirationLayout = "2006-01-02T15:04:05"

// tokenExpirationBuffer is how far ahead of the reported expiration we
// proactively refresh, to avoid racing a token that expires mid-request.
const tokenExpirationBuffer = 60 * time.Second

type tokenRequest struct {
	RequestedUserRoles []string `json:"requested_user_roles"`
	RefreshToken       string   `json:"refresh_token"`
}

type tokenResponse struct {
	AccessToken           string `json:"access_token"`
	AccessTokenExpiration string `json:"access_token_expiration"`
}

// accessToken returns a valid access token, refreshing it if it's missing or
// close to expiring.
func (c *Client) accessTokenFor(ctx context.Context) (string, error) {
	if c.refreshToken == "" {
		return "", fmt.Errorf("no refresh token configured: set REFRESH_TOKEN (see client.WithRefreshToken)")
	}

	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()

	if c.accessToken != "" && time.Now().Before(c.accessTokenExpiration.Add(-tokenExpirationBuffer)) {
		return c.accessToken, nil
	}

	if err := c.refreshAccessTokenLocked(ctx); err != nil {
		return "", err
	}

	return c.accessToken, nil
}

// invalidateAccessToken forces the next accessTokenFor call to fetch a fresh
// token, used when a request unexpectedly comes back 401 mid-run.
func (c *Client) invalidateAccessToken() {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()
	c.accessToken = ""
}

// refreshAccessTokenLocked exchanges the refresh token for a new access
// token. Callers must hold c.tokenMu.
func (c *Client) refreshAccessTokenLocked(ctx context.Context) error {
	reqBody, err := json.Marshal(tokenRequest{
		RequestedUserRoles: []string{"ROLE_CUSTOMER"},
		RefreshToken:       c.refreshToken,
	})
	if err != nil {
		return fmt.Errorf("encoding token request: %w", err)
	}

	reqURL := c.authURL + "/v3/auth/token"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("creating token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("requesting access token: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return parseAPIError(resp.StatusCode, respBody)
	}

	var tr tokenResponse
	if err := json.Unmarshal(respBody, &tr); err != nil {
		return fmt.Errorf("decoding token response: %w", err)
	}
	if tr.AccessToken == "" {
		return fmt.Errorf("token response missing access_token")
	}

	expiration, err := time.ParseInLocation(tokenExpirationLayout, tr.AccessTokenExpiration, time.UTC)
	if err != nil {
		return fmt.Errorf("parsing access_token_expiration %q: %w", tr.AccessTokenExpiration, err)
	}

	c.accessToken = tr.AccessToken
	c.accessTokenExpiration = expiration

	return nil
}
