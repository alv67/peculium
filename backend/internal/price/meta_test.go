package price

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// Stock chart meta on the NASDAQ (exchangeCountry fallback would give US).
const metaChartStock = `{"chart":{"result":[{"meta":{"symbol":"AAPL","currency":"USD","exchangeName":"NMS","instrumentType":"EQUITY","longName":"Apple Inc."}}]}}`

const metaChartEtf = `{"chart":{"result":[{"meta":{"symbol":"VWCE","currency":"USD","exchangeName":"GMT","instrumentType":"ETF","longName":"Vanguard FTSE All-World UCITS ETF"}}]}}`

const metaProfileBody = `{"quoteSummary":{"result":[{"assetProfile":{"sector":"Technology","industry":"Consumer Electronics","country":"United States"}}]}}`

const metaFundBody = `{"quoteSummary":{"result":[{"defaultKeyStatistics":{"category":"Equity Global"}}]}}`

// metaTransport routes the Yahoo endpoints FetchMeta touches (chart, crumb
// handshake, quoteSummary) and records each quoteSummary call so tests can
// assert how many requests carry the assetProfile module.
type metaTransport struct {
	chartBody   string
	profileBody string
	profileCode int
	profileHits int
	modules     []string
}

func (m *metaTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	switch {
	case req.URL.Host == "fc.yahoo.com":
		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       http.NoBody,
			Header:     http.Header{"Set-Cookie": []string{"A1=dummy"}},
		}, nil
	case strings.HasSuffix(req.URL.Path, "/getcrumb"):
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("dummy-crumb")),
			Header:     http.Header{},
		}, nil
	case strings.Contains(req.URL.Path, "quoteSummary"):
		m.profileHits++
		m.modules = append(m.modules, req.URL.Query().Get("modules"))
		return &http.Response{
			StatusCode: m.profileCode,
			Body:       io.NopCloser(strings.NewReader(m.profileBody)),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		}, nil
	case strings.Contains(req.URL.Path, "/v8/finance/chart/"):
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(m.chartBody)),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		}, nil
	}
	return nil, http.ErrNoLocation
}

func newMetaFetcher(chartBody, profileBody string, profileCode int) (*YahooFetcher, *metaTransport) {
	tr := &metaTransport{chartBody: chartBody, profileBody: profileBody, profileCode: profileCode}
	f := NewYahooFetcher(nil, time.Hour, WithMinInterval(0))
	f.client = &http.Client{Transport: tr}
	return f, tr
}

// TestFetchMetaStockCarriesProfileSectorIndustry: the same assetProfile call
// that resolves the issuer country must also carry sector and industry, and
// the profile module must be requested exactly once (no duplicate request).
func TestFetchMetaStockCarriesProfileSectorIndustry(t *testing.T) {
	f, tr := newMetaFetcher(metaChartStock, metaProfileBody, http.StatusOK)

	meta, err := f.FetchMeta(context.Background(), "AAPL")
	if err != nil {
		t.Fatalf("FetchMeta: %v", err)
	}
	if meta.Sector != "Technology" || meta.Industry != "Consumer Electronics" {
		t.Fatalf("sector/industry = %q/%q, want Technology/Consumer Electronics", meta.Sector, meta.Industry)
	}
	if meta.Country != "United States" {
		t.Fatalf("country = %q, want United States", meta.Country)
	}
	if tr.profileHits != 1 {
		t.Fatalf("quoteSummary calls = %d, want 1 (no duplicate profile request)", tr.profileHits)
	}
}

// TestFetchMetaStockProfileFailureFallsBack: a failing profile must not fail
// the lookup: sector/industry stay empty and the country falls back to the
// exchange mapping.
func TestFetchMetaStockProfileFailureFallsBack(t *testing.T) {
	f, tr := newMetaFetcher(metaChartStock, "", http.StatusInternalServerError)

	meta, err := f.FetchMeta(context.Background(), "AAPL")
	if err != nil {
		t.Fatalf("FetchMeta: %v, want the lookup to survive a profile failure", err)
	}
	if meta.Sector != "" || meta.Industry != "" {
		t.Fatalf("sector/industry = %q/%q, want empty", meta.Sector, meta.Industry)
	}
	if meta.Country != "US" {
		t.Fatalf("country = %q, want the US exchange fallback", meta.Country)
	}
	if tr.profileHits != 1 {
		t.Fatalf("quoteSummary calls = %d, want 1", tr.profileHits)
	}
}

// TestFetchMetaEtfNeverRequestsAssetProfile: non-equity types must not ask
// the provider for assetProfile modules; the fund-category refinement still
// runs on its own modules.
func TestFetchMetaEtfNeverRequestsAssetProfile(t *testing.T) {
	f, tr := newMetaFetcher(metaChartEtf, metaFundBody, http.StatusOK)

	meta, err := f.FetchMeta(context.Background(), "VWCE")
	if err != nil {
		t.Fatalf("FetchMeta: %v", err)
	}
	if meta.Type != "etf" || meta.Country != "" || meta.Sector != "" || meta.Industry != "" {
		t.Fatalf("meta = %+v, want a plain etf without issuer fields", meta)
	}
	for _, modules := range tr.modules {
		if strings.Contains(modules, "assetProfile") {
			t.Fatalf("etf lookup requested assetProfile modules %q", modules)
		}
	}
}
