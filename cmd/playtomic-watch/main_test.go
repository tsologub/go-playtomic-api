package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rafa-garcia/go-playtomic-api/client"
	"github.com/rafa-garcia/go-playtomic-api/models"
)

func TestExportRotatedAccessTokenWritesWhenChanged(t *testing.T) {
	envFile := filepath.Join(t.TempDir(), "github_env")
	if err := os.WriteFile(envFile, nil, 0o644); err != nil {
		t.Fatalf("creating env file: %v", err)
	}
	t.Setenv("GITHUB_ENV", envFile)

	exportRotatedAccessToken("stale-access-token", "refreshed-access-token")

	data, err := os.ReadFile(envFile)
	if err != nil {
		t.Fatalf("reading env file: %v", err)
	}
	if !strings.Contains(string(data), "ROTATED_ACCESS_TOKEN=refreshed-access-token\n") {
		t.Errorf("expected env file to contain the refreshed token, got %q", data)
	}
}

func TestExportRotatedAccessTokenNoopWhenUnchanged(t *testing.T) {
	envFile := filepath.Join(t.TempDir(), "github_env")
	if err := os.WriteFile(envFile, nil, 0o644); err != nil {
		t.Fatalf("creating env file: %v", err)
	}
	t.Setenv("GITHUB_ENV", envFile)

	exportRotatedAccessToken("same-access-token", "same-access-token")

	data, err := os.ReadFile(envFile)
	if err != nil {
		t.Fatalf("reading env file: %v", err)
	}
	if len(data) != 0 {
		t.Errorf("expected no write when the access token is unchanged, got %q", data)
	}
}

func TestExportRotatedAccessTokenNoopOutsideActions(t *testing.T) {
	t.Setenv("GITHUB_ENV", "")
	// Must not panic or try to open an empty path.
	exportRotatedAccessToken("stale-access-token", "refreshed-access-token")
}

// TestRefreshedAccessTokenExportEndToEnd exercises the real path this
// feature depends on: a client is seeded with a stale access token, a
// request fails with 401 and falls back to the refresh token, and
// exportRotatedAccessToken picks up the resulting access token and persists
// it to GITHUB_ENV.
func TestRefreshedAccessTokenExportEndToEnd(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v3/auth/token", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]string{
			"access_token":            "refreshed-access-token",
			"access_token_expiration": time.Now().Add(time.Hour).UTC().Format("2006-01-02T15:04:05"),
		}
		json.NewEncoder(w).Encode(resp)
	})
	mux.HandleFunc("/classes", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "Bearer stale-access-token" {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"status": "UNAUTHORIZED"})
			return
		}
		json.NewEncoder(w).Encode([]models.Class{})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	c := client.NewClient(
		client.WithBaseURL(server.URL),
		client.WithAuthBaseURL(server.URL),
		client.WithRefreshToken("refresh-token"),
		client.WithAccessToken("stale-access-token"),
	)

	if _, err := c.GetClasses(context.Background(), &models.SearchClassesParams{TenantIDs: []string{"t"}}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	envFile := filepath.Join(t.TempDir(), "github_env")
	if err := os.WriteFile(envFile, nil, 0o644); err != nil {
		t.Fatalf("creating env file: %v", err)
	}
	t.Setenv("GITHUB_ENV", envFile)

	exportRotatedAccessToken("stale-access-token", c.AccessToken())

	data, err := os.ReadFile(envFile)
	if err != nil {
		t.Fatalf("reading env file: %v", err)
	}
	if !strings.Contains(string(data), "ROTATED_ACCESS_TOKEN=refreshed-access-token\n") {
		t.Errorf("expected the refreshed access token to be exported, got %q", data)
	}
}
