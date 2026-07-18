// Package client provides a Go client for accessing the Playtomic API.
package client

import (
	"net/http"
	"sync"
	"time"
)

const (
	// DefaultBaseUrlV1 is the default Playtomic API endpoint
	DefaultBaseUrlV1 = "https://api.app.playtomic.io/v1"
	DefaultBaseUrlV2 = "https://api.app.playtomic.io/v2"

	// DefaultAuthBaseURL is the default Playtomic auth endpoint, used to
	// exchange a refresh token for an access token.
	DefaultAuthBaseURL = "https://api.app.playtomic.io"

	// DefaultTimeout is the default client timeout
	DefaultTimeout = 30 * time.Second

	// DefaultMaxRetries is the default number of request retries
	DefaultMaxRetries = 3

	// DefaultUserAgent is the default User-Agent sent with requests
	DefaultUserAgent = "PlaytomicGoClient/1.0"
)

// Client provides access to the Playtomic API
type Client struct {
	httpClient *http.Client
	baseURL    string
	authURL    string
	userAgent  string
	maxRetries int
	debug      bool

	refreshToken string

	tokenMu               sync.Mutex
	accessToken           string
	accessTokenExpiration time.Time
}

// NewClient creates a new Playtomic API client with the given options
func NewClient(opts ...Option) *Client {
	c := &Client{
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
		baseURL:    DefaultBaseUrlV1,
		authURL:    DefaultAuthBaseURL,
		userAgent:  DefaultUserAgent,
		maxRetries: DefaultMaxRetries,
	}

	// Apply options
	for _, opt := range opts {
		opt(c)
	}

	return c
}
