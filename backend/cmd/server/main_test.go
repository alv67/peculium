package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alv67/peculium/internal/config"
)

func TestVersionHandler(t *testing.T) {
	cfg := &config.Config{Version: "1.2.3", Commit: "abcdef", BuiltAt: "2026-10-04T12:00:00Z"}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/version", nil)
	rr := httptest.NewRecorder()
	versionHandler(cfg).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	var got map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body %q: %v", rr.Body, err)
	}
	want := map[string]string{"version": "1.2.3", "commit": "abcdef", "built_at": "2026-10-04T12:00:00Z"}
	if len(got) != len(want) {
		t.Fatalf("fields = %v, want exactly version/commit/built_at", got)
	}
	for k, v := range want {
		if got[k] != v {
			t.Fatalf("field %q = %q, want %q", k, got[k], v)
		}
	}
}

func TestVersionDefaults(t *testing.T) {
	t.Setenv("PECULIUM_VERSION", "")
	cfg := config.Load()
	if cfg.Version != "dev" {
		t.Fatalf("default Version = %q, want %q", cfg.Version, "dev")
	}
	if cfg.Commit != "" || cfg.BuiltAt != "" {
		t.Fatalf("default Commit/BuiltAt = %q/%q, want empty", cfg.Commit, cfg.BuiltAt)
	}
}
