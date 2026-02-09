package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	content := []byte(`tournaments:
  - tenant_id: "tenant-1"
    visibility: "PUBLIC"
    registration_status: "OPEN"
    status: "PENDING"
    min_available_places: 1
    blacklist:
      - "ladies"
      - "femenino"
  - tenant_id: "tenant-2"
    blacklist:
      - "women"
`)
	path := writeTempFile(t, content)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.Tournaments) != 2 {
		t.Fatalf("expected 2 tournament filters, got %d", len(cfg.Tournaments))
	}

	tf := cfg.Tournaments[0]
	if tf.TenantID != "tenant-1" {
		t.Errorf("expected tenant_id 'tenant-1', got %q", tf.TenantID)
	}
	if tf.Visibility != "PUBLIC" {
		t.Errorf("expected visibility 'PUBLIC', got %q", tf.Visibility)
	}
	if tf.RegistrationStatus != "OPEN" {
		t.Errorf("expected registration_status 'OPEN', got %q", tf.RegistrationStatus)
	}
	if tf.Status != "PENDING" {
		t.Errorf("expected status 'PENDING', got %q", tf.Status)
	}
	if tf.MinAvailablePlaces != 1 {
		t.Errorf("expected min_available_places 1, got %d", tf.MinAvailablePlaces)
	}
	if len(tf.Blacklist) != 2 {
		t.Fatalf("expected 2 blacklist entries, got %d", len(tf.Blacklist))
	}
	if tf.Blacklist[0] != "ladies" {
		t.Errorf("expected blacklist[0] 'ladies', got %q", tf.Blacklist[0])
	}
}

func TestLoad_MissingTenantID(t *testing.T) {
	content := []byte(`tournaments:
  - visibility: "PUBLIC"
`)
	path := writeTempFile(t, content)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
}

func TestLoad_NoTournaments(t *testing.T) {
	content := []byte(`tournaments: []
`)
	path := writeTempFile(t, content)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected validation error for empty tournaments, got nil")
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/config.yaml")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	content := []byte(`{{{invalid yaml`)
	path := writeTempFile(t, content)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}

func writeTempFile(t *testing.T, content []byte) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}
	return path
}
