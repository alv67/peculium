package price

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/shopspring/decimal"

	"github.com/alv67/peculium/internal/geo"
	"github.com/alv67/peculium/internal/model"
)

// ETFFetcher resolves the geographic and sector exposure of an ETF from its ISIN
// and can look up an ETF ISIN from a ticker through the python-service.
type ETFFetcher interface {
	FetchExposure(ctx context.Context, isin string) (*model.AssetExposure, error)
	FetchMorningstarExposure(ctx context.Context, isin string) (*model.AssetExposure, error)
	SearchTicker(ctx context.Context, query string) ([]EtfSearchResult, error)
}

// EtfSearchResult is an ETF matched by the python-service ticker search.
type EtfSearchResult struct {
	ISIN   string `json:"isin"`
	Name   string `json:"name"`
	Ticker string `json:"ticker"`
}

// JustETFFetcher fetches ETF exposure from the python-service microservice.
type JustETFFetcher struct {
	baseURL string
	client  *http.Client
}

const (
	etfMaxRetries = 2
	etfRetryDelay = 400 * time.Millisecond
)

type etfExposureRow struct {
	Name   string  `json:"name"`
	Weight float64 `json:"weight"`
}

type etfExposureResponse struct {
	Countries []etfExposureRow `json:"countries"`
	Regions   []etfExposureRow `json:"regions"`
	Sectors   []etfExposureRow `json:"sectors"`
}

type etfErrorResponse struct {
	Detail string `json:"detail"`
}

// NewJustETFFetcher builds a fetcher that talks to the python-service base URL.
func NewJustETFFetcher(baseURL string) *JustETFFetcher {
	return &JustETFFetcher{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  &http.Client{Timeout: 15 * time.Second},
	}
}

// FetchExposure calls the python-service exposure endpoint for an ISIN. Non-2xx
// responses are returned without retrying; network/timeout failures are retried
// up to etfMaxRetries times with a small backoff.
func (f *JustETFFetcher) FetchExposure(ctx context.Context, isin string) (*model.AssetExposure, error) {
	url := fmt.Sprintf("%s/api/v1/etf/%s/exposure", f.baseURL, isin)
	return f.fetchPath(ctx, url, isin)
}

// FetchMorningstarExposure calls the python-service Morningstar exposure
// endpoint for an ISIN, using the same retry pattern as FetchExposure.
func (f *JustETFFetcher) FetchMorningstarExposure(ctx context.Context, isin string) (*model.AssetExposure, error) {
	url := fmt.Sprintf("%s/api/v1/etf/%s/morningstar-exposure", f.baseURL, isin)
	return f.fetchPath(ctx, url, isin)
}

func (f *JustETFFetcher) fetchPath(ctx context.Context, url, isin string) (*model.AssetExposure, error) {
	var err error
	for attempt := 0; attempt <= etfMaxRetries; attempt++ {
		if attempt > 0 {
			if err := sleepCtx(ctx, etfRetryDelay); err != nil {
				return nil, err
			}
		}
		var exposure *model.AssetExposure
		exposure, err = f.fetchOnce(ctx, url)
		if err == nil {
			return exposure, nil
		}
		var transportErr *etfTransportError
		if !errors.As(err, &transportErr) {
			return nil, err
		}
	}
	return nil, err
}

func (f *JustETFFetcher) fetchOnce(ctx context.Context, url string) (*model.AssetExposure, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, &etfTransportError{err: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, &etfStatusError{status: resp.StatusCode, message: statusMessage(resp.StatusCode, body)}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var result etfExposureResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal: %w (body: %s)", err, string(body))
	}

	return &model.AssetExposure{
		Countries: exposureRows(result.Countries),
		Regions:   exposureRows(result.Regions),
		Sectors:   exposureRows(result.Sectors),
	}, nil
}

// statusMessage builds a descriptive error for a non-2xx response, including
// the upstream detail when the body is the documented JSON error.
func statusMessage(status int, body []byte) string {
	msg := fmt.Sprintf("python service returned status %d", status)
	var errResp etfErrorResponse
	if json.Unmarshal(body, &errResp) == nil && errResp.Detail != "" {
		msg = fmt.Sprintf("%s: %s", msg, errResp.Detail)
	}
	return msg
}

// SearchTicker resolves a ticker (or name fragment) to ETF ISINs through the
// python-service search endpoint. Network failures are retried like FetchExposure.
func (f *JustETFFetcher) SearchTicker(ctx context.Context, query string) ([]EtfSearchResult, error) {
	url := fmt.Sprintf("%s/api/v1/etf/search?q=%s", f.baseURL, url.QueryEscape(query))

	var err error
	for attempt := 0; attempt <= etfMaxRetries; attempt++ {
		if attempt > 0 {
			if err := sleepCtx(ctx, etfRetryDelay); err != nil {
				return nil, err
			}
		}
		var results []EtfSearchResult
		results, err = f.searchOnce(ctx, url)
		if err == nil {
			return results, nil
		}
		var transportErr *etfTransportError
		if !errors.As(err, &transportErr) {
			return nil, err
		}
	}
	return nil, err
}

func (f *JustETFFetcher) searchOnce(ctx context.Context, url string) ([]EtfSearchResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, &etfTransportError{err: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, &etfStatusError{status: resp.StatusCode, message: statusMessage(resp.StatusCode, body)}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var results []EtfSearchResult
	if err := json.Unmarshal(body, &results); err != nil {
		return nil, fmt.Errorf("unmarshal: %w (body: %s)", err, string(body))
	}
	return results, nil
}

// BestMatch picks the best ETF search result for an asset, preferring an exact
// ticker match, then the result whose name shares the most words with the asset
// name (guards against fuzzy search picking an unrelated fund), and finally the
// first result. Returns false when no result carries an ISIN.
func BestMatch(results []EtfSearchResult, ticker, assetName string) (EtfSearchResult, bool) {
	if len(results) == 0 {
		return EtfSearchResult{}, false
	}
	if ticker != "" {
		for _, r := range results {
			if r.ISIN != "" && strings.EqualFold(r.Ticker, ticker) {
				return r, true
			}
		}
	}
	best := EtfSearchResult{}
	bestScore := -1
	for _, r := range results {
		if r.ISIN == "" {
			continue
		}
		score := nameOverlap(r.Name, assetName)
		if score > bestScore {
			best, bestScore = r, score
		}
	}
	if best.ISIN == "" {
		return EtfSearchResult{}, false
	}
	return best, true
}

// nameOverlap counts how many distinct words of a are matched by a word in b,
// in either direction (prefix match included). Used to rank search results
// against the asset name.
func nameOverlap(a, b string) int {
	wa := tokenize(strings.ToLower(a))
	wb := tokenize(strings.ToLower(b))
	score := 0
	for _, x := range wa {
		for _, y := range wb {
			if x == y || strings.HasPrefix(x, y) || strings.HasPrefix(y, x) {
				score++
				break
			}
		}
	}
	return score
}

func tokenize(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

// exposureRows converts the raw python-service rows into model rows.
func exposureRows(rows []etfExposureRow) []model.ExposureRow {
	out := make([]model.ExposureRow, 0, len(rows))
	for _, row := range rows {
		if row.Name == "" || row.Weight <= 0 {
			continue
		}
		out = append(out, model.ExposureRow{Name: row.Name, Weight: decimal.NewFromFloat(row.Weight)})
	}
	return out
}

// AggregateRegions maps raw country names to canonical macro-regions, summing
// the weights of countries in the same region. Country names that resolve to no
// ISO code land in the geo package's fallback region. When the aggregated
// weights do not cover 100% (some providers expose only the countries above a
// threshold, or drop non-canonical ones), the residual is added to the
// "Other / Not Classified" region so the region dimension always sums to 100.
func AggregateRegions(rows []model.ExposureRow) []model.ExposureRow {
	weights := map[string]decimal.Decimal{}
	for _, row := range rows {
		if row.Name == "" || !row.Weight.IsPositive() {
			continue
		}
		region := geo.RegionForCountry(geo.NormalizeCountry(row.Name))
		if region == "" {
			region = row.Name
		}
		weights[region] = weights[region].Add(row.Weight)
	}
	total := decimal.Zero
	for _, w := range weights {
		total = total.Add(w)
	}
	// If the mapped regions leave more than a rounding gap (0.5%), the residual
	// belongs to the "Other / Not Classified" bucket.
	if total.Sub(decimal.NewFromInt(100)).Abs().GreaterThan(decimal.NewFromFloat(0.5)) {
		residual := decimal.NewFromInt(100).Sub(total)
		if residual.IsPositive() {
			weights[geo.OtherRegion] = weights[geo.OtherRegion].Add(residual)
		}
	}
	return sortedExposureRows(weights)
}

// AggregateSectors maps raw provider sector names to canonical GICS sectors,
// summing the weights of names that normalize to the same sector.
func AggregateSectors(rows []model.ExposureRow) []model.ExposureRow {
	weights := map[string]decimal.Decimal{}
	for _, row := range rows {
		if row.Name == "" || !row.Weight.IsPositive() {
			continue
		}
		sector := geo.NormalizeSector(row.Name)
		if sector == "" {
			continue
		}
		weights[sector] = weights[sector].Add(row.Weight)
	}
	return sortedExposureRows(weights)
}

func sortedExposureRows(weights map[string]decimal.Decimal) []model.ExposureRow {
	rows := make([]model.ExposureRow, 0, len(weights))
	for name, weight := range weights {
		rows = append(rows, model.ExposureRow{Name: name, Weight: weight})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Name < rows[j].Name })
	return rows
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	select {
	case <-time.After(d):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// etfStatusError reports a non-2xx response from the python-service.
type etfStatusError struct {
	status  int
	message string
}

func (e *etfStatusError) Error() string { return e.message }

// etfTransportError reports a network/timeout failure reaching the
// python-service. It is retryable, unlike status and parse errors.
type etfTransportError struct {
	err error
}

func (e *etfTransportError) Error() string { return fmt.Sprintf("python service request: %v", e.err) }
func (e *etfTransportError) Unwrap() error { return e.err }
