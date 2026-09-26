package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSecretFilePrecedence(t *testing.T) {
	dir := t.TempDir()
	dbFile := filepath.Join(dir, "postgres_password")
	jwtFile := filepath.Join(dir, "jwt_secret")
	if err := os.WriteFile(dbFile, []byte("db-from-file\n\n"), 0o644); err != nil {
		t.Fatalf("write db file: %v", err)
	}
	if err := os.WriteFile(jwtFile, []byte("  jwt-from-file  "), 0o644); err != nil {
		t.Fatalf("write jwt file: %v", err)
	}

	// Explicit env value wins over the file.
	t.Setenv("PECULIUM_DB_PASSWORD", "db-from-env")
	t.Setenv("PECULIUM_DB_PASSWORD_FILE", dbFile)
	t.Setenv("PECULIUM_JWT_SECRET", "")
	t.Setenv("PECULIUM_JWT_SECRET_FILE", jwtFile)

	cfg := Load()
	if cfg.DBPassword != "db-from-env" {
		t.Fatalf("DBPassword = %q, want %q", cfg.DBPassword, "db-from-env")
	}
	if cfg.JWTSecret != "jwt-from-file" {
		t.Fatalf("JWTSecret = %q, want file content trimmed to %q", cfg.JWTSecret, "jwt-from-file")
	}

	// The file wins over the fallback: an unset env var must not mask *_FILE.
	t.Setenv("PECULIUM_DB_PASSWORD", "")
	cfg = Load()
	if cfg.DBPassword != "db-from-file" {
		t.Fatalf("DBPassword = %q, want %q (default masked the file)", cfg.DBPassword, "db-from-file")
	}

	// With neither env nor file, the default applies.
	t.Setenv("PECULIUM_DB_PASSWORD_FILE", "")
	t.Setenv("PECULIUM_JWT_SECRET_FILE", "")
	cfg = Load()
	if cfg.DBPassword != "peculium" {
		t.Fatalf("DBPassword = %q, want default %q", cfg.DBPassword, "peculium")
	}
	if cfg.JWTSecret != "change-me-in-production" {
		t.Fatalf("JWTSecret = %q, want default %q", cfg.JWTSecret, "change-me-in-production")
	}
}

func TestSecretFileUnreadableFallsBackToDefault(t *testing.T) {
	t.Setenv("PECULIUM_DB_PASSWORD", "")
	t.Setenv("PECULIUM_DB_PASSWORD_FILE", filepath.Join(t.TempDir(), "missing"))
	cfg := Load()
	if cfg.DBPassword != "peculium" {
		t.Fatalf("DBPassword = %q, want default %q", cfg.DBPassword, "peculium")
	}
}
