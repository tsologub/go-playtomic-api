package client

import (
	"net/http"
	"time"
)

// Option defines a function that configures the client
type Option func(*Client)

// WithBaseURL sets a custom base URL for the client
func WithBaseURL(url string) Option {
	return func(c *Client) {
		c.baseURL = url
	}
}

// WithAuthBaseURL sets a custom base URL for the token exchange endpoint
// (useful for testing against a mock server).
func WithAuthBaseURL(url string) Option {
	return func(c *Client) {
		c.authURL = url
	}
}

// WithRefreshToken sets the refresh token used to obtain access tokens when
// none is cached (or the cached one fails). Required for all requests, since
// the Playtomic API requires a Bearer access token on every call.
func WithRefreshToken(refreshToken string) Option {
	return func(c *Client) {
		c.refreshToken = refreshToken
	}
}

// WithAccessToken seeds the client with an already-obtained access token, so
// it's used directly without an initial exchange against the refresh token.
// It's trusted until a request fails with 401, at which point the client
// falls back to WithRefreshToken to obtain a new one. Optional - if omitted,
// the client exchanges the refresh token for an access token on first use.
func WithAccessToken(accessToken string) Option {
	return func(c *Client) {
		c.accessToken = accessToken
	}
}

// WithTimeout sets the HTTP client timeout
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.httpClient.Timeout = timeout
	}
}

// WithRetries sets the maximum number of retries for failed requests
func WithRetries(retries int) Option {
	return func(c *Client) {
		c.maxRetries = retries
	}
}

// WithDebug enables debug logging
func WithDebug(enabled bool) Option {
	return func(c *Client) {
		c.debug = enabled
	}
}

// WithUserAgent sets a custom User-Agent header
func WithUserAgent(userAgent string) Option {
	return func(c *Client) {
		c.userAgent = userAgent
	}
}

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}
