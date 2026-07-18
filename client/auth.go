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
	RefreshToken          string `json:"refresh_token"`
}

// accessToken returns a valid access token, refreshing it if it's missing or
// close to expiring.
//
// A token supplied via WithAccessToken (accessTokenExpiration left zero) is
// trusted as-is with no proactive expiration check - we don't know its real
// expiry, so it's used until a request actually fails with 401. Once this
// client performs its own exchange, accessTokenExpiration is set to a real
// value and checked proactively from then on.
func (c *Client) accessTokenFor(ctx context.Context) (string, error) {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()

	if c.accessToken != "" && (c.accessTokenExpiration.IsZero() || time.Now().Before(c.accessTokenExpiration.Add(-tokenExpirationBuffer))) {
		return c.accessToken, nil
	}

	if c.refreshToken == "" {
		return "", fmt.Errorf("no refresh token configured: set REFRESH_TOKEN (see client.WithRefreshToken)")
	}

	if err := c.refreshAccessTokenLocked(ctx); err != nil {
		return "", err
	}

	return c.accessToken, nil
}

// AccessToken returns the access token currently held by the client. If this
// client refreshed it (e.g. after a WithAccessToken-seeded token failed),
// this will differ from the value originally configured - callers that want
// to persist a refreshed access token across process runs (e.g. back into a
// secret store) should read it from here once requests are done.
func (c *Client) AccessToken() string {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()
	return c.accessToken
}

// RefreshToken returns the refresh token currently held by the client. The
// Playtomic API rotates the refresh token on every exchange and invalidates
// the previous one - after this client performs an exchange, this will
// differ from the value originally passed to WithRefreshToken. Callers that
// want to keep a long-lived refresh token working across process runs (e.g.
// back into a secret store) must read it from here once requests are done
// and persist it; reusing the original value on the next run will fail.
func (c *Client) RefreshToken() string {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()
	return c.refreshToken
}

// invalidateAccessToken forces the next accessTokenFor call to fetch a fresh
// token, used when a request unexpectedly comes back 401 mid-run.
func (c *Client) invalidateAccessToken() {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()
	c.accessToken = ""
	c.accessTokenExpiration = time.Time{}
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
	if tr.RefreshToken != "" {
		// The API rotates the refresh token on every exchange and invalidates
		// the one we just used - keep using the new one for the rest of this
		// process's lifetime, and let the caller (RefreshToken) read it back
		// to persist it, or the next exchange will fail.
		c.refreshToken = tr.RefreshToken
	}

	return nil
}
