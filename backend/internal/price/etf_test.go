package price

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/alv67/peculium/internal/model"
)

const sampleETFETFExposureJSON = `{"countries":[{"name":"United States","weight":63.34}],"sectors":[{"name":"Information Technology","weight":27.5}]}`

func TestJustETFFetcherFetchExposure(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/etf/IE00B4L5Y983/exposure" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, sampleETFETFExposureJSON)
	}))
	defer ts.Close()

	f := NewJustETFFetcher(ts.URL)
	exposure, err := f.FetchExposure(context.Background(), "IE00B4L5Y983")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(exposure.Countries) != 1 || exposure.Countries[0].Name != "United States" ||
		!exposure.Countries[0].Weight.Equal(decimal.NewFromFloat(63.34)) {
		t.Fatalf("unexpected countries: %+v", exposure.Countries)
	}
	if len(exposure.Sectors) != 1 || exposure.Sectors[0].Name != "Information Technology" ||
		!exposure.Sectors[0].Weight.Equal(decimal.NewFromFloat(27.5)) {
		t.Fatalf("unexpected sectors: %+v", exposure.Sectors)
	}
}

func TestJustETFFetcherFetchMorningstarExposure(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/etf/IE00B4L5Y983/morningstar-exposure" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, sampleETFETFExposureJSON)
	}))
	defer ts.Close()

	f := NewJustETFFetcher(ts.URL)
	exposure, err := f.FetchMorningstarExposure(context.Background(), "IE00B4L5Y983")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(exposure.Countries) != 1 || exposure.Countries[0].Name != "United States" ||
		!exposure.Countries[0].Weight.Equal(decimal.NewFromFloat(63.34)) {
		t.Fatalf("unexpected countries: %+v", exposure.Countries)
	}
	if len(exposure.Sectors) != 1 || exposure.Sectors[0].Name != "Information Technology" ||
		!exposure.Sectors[0].Weight.Equal(decimal.NewFromFloat(27.5)) {
		t.Fatalf("unexpected sectors: %+v", exposure.Sectors)
	}
}

const sampleMorningstarWithRegionsJSON = `{"countries":[{"name":"United States","weight":60}],"regions":[{"name":"North America","weight":60},{"name":"United Kingdom","weight":25},{"name":"Japan","weight":15}],"sectors":[{"name":"Information Technology","weight":27.5}]}`

// TestJustETFFetcherFetchMorningstarExposureRegions verifies that the official
// regions field returned by the python-service is parsed through into the
// exposure model.
func TestJustETFFetcherFetchMorningstarExposureRegions(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/etf/IE00B4L5Y983/morningstar-exposure" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, sampleMorningstarWithRegionsJSON)
	}))
	defer ts.Close()

	f := NewJustETFFetcher(ts.URL)
	exposure, err := f.FetchMorningstarExposure(context.Background(), "IE00B4L5Y983")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []model.ExposureRow{
		{Name: "North America", Weight: decimal.NewFromInt(60)},
		{Name: "United Kingdom", Weight: decimal.NewFromInt(25)},
		{Name: "Japan", Weight: decimal.NewFromInt(15)},
	}
	if len(exposure.Regions) != len(want) {
		t.Fatalf("regions len = %d, want %d: %+v", len(exposure.Regions), len(want), exposure.Regions)
	}
	for i, row := range want {
		if exposure.Regions[i].Name != row.Name || !exposure.Regions[i].Weight.Equal(row.Weight) {
			t.Fatalf("region[%d] = %+v, want %+v", i, exposure.Regions[i], row)
		}
	}
}

func TestJustETFFetcherSearchTicker(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/etf/search" || r.URL.Query().Get("q") != "EUNL" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `[{"isin":"IE00B4L5Y983","name":"iShares Core MSCI World UCITS ETF USD (Acc)"}]`)
	}))
	defer ts.Close()

	f := NewJustETFFetcher(ts.URL)
	results, err := f.SearchTicker(context.Background(), "EUNL")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 || results[0].ISIN != "IE00B4L5Y983" {
		t.Fatalf("unexpected results: %+v", results)
	}
}

func TestBestMatchPrefersNameOverlap(t *testing.T) {
	results := []EtfSearchResult{
		{ISIN: "IE00BKM4GZ66", Name: "iShares Core MSCI Emerging Markets IMI UCITS ETF (Acc)"},
		{ISIN: "IE00B4K48X80", Name: "iShares Core MSCI Europe UCITS ETF EUR (Acc)"},
	}
	best, ok := BestMatch(results, "SMEA.MI", "iShares Core MSCI Europe UCITS ETF EUR (Acc)")
	if !ok || best.ISIN != "IE00B4K48X80" {
		t.Fatalf("expected Europe ETF, got %+v (ok=%v)", best, ok)
	}
}

func TestBestMatchPrefersExactTicker(t *testing.T) {
	results := []EtfSearchResult{
		{ISIN: "IE00BKM4GZ66", Ticker: "EIMI", Name: "iShares Core MSCI Emerging Markets IMI UCITS ETF (Acc)"},
		{ISIN: "IE00B4K48X80", Ticker: "SMEA", Name: "iShares Core MSCI Europe UCITS ETF EUR (Acc)"},
	}
	best, ok := BestMatch(results, "SMEA", "")
	if !ok || best.ISIN != "IE00B4K48X80" {
		t.Fatalf("expected exact ticker match, got %+v (ok=%v)", best, ok)
	}
}

func TestBestMatchEmpty(t *testing.T) {
	if _, ok := BestMatch(nil, "X", "name"); ok {
		t.Fatal("expected no match for empty results")
	}
	if _, ok := BestMatch([]EtfSearchResult{{Ticker: "X"}}, "X", "name"); ok {
		t.Fatal("expected no match when results carry no ISIN")
	}
}

func TestJustETFFetcherStatusError(t *testing.T) {
	var calls atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusBadGateway)
		io.WriteString(w, `{"detail":"upstream fetch failed: timeout"}`)
	}))
	defer ts.Close()

	f := NewJustETFFetcher(ts.URL)
	_, err := f.FetchExposure(context.Background(), "IE00B4L5Y983")
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "502") || !strings.Contains(err.Error(), "upstream fetch failed") {
		t.Fatalf("error should include status and detail, got: %v", err)
	}
	// Status errors are not retried.
	if calls.Load() != 1 {
		t.Fatalf("expected 1 attempt, got %d", calls.Load())
	}
}

func TestJustETFFetcherRetriesTransientError(t *testing.T) {
	var calls atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) <= 2 {
			// Force a connection failure to simulate a transient network error.
			conn, _, err := w.(http.Hijacker).Hijack()
			if err != nil {
				return
			}
			conn.Close()
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"countries": []map[string]any{{"name": "United States", "weight": 63.34}},
			"sectors":   []map[string]any{{"name": "Information Technology", "weight": 27.5}},
		})
	}))
	defer ts.Close()

	f := NewJustETFFetcher(ts.URL)
	exposure, err := f.FetchExposure(context.Background(), "IE00B4L5Y983")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(exposure.Countries) != 1 || exposure.Countries[0].Name != "United States" {
		t.Fatalf("unexpected countries: %+v", exposure.Countries)
	}
	if calls.Load() != 3 {
		t.Fatalf("expected 3 attempts (1 + 2 retries), got %d", calls.Load())
	}
}

func TestAggregateRegions(t *testing.T) {
	tests := []struct {
		name string
		in   []model.ExposureRow
		want []model.ExposureRow
	}{
		{
			name: "countries in the same region are summed",
			in: []model.ExposureRow{
				{Name: "United States", Weight: decimal.NewFromFloat(30)},
				{Name: "USA", Weight: decimal.NewFromFloat(33.34)},
				{Name: "Japan", Weight: decimal.NewFromFloat(36.66)},
			},
			want: []model.ExposureRow{
				{Name: "Japan", Weight: decimal.NewFromFloat(36.66)},
				{Name: "North America", Weight: decimal.NewFromFloat(63.34)},
			},
		},
		{
			name: "ISO codes and full names map to the same region",
			in: []model.ExposureRow{
				{Name: "DE", Weight: decimal.NewFromInt(40)},
				{Name: "France", Weight: decimal.NewFromInt(35)},
				{Name: "GB", Weight: decimal.NewFromInt(25)},
			},
			want: []model.ExposureRow{
				{Name: "Europe Developed", Weight: decimal.NewFromInt(75)},
				{Name: "United Kingdom", Weight: decimal.NewFromInt(25)},
			},
		},
		{
			name: "unrecognized country lands in Other / Not Classified",
			in: []model.ExposureRow{
				{Name: "Atlantis", Weight: decimal.NewFromInt(100)},
			},
			want: []model.ExposureRow{
				{Name: "Other / Not Classified", Weight: decimal.NewFromInt(100)},
			},
		},
		{
			name: "empty and non-positive rows are dropped",
			in: []model.ExposureRow{
				{Name: "", Weight: decimal.NewFromInt(50)},
				{Name: "United States", Weight: decimal.NewFromFloat(0)},
				{Name: "Japan", Weight: decimal.NewFromInt(100)},
			},
			want: []model.ExposureRow{
				{Name: "Japan", Weight: decimal.NewFromInt(100)},
			},
		},
		{
			name: "residual below 100 lands in Other / Not Classified",
			in: []model.ExposureRow{
				{Name: "United States", Weight: decimal.NewFromFloat(60)},
				{Name: "Japan", Weight: decimal.NewFromFloat(35.22)},
			},
			want: []model.ExposureRow{
				{Name: "Japan", Weight: decimal.NewFromFloat(35.22)},
				{Name: "North America", Weight: decimal.NewFromFloat(60)},
				{Name: "Other / Not Classified", Weight: decimal.NewFromFloat(4.78)},
			},
		},
		{
			name: "no residual bucket when regions already sum to 100",
			in: []model.ExposureRow{
				{Name: "United States", Weight: decimal.NewFromFloat(63.34)},
				{Name: "Japan", Weight: decimal.NewFromFloat(36.66)},
			},
			want: []model.ExposureRow{
				{Name: "Japan", Weight: decimal.NewFromFloat(36.66)},
				{Name: "North America", Weight: decimal.NewFromFloat(63.34)},
			},
		},
		{
			name: "Korea and Taiwan map to Asia Developed",
			in: []model.ExposureRow{
				{Name: "KR", Weight: decimal.NewFromInt(40)},
				{Name: "TW", Weight: decimal.NewFromInt(30)},
				{Name: "HK", Weight: decimal.NewFromInt(30)},
			},
			want: []model.ExposureRow{
				{Name: "Asia Developed", Weight: decimal.NewFromInt(100)},
			},
		},
		{
			name: "Australia and New Zealand map to Australasia",
			in: []model.ExposureRow{
				{Name: "AU", Weight: decimal.NewFromInt(70)},
				{Name: "NZ", Weight: decimal.NewFromInt(30)},
			},
			want: []model.ExposureRow{
				{Name: "Australasia", Weight: decimal.NewFromInt(100)},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AggregateRegions(tt.in)
			if !exposureRowsEqual(got, tt.want) {
				t.Fatalf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestAggregateSectors(t *testing.T) {
	tests := []struct {
		name string
		in   []model.ExposureRow
		want []model.ExposureRow
	}{
		{
			name: "provider aliases normalize to GICS",
			in: []model.ExposureRow{
				{Name: "Technology", Weight: decimal.NewFromInt(40)},
				{Name: "Financial Services", Weight: decimal.NewFromInt(30)},
				{Name: "Healthcare", Weight: decimal.NewFromInt(30)},
			},
			want: []model.ExposureRow{
				{Name: "Financials", Weight: decimal.NewFromInt(30)},
				{Name: "Health Care", Weight: decimal.NewFromInt(30)},
				{Name: "Information Technology", Weight: decimal.NewFromInt(40)},
			},
		},
		{
			name: "aliases of the same sector are summed",
			in: []model.ExposureRow{
				{Name: "Technology", Weight: decimal.NewFromInt(20)},
				{Name: "Information Technology", Weight: decimal.NewFromInt(50)},
				{Name: "Consumer Cyclical", Weight: decimal.NewFromInt(30)},
			},
			want: []model.ExposureRow{
				{Name: "Consumer Discretionary", Weight: decimal.NewFromInt(30)},
				{Name: "Information Technology", Weight: decimal.NewFromInt(70)},
			},
		},
		{
			name: "canonical names are kept",
			in: []model.ExposureRow{
				{Name: "Energy", Weight: decimal.NewFromInt(100)},
			},
			want: []model.ExposureRow{
				{Name: "Energy", Weight: decimal.NewFromInt(100)},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AggregateSectors(tt.in)
			if !exposureRowsEqual(got, tt.want) {
				t.Fatalf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func exposureRowsEqual(a, b []model.ExposureRow) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Name != b[i].Name || !a[i].Weight.Equal(b[i].Weight) {
			return false
		}
	}
	return true
}
