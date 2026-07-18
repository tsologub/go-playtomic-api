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

func TestExportRotatedRefreshTokenWritesWhenChanged(t *testing.T) {
	envFile := filepath.Join(t.TempDir(), "github_env")
	if err := os.WriteFile(envFile, nil, 0o644); err != nil {
		t.Fatalf("creating env file: %v", err)
	}
	t.Setenv("GITHUB_ENV", envFile)

	exportRotatedRefreshToken("original-token", "rotated-token")

	data, err := os.ReadFile(envFile)
	if err != nil {
		t.Fatalf("reading env file: %v", err)
	}
	if !strings.Contains(string(data), "ROTATED_REFRESH_TOKEN=rotated-token\n") {
		t.Errorf("expected env file to contain rotated token, got %q", data)
	}
}

func TestExportRotatedRefreshTokenNoopWhenUnchanged(t *testing.T) {
	envFile := filepath.Join(t.TempDir(), "github_env")
	if err := os.WriteFile(envFile, nil, 0o644); err != nil {
		t.Fatalf("creating env file: %v", err)
	}
	t.Setenv("GITHUB_ENV", envFile)

	exportRotatedRefreshToken("same-token", "same-token")

	data, err := os.ReadFile(envFile)
	if err != nil {
		t.Fatalf("reading env file: %v", err)
	}
	if len(data) != 0 {
		t.Errorf("expected no write when token is unchanged, got %q", data)
	}
}

func TestExportRotatedRefreshTokenNoopOutsideActions(t *testing.T) {
	t.Setenv("GITHUB_ENV", "")
	// Must not panic or try to open an empty path.
	exportRotatedRefreshToken("original-token", "rotated-token")
}

// TestRotatedTokenExportEndToEnd exercises the real path this feature
// depends on: a client performs a request, which rotates its refresh token
// via the mocked /v3/auth/token exchange, and exportRotatedRefreshToken
// picks that rotated value up and persists it to GITHUB_ENV.
func TestRotatedTokenExportEndToEnd(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v3/auth/token", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]string{
			"access_token":            "access-token",
			"access_token_expiration": time.Now().Add(time.Hour).UTC().Format("2006-01-02T15:04:05"),
			"refresh_token":           "rotated-refresh-token",
		}
		json.NewEncoder(w).Encode(resp)
	})
	mux.HandleFunc("/classes", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]models.Class{})
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	c := client.NewClient(
		client.WithBaseURL(server.URL),
		client.WithAuthBaseURL(server.URL),
		client.WithRefreshToken("original-token"),
	)

	if _, err := c.GetClasses(context.Background(), &models.SearchClassesParams{TenantIDs: []string{"t"}}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	envFile := filepath.Join(t.TempDir(), "github_env")
	if err := os.WriteFile(envFile, nil, 0o644); err != nil {
		t.Fatalf("creating env file: %v", err)
	}
	t.Setenv("GITHUB_ENV", envFile)

	exportRotatedRefreshToken("original-token", c.RefreshToken())

	data, err := os.ReadFile(envFile)
	if err != nil {
		t.Fatalf("reading env file: %v", err)
	}
	if !strings.Contains(string(data), "ROTATED_REFRESH_TOKEN=rotated-refresh-token\n") {
		t.Errorf("expected rotated token to be exported, got %q", data)
	}
}
