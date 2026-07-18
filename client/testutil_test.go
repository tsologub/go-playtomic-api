package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// newAuthTestServer builds a test server that serves POST /v3/auth/token
// with a valid, far-future token and delegates every other path to handler.
// It also returns the number of times the token endpoint was hit, via the
// returned *int pointer (safe to read only after the test's requests finish).
func newAuthTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/v3/auth/token", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST to /v3/auth/token, got %s", r.Method)
		}

		var body tokenRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decoding token request: %v", err)
		}
		if body.RefreshToken == "" {
			t.Errorf("Expected non-empty refresh_token in token request")
		}

		resp := tokenResponse{
			AccessToken:           "test-access-token",
			AccessTokenExpiration: time.Now().Add(time.Hour).UTC().Format(tokenExpirationLayout),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	})
	mux.Handle("/", handler)

	return httptest.NewServer(mux)
}

// newTestClient creates a Client wired to server for both data and auth
// requests, with a placeholder refresh token.
func newTestClient(server *httptest.Server, opts ...Option) *Client {
	base := []Option{
		WithBaseURL(server.URL),
		WithAuthBaseURL(server.URL),
		WithRefreshToken("test-refresh-token"),
	}
	return NewClient(append(base, opts...)...)
}
