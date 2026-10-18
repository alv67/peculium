package price

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestJustETFFetcherRecordsHealth(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/etf/IE00B4L5Y983/exposure":
			io.WriteString(w, sampleETFETFExposureJSON)
		case "/api/v1/etf/search":
			io.WriteString(w, `[{"isin":"IE00B4L5Y983","name":"iShares Core S&P 500","ticker":"CSPX"}]`)
		default:
			time.Sleep(20 * time.Millisecond)
			http.Error(w, "no such isin", http.StatusNotFound)
		}
	}))
	defer ts.Close()

	rec := &recStub{}
	f := NewJustETFFetcher(ts.URL, WithETFHealthRecorder(rec))
	ctx := context.Background()
	assetID := uuid.New()

	if _, err := f.FetchExposure(ctx, assetID, "IE00B4L5Y983", "CSPX"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.SearchTicker(ctx, "spx"); err != nil {
		t.Fatal(err)
	}
	// Empty ticker: the label falls back to the bare ISIN.
	if _, err := f.FetchExposure(ctx, assetID, "ZZZZ", ""); err == nil {
		t.Fatal("expected a 404 failure")
	}

	if len(rec.events) != 3 {
		t.Fatalf("events = %d, want 3: %+v", len(rec.events), rec.events)
	}
	ok, search, fail := rec.events[0], rec.events[1], rec.events[2]
	if ok.EventType != "etf_exposure" || ok.Status != "success" || ok.Message != "CSPX (IE00B4L5Y983)" {
		t.Fatalf("success event = %+v", ok)
	}
	// The exposure event points back at the asset, so the activity list can
	// show its ticker; the search belongs to no asset and must not carry one.
	if ok.AssetID == nil || *ok.AssetID != assetID {
		t.Fatalf("success event asset id = %v, want %v", ok.AssetID, assetID)
	}
	if search.EventType != "etf_search" || search.Status != "success" || search.AssetID != nil {
		t.Fatalf("search event = %+v, want etf_search/success with no asset id", search)
	}
	if fail.EventType != "etf_exposure" || fail.Status != "failure" || fail.Code != "http_404" {
		t.Fatalf("failure event = %+v, want etf_exposure/failure/http_404", fail)
	}
	if fail.AssetID == nil || *fail.AssetID != assetID {
		t.Fatalf("failure event asset id = %v, want %v", fail.AssetID, assetID)
	}
	if fail.DurationMs < 15 {
		t.Fatalf("failure duration_ms = %d, want >= 15 (server slept 20ms)", fail.DurationMs)
	}
}

// TestJustETFFetcherMorningstarExposureCarriesAssetID covers the second
// exposure entry point: its event must point back at the asset too.
func TestJustETFFetcherMorningstarExposureCarriesAssetID(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, sampleETFETFExposureJSON)
	}))
	defer ts.Close()

	rec := &recStub{}
	f := NewJustETFFetcher(ts.URL, WithETFHealthRecorder(rec))
	assetID := uuid.New()

	if _, err := f.FetchMorningstarExposure(context.Background(), assetID, "IE00B4L5Y983", "CSPX"); err != nil {
		t.Fatal(err)
	}
	if len(rec.events) != 1 {
		t.Fatalf("events = %d, want 1: %+v", len(rec.events), rec.events)
	}
	ev := rec.events[0]
	if ev.EventType != "etf_morningstar_exposure" || ev.Status != "success" {
		t.Fatalf("event = %+v, want etf_morningstar_exposure/success", ev)
	}
	if ev.AssetID == nil || *ev.AssetID != assetID {
		t.Fatalf("event asset id = %v, want %v", ev.AssetID, assetID)
	}
}

func TestJustETFRecorderOptional(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, sampleETFETFExposureJSON)
	}))
	defer ts.Close()

	// No WithETFHealthRecorder: calls work and nothing panics.
	f := NewJustETFFetcher(ts.URL)
	if _, err := f.FetchExposure(context.Background(), uuid.Nil, "IE00B4L5Y983", "CSPX"); err != nil {
		t.Fatal(err)
	}
}
