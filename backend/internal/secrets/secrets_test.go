package secrets

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureGeneratesWhenAbsent(t *testing.T) {
	dir := t.TempDir()
	generated, err := Ensure(dir)
	if err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	if len(generated) != len(known) {
		t.Fatalf("generated = %v, want all %d secrets", generated, len(known))
	}
	for _, s := range known {
		b, err := os.ReadFile(filepath.Join(dir, s.Name))
		if err != nil {
			t.Fatalf("read %s: %v", s.Name, err)
		}
		raw, err := hex.DecodeString(string(b))
		if err != nil {
			t.Fatalf("%s is not hex: %v", s.Name, err)
		}
		if len(raw) != 32 {
			t.Fatalf("%s decoded to %d bytes, want 32", s.Name, len(raw))
		}
	}
}

func TestEnsureCreatesDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "secrets")
	if _, err := Ensure(dir); err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat dir: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o755 {
		t.Fatalf("dir mode = %o, want 755", perm)
	}
}

func TestEnsureEnvOverrideWinsWhenFileAbsent(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("POSTGRES_PASSWORD", "operator-value")
	generated, err := Ensure(dir)
	if err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "postgres_password"))
	if err != nil {
		t.Fatalf("read postgres_password: %v", err)
	}
	if string(b) != "operator-value" {
		t.Fatalf("postgres_password = %q, want %q", b, "operator-value")
	}
	if len(generated) != 1 || generated[0] != "jwt_secret" {
		t.Fatalf("generated = %v, want [jwt_secret]", generated)
	}
}

func TestEnsureKeepsExistingFile(t *testing.T) {
	dir := t.TempDir()
	if _, err := Ensure(dir); err != nil {
		t.Fatalf("first Ensure: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "jwt_secret"))
	if err != nil {
		t.Fatalf("read jwt_secret: %v", err)
	}
	before := string(b)

	t.Setenv("PECULIUM_JWT_SECRET", "operator-value")
	generated, err := Ensure(dir)
	if err != nil {
		t.Fatalf("second Ensure: %v", err)
	}
	if len(generated) != 0 {
		t.Fatalf("generated = %v, want none", generated)
	}
	after, err := os.ReadFile(filepath.Join(dir, "jwt_secret"))
	if err != nil {
		t.Fatalf("read jwt_secret: %v", err)
	}
	if string(after) != before {
		t.Fatal("existing file was modified")
	}
	if string(after) == "operator-value" {
		t.Fatal("env override must not replace an existing file")
	}
}

func TestEnsureTreatsEmptyFileAsAbsent(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "jwt_secret"), nil, 0o644); err != nil {
		t.Fatalf("write empty file: %v", err)
	}
	generated, err := Ensure(dir)
	if err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	found := false
	for _, name := range generated {
		if name == "jwt_secret" {
			found = true
		}
	}
	if !found {
		t.Fatalf("generated = %v, want jwt_secret in it", generated)
	}
	b, err := os.ReadFile(filepath.Join(dir, "jwt_secret"))
	if err != nil {
		t.Fatalf("read jwt_secret: %v", err)
	}
	if len(b) == 0 {
		t.Fatal("empty file was not regenerated")
	}
}

func TestEnsureFileMode(t *testing.T) {
	dir := t.TempDir()
	if _, err := Ensure(dir); err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	for _, s := range known {
		info, err := os.Stat(filepath.Join(dir, s.Name))
		if err != nil {
			t.Fatalf("stat %s: %v", s.Name, err)
		}
		if perm := info.Mode().Perm(); perm != 0o644 {
			t.Fatalf("%s mode = %o, want 644", s.Name, perm)
		}
	}
}

func TestHealthSuccess(t *testing.T) {
	dir := t.TempDir()
	if _, err := Ensure(dir); err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	if err := Health(dir); err != nil {
		t.Fatalf("Health: %v", err)
	}
}

func TestHealthFailure(t *testing.T) {
	if err := Health(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("Health on missing dir: want error, got nil")
	}

	dir := t.TempDir()
	if _, err := Ensure(dir); err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "jwt_secret"), nil, 0o644); err != nil {
		t.Fatalf("truncate jwt_secret: %v", err)
	}
	if err := Health(dir); err == nil {
		t.Fatal("Health with empty secret file: want error, got nil")
	}
}

func TestRandomHexValuesAreDistinct(t *testing.T) {
	dir1, dir2 := t.TempDir(), t.TempDir()
	if _, err := Ensure(dir1); err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	if _, err := Ensure(dir2); err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	for _, s := range known {
		a, err := os.ReadFile(filepath.Join(dir1, s.Name))
		if err != nil {
			t.Fatalf("read %s: %v", s.Name, err)
		}
		b, err := os.ReadFile(filepath.Join(dir2, s.Name))
		if err != nil {
			t.Fatalf("read %s: %v", s.Name, err)
		}
		if string(a) == string(b) {
			t.Fatalf("%s: two generations produced the same value", s.Name)
		}
	}
}
