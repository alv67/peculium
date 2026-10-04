package price

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
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

	if _, err := f.FetchExposure(ctx, "IE00B4L5Y983"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.SearchTicker(ctx, "spx"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.FetchExposure(ctx, "ZZZZ"); err == nil {
		t.Fatal("expected a 404 failure")
	}

	if len(rec.events) != 3 {
		t.Fatalf("events = %d, want 3: %+v", len(rec.events), rec.events)
	}
	ok, search, fail := rec.events[0], rec.events[1], rec.events[2]
	if ok.EventType != "etf_exposure" || ok.Status != "success" || ok.Message != "IE00B4L5Y983" {
		t.Fatalf("success event = %+v", ok)
	}
	if search.EventType != "etf_search" || search.Status != "success" {
		t.Fatalf("search event = %+v", search)
	}
	if fail.EventType != "etf_exposure" || fail.Status != "failure" || fail.Code != "http_404" {
		t.Fatalf("failure event = %+v, want etf_exposure/failure/http_404", fail)
	}
	if fail.DurationMs < 15 {
		t.Fatalf("failure duration_ms = %d, want >= 15 (server slept 20ms)", fail.DurationMs)
	}
}

func TestJustETFRecorderOptional(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, sampleETFETFExposureJSON)
	}))
	defer ts.Close()

	// No WithETFHealthRecorder: calls work and nothing panics.
	f := NewJustETFFetcher(ts.URL)
	if _, err := f.FetchExposure(context.Background(), "IE00B4L5Y983"); err != nil {
		t.Fatal(err)
	}
}
