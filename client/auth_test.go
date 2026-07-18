package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestAccessTokenForFetchesAndCaches(t *testing.T) {
	var tokenCalls int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v3/auth/token" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		atomic.AddInt32(&tokenCalls, 1)

		resp := tokenResponse{
			AccessToken:           "access-1",
			AccessTokenExpiration: time.Now().Add(time.Hour).UTC().Format(tokenExpirationLayout),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := NewClient(WithAuthBaseURL(server.URL), WithRefreshToken("initial-refresh-token"))

	token, err := c.accessTokenFor(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if token != "access-1" {
		t.Errorf("expected access-1, got %s", token)
	}

	// Second call should reuse the cached token, not hit the server again.
	if _, err := c.accessTokenFor(context.Background()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if calls := atomic.LoadInt32(&tokenCalls); calls != 1 {
		t.Errorf("expected 1 token request, got %d", calls)
	}
}

func TestAccessTokenForAdoptsRotatedRefreshToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body tokenRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decoding token request: %v", err)
		}
		if body.RefreshToken != "original-refresh-token" {
			t.Errorf("expected first exchange to use original-refresh-token, got %s", body.RefreshToken)
		}

		resp := tokenResponse{
			AccessToken:           "access-1",
			AccessTokenExpiration: time.Now().Add(time.Hour).UTC().Format(tokenExpirationLayout),
			RefreshToken:          "rotated-refresh-token",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := NewClient(WithAuthBaseURL(server.URL), WithRefreshToken("original-refresh-token"))

	if _, err := c.accessTokenFor(context.Background()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// The API rotates and invalidates the refresh token on every exchange -
	// the client must adopt the new one for any future exchange, and expose
	// it via RefreshToken() so the caller can persist it.
	if c.RefreshToken() != "rotated-refresh-token" {
		t.Errorf("expected RefreshToken() to reflect the rotated token, got %s", c.RefreshToken())
	}
	if c.refreshToken != "rotated-refresh-token" {
		t.Errorf("expected internal refreshToken to be updated for future exchanges, got %s", c.refreshToken)
	}
}

func TestAccessTokenForUsesSeededTokenWithoutExchange(t *testing.T) {
	var tokenCalls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&tokenCalls, 1)
		t.Errorf("did not expect a call to %s when a seeded access token is present", r.URL.Path)
	}))
	defer server.Close()

	c := NewClient(
		WithAuthBaseURL(server.URL),
		WithRefreshToken("refresh-token"),
		WithAccessToken("seeded-access-token"),
	)

	token, err := c.accessTokenFor(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if token != "seeded-access-token" {
		t.Errorf("expected the seeded token to be used as-is, got %s", token)
	}
	if calls := atomic.LoadInt32(&tokenCalls); calls != 0 {
		t.Errorf("expected no auth/token calls, got %d", calls)
	}
	if c.AccessToken() != "seeded-access-token" {
		t.Errorf("expected AccessToken() to return the seeded token, got %s", c.AccessToken())
	}
}

func TestSendRequestFallsBackToRefreshWhenSeededAccessTokenFails(t *testing.T) {
	var tokenCalls, dataCalls int32

	mux := http.NewServeMux()
	mux.HandleFunc("/v3/auth/token", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&tokenCalls, 1)
		resp := tokenResponse{
			AccessToken:           "refreshed-access-token",
			AccessTokenExpiration: time.Now().Add(time.Hour).UTC().Format(tokenExpirationLayout),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	mux.HandleFunc("/classes", func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&dataCalls, 1)
		if n == 1 {
			auth := r.Header.Get("Authorization")
			if auth != "Bearer stale-seeded-token" {
				t.Errorf("expected first attempt to use the seeded token, got %q", auth)
			}
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"status": "UNAUTHORIZED"})
			return
		}
		auth := r.Header.Get("Authorization")
		if auth != "Bearer refreshed-access-token" {
			t.Errorf("expected retry to use the refreshed token, got %q", auth)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]string{})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	c := NewClient(
		WithBaseURL(server.URL),
		WithAuthBaseURL(server.URL),
		WithRefreshToken("refresh-token"),
		WithAccessToken("stale-seeded-token"),
	)

	var result []map[string]string
	err := c.sendRequest(context.Background(), http.MethodGet, "/classes", "", nil, &result)
	if err != nil {
		t.Fatalf("expected the retry to succeed, got %v", err)
	}
	if calls := atomic.LoadInt32(&tokenCalls); calls != 1 {
		t.Errorf("expected exactly 1 auth/token call, got %d", calls)
	}
	if c.AccessToken() != "refreshed-access-token" {
		t.Errorf("expected AccessToken() to reflect the refreshed token, got %s", c.AccessToken())
	}
}

func TestAccessTokenForNoRefreshToken(t *testing.T) {
	c := NewClient()
	if _, err := c.accessTokenFor(context.Background()); err == nil {
		t.Fatal("expected error when no refresh token is configured")
	}
}

func TestAccessTokenForInvalidRefreshToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"status":            "USER_NOT_FOUND",
			"localized_message": "Invalid code.",
		})
	}))
	defer server.Close()

	c := NewClient(WithAuthBaseURL(server.URL), WithRefreshToken("bad-token"))

	_, err := c.accessTokenFor(context.Background())
	if err == nil {
		t.Fatal("expected error for invalid refresh token")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", apiErr.StatusCode)
	}
	if apiErr.Message != "Invalid code." {
		t.Errorf("expected localized_message as message, got %q", apiErr.Message)
	}
}

func TestSendRequestRetriesOnceOn401(t *testing.T) {
	var tokenCalls, dataCalls int32

	mux := http.NewServeMux()
	mux.HandleFunc("/v3/auth/token", func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&tokenCalls, 1)
		resp := tokenResponse{
			AccessToken:           "access-token",
			AccessTokenExpiration: time.Now().Add(time.Hour).UTC().Format(tokenExpirationLayout),
		}
		_ = n
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	mux.HandleFunc("/classes", func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&dataCalls, 1)
		if n == 1 {
			// First call: simulate a stale/rejected token.
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"status": "UNAUTHORIZED"})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]string{})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	c := newTestClient(server)

	var result []map[string]string
	err := c.sendRequest(context.Background(), http.MethodGet, "/classes", "", nil, &result)
	if err != nil {
		t.Fatalf("expected the second attempt to succeed, got %v", err)
	}
	if calls := atomic.LoadInt32(&dataCalls); calls != 2 {
		t.Errorf("expected 2 data requests (1 retry), got %d", calls)
	}
	if calls := atomic.LoadInt32(&tokenCalls); calls != 2 {
		t.Errorf("expected token to be fetched twice (initial + after invalidation), got %d", calls)
	}
}

func TestSendRequestGivesUpAfterSecond401(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v3/auth/token", func(w http.ResponseWriter, r *http.Request) {
		resp := tokenResponse{
			AccessToken:           "access-token",
			AccessTokenExpiration: time.Now().Add(time.Hour).UTC().Format(tokenExpirationLayout),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	mux.HandleFunc("/classes", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"status": "UNAUTHORIZED"})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	c := newTestClient(server)

	var result []map[string]string
	err := c.sendRequest(context.Background(), http.MethodGet, "/classes", "", nil, &result)
	if err == nil {
		t.Fatal("expected an error after a persistent 401")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", apiErr.StatusCode)
	}
}
