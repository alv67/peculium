package price

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"

	"github.com/alv67/peculium/internal/model"
	"github.com/alv67/peculium/internal/repository"
)

// HealthRecorder defines the interface for logging price fetch events.
type HealthRecorder interface {
	RecordEvent(ctx context.Context, event *model.HealthEvent) error
}

type YahooFetcher struct {
	repos             *repository.Repository
	client            *http.Client
	refreshInterval   time.Duration
	minInterval       time.Duration
	budget            RateBudget
	historyCooldown   map[uuid.UUID]time.Time
	fxHistoryCooldown map[string]time.Time
	splitCooldown     map[uuid.UUID]time.Time
	throttle          *throttler
	mu                sync.Mutex
	health            HealthRecorder
	crumbValue        string
	crumbCookie       string
	crumbAt           time.Time
}

type YahooFetcherOption func(*YahooFetcher)

// WithMinInterval sets the minimum gap between consecutive Yahoo HTTP calls.
func WithMinInterval(d time.Duration) YahooFetcherOption {
	return func(f *YahooFetcher) { f.minInterval = d }
}

// WithRateBudget sets the global rate budget shared across all fetchers.
func WithRateBudget(b RateBudget) YahooFetcherOption {
	return func(f *YahooFetcher) { f.budget = b }
}

// WithHealthRecorder sets the service used to record fetch health events.
func WithHealthRecorder(hr HealthRecorder) YahooFetcherOption {
	return func(f *YahooFetcher) { f.health = hr }
}

// recordHealth records a fetch outcome in the health log. It is a no-op when
// no recorder is configured and never fails the caller: health tracking must
// not break price fetching. assetID is nil for non per-asset events (FX,
// batch summaries). since anchors when the measured operation started; the
// duration is always computed here, never passed by hand. A context that is
// already done is handled by the recorder (HealthService writes on a fresh
// short-lived context), so terminal outcomes of timed-out operations land.
func (f *YahooFetcher) recordHealth(ctx context.Context, assetID *uuid.UUID, eventType, status, code, message string, since time.Time) {
	if f.health == nil {
		return
	}
	ev := &model.HealthEvent{
		ID:         uuid.New(),
		AssetID:    assetID,
		EventType:  eventType,
		Status:     status,
		Code:       code,
		Message:    message,
		DurationMs: int(time.Since(since).Milliseconds()),
		CreatedAt:  time.Now().UTC(),
	}
	if err := f.health.RecordEvent(ctx, ev); err != nil {
		log.Warn().Err(err).Str("event_type", eventType).Msg("failed to record health event")
	}
}

// NewYahooFetcher builds a fetcher that paces every Yahoo HTTP call through a
// shared throttler (400ms minimum gap, no global budget by default).
func NewYahooFetcher(repos *repository.Repository, refreshInterval time.Duration, opts ...YahooFetcherOption) *YahooFetcher {
	f := &YahooFetcher{
		repos:             repos,
		refreshInterval:   refreshInterval,
		historyCooldown:   map[uuid.UUID]time.Time{},
		fxHistoryCooldown: map[string]time.Time{},
		splitCooldown:     map[uuid.UUID]time.Time{},
		minInterval:       400 * time.Millisecond,
		budget:            NoopBudget{},
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
	for _, opt := range opts {
		opt(f)
	}
	f.throttle = newThrottler(f.minInterval, f.budget)
	return f
}

// doRequest runs the HTTP call through the shared throttler so that every
// Yahoo request respects the pacing and the global rate budget.
func (f *YahooFetcher) doRequest(ctx context.Context, req *http.Request) (*http.Response, error) {
	var resp *http.Response
	err := f.throttle.Do(ctx, func(ctx context.Context) error {
		var err error
		resp, err = f.client.Do(req.WithContext(ctx))
		return err
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

type fetchStatusError struct {
	status  int
	message string
}

func (e *fetchStatusError) Error() string { return e.message }

// statusOf returns the HTTP status wrapped in err, or 0 if err is not a
// fetchStatusError.
func statusOf(err error) int {
	var fse *fetchStatusError
	if errors.As(err, &fse) {
		return fse.status
	}
	return 0
}

type yahooChartQuote struct {
	Open   []*float64 `json:"open"`
	High   []*float64 `json:"high"`
	Low    []*float64 `json:"low"`
	Close  []*float64 `json:"close"`
	Volume []*int64   `json:"volume"`
}

type yahooChartResult struct {
	Meta struct {
		Symbol           string `json:"symbol"`
		Currency         string `json:"currency"`
		ExchangeName     string `json:"exchangeName"`
		FullExchangeName string `json:"fullExchangeName"`
		InstrumentType   string `json:"instrumentType"`
		LongName         string `json:"longName"`
		ShortName        string `json:"shortName"`
	} `json:"meta"`
	Timestamp  []int64 `json:"timestamp"`
	Indicators struct {
		Quote []yahooChartQuote `json:"quote"`
	} `json:"indicators"`
	Events *yahooChartEvents `json:"events"`
}

type yahooChartEvents struct {
	Splits map[string]yahooSplitEvent `json:"splits"`
}

type yahooSplitEvent struct {
	Date        int64   `json:"date"`
	Numerator   float64 `json:"numerator"`
	Denominator float64 `json:"denominator"`
	SplitRatio  string  `json:"splitRatio"`
}

type yahooChartResponse struct {
	Chart struct {
		Result []yahooChartResult `json:"result"`
		Error  interface{}        `json:"error"`
	} `json:"chart"`
}

type historyBar struct {
	Date  time.Time
	Close decimal.Decimal
}

type HistoryAsset struct {
	ID     uuid.UUID
	Ticker string
	From   time.Time
	Full   bool // true = backfill completo dall'inizio disponibile su Yahoo
}

// FetchAll refreshes prices for every asset in the DB that is stale and keeps
// the USD->X FX rates fresh. Used by the worker.
func (f *YahooFetcher) FetchAll(ctx context.Context) error {
	assets, err := f.repos.Asset.ListYahoo(ctx)
	if err != nil {
		return fmt.Errorf("list assets: %w", err)
	}

	report, err := f.RefreshStale(ctx, assets)
	if err != nil {
		return err
	}
	if len(report.Refreshed) > 0 {
		log.Info().Strs("symbols", report.Refreshed).Msg("prices refreshed")
	} else {
		log.Info().Msg("no stale prices to refresh")
	}
	for _, iss := range report.Issues {
		log.Warn().Str("symbol", iss.Symbol).Str("code", iss.Code).Msg(iss.Message)
	}

	issues, err := f.RefreshFX(ctx)
	if err != nil {
		log.Warn().Err(err).Msg("fx refresh failed")
		return nil
	}
	for _, iss := range issues {
		log.Warn().Str("symbol", iss.Symbol).Str("code", iss.Code).Msg(iss.Message)
	}
	f.EnsureFXHistory(ctx)
	return nil
}

// RefreshFX keeps USD->X rates for every enabled whitelisted currency up to
// date, using the same staleness/throttle logic as asset prices.
func (f *YahooFetcher) RefreshFX(ctx context.Context) ([]FetchIssue, error) {
	currencies, err := f.repos.Currency.ListEnabled(ctx)
	if err != nil {
		return nil, fmt.Errorf("list currencies: %w", err)
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	issues := make([]FetchIssue, 0)
	now := time.Now().UTC()
	for _, cur := range currencies {
		quote := cur.Code
		if quote == "USD" || quote == "" {
			continue
		}
		fetchedAt, err := f.repos.FX.FetchedAt(ctx, quote)
		if err == nil && fetchedAt.Add(f.refreshInterval).After(now) {
			continue
		}

		start := time.Now()
		rate, err := f.fetchFX(ctx, quote)
		if err != nil {
			msg := fmt.Sprintf("fx %s: %v", quote, err)
			issues = append(issues, FetchIssue{Symbol: quote, RequestType: "fx", Code: issueCode(err), Message: msg})
			log.Warn().Err(err).Str("currency", quote).Msg("fx fetch failed")
			f.recordHealth(ctx, nil, "fx_fetch", "failure", issueCode(err), msg, start)
			continue
		}
		if err := f.repos.FX.Upsert(ctx, "USD", quote, rate); err != nil {
			msg := fmt.Sprintf("fx %s: fx save failed: %v", quote, err)
			issues = append(issues, FetchIssue{Symbol: quote, RequestType: "fx", Code: "error", Message: msg})
			log.Warn().Err(err).Str("currency", quote).Msg("fx save failed")
			f.recordHealth(ctx, nil, "fx_fetch", "failure", "error", msg, start)
			continue
		}
		f.recordHealth(ctx, nil, "fx_fetch", "success", "", quote+" rate updated", start)
		log.Info().Str("currency", quote).Str("rate", rate.String()).Msg("fx rate updated")
	}
	return issues, nil
}

func (f *YahooFetcher) fetchFX(ctx context.Context, quote string) (decimal.Decimal, error) {
	chart, err := f.fetchChart(ctx, "USD"+quote+"=X")
	if err != nil {
		return decimal.Zero, err
	}
	if len(chart.Indicators.Quote) == 0 {
		return decimal.Zero, fmt.Errorf("no quote indicators")
	}
	quoteData := chart.Indicators.Quote[0]

	idx := -1
	for i := len(chart.Timestamp) - 1; i >= 0; i-- {
		if i < len(quoteData.Close) && quoteData.Close[i] != nil {
			idx = i
			break
		}
	}
	if idx == -1 {
		return decimal.Zero, fmt.Errorf("no fx rate available")
	}
	return decimal.NewFromFloat(*quoteData.Close[idx]), nil
}

// FetchFXRate returns the latest USD->quote close from Yahoo. It is used to
// validate that a currency can be managed before adding it to the whitelist.
func (f *YahooFetcher) FetchFXRate(ctx context.Context, quote string) (decimal.Decimal, error) {
	return f.fetchFX(ctx, quote)
}

// fetchChart performs one Yahoo chart request and returns the parsed result.
func (f *YahooFetcher) fetchChart(ctx context.Context, ticker string) (*yahooChartResult, error) {
	url := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s?interval=1d&range=5d", ticker)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)")

	resp, err := f.doRequest(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("yahoo request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, resp.Body)
		return nil, &fetchStatusError{status: resp.StatusCode, message: fmt.Sprintf("yahoo returned status %d", resp.StatusCode)}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var result yahooChartResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal: %w (body: %s)", err, string(body))
	}
	if result.Chart.Error != nil {
		return nil, fmt.Errorf("yahoo error: %v", result.Chart.Error)
	}
	if len(result.Chart.Result) == 0 {
		return nil, fmt.Errorf("no chart data")
	}

	chart := &result.Chart.Result[0]
	return chart, nil
}

func (f *YahooFetcher) fetchChartRange(ctx context.Context, ticker string, from, to time.Time) ([]historyBar, error) {
	url := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s?interval=1d&period1=%d&period2=%d", ticker, from.Unix(), to.Unix())

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)")

	resp, err := f.doRequest(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("yahoo request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, resp.Body)
		return nil, &fetchStatusError{status: resp.StatusCode, message: fmt.Sprintf("yahoo returned status %d", resp.StatusCode)}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var result yahooChartResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal: %w (body: %s)", err, string(body))
	}
	if result.Chart.Error != nil {
		return nil, fmt.Errorf("yahoo error: %v", result.Chart.Error)
	}
	if len(result.Chart.Result) == 0 {
		return nil, fmt.Errorf("no chart data")
	}

	chart := &result.Chart.Result[0]
	var bars []historyBar
	if len(chart.Indicators.Quote) > 0 {
		quote := chart.Indicators.Quote[0]
		for i, ts := range chart.Timestamp {
			if i < len(quote.Close) && quote.Close[i] != nil {
				bars = append(bars, historyBar{
					Date:  priceDate(ts),
					Close: decimal.NewFromFloat(*quote.Close[i]),
				})
			}
		}
	}
	if len(bars) == 0 {
		return nil, fmt.Errorf("no history bars")
	}
	return bars, nil
}

// fullHistoryStart is the period1 used for full history backfills, far enough
// back to cover every asset Yahoo has data for.
var fullHistoryStart = time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)

// ErrSyncIncomplete reports a sync that finished but did not persist
// everything (bars/splits lost, typically because the operation ran out of
// time). Callers that map a sync outcome to a job status turn it into
// "partial" instead of "done", so a truncated download is never reported as a
// success. It is not a hard failure for the surrounding app load.
var ErrSyncIncomplete = errors.New("sync incomplete")

// EnsureHistory backfills daily closes from Yahoo for the given assets so the
// portfolio history series has market values from the first transaction to now.
func (f *YahooFetcher) EnsureHistory(ctx context.Context, assets []HistoryAsset) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	now := time.Now().UTC()
	incomplete := false
	for _, a := range assets {
		earliest, latest, err := f.repos.Price.MinMaxDate(ctx, a.ID)
		if err != nil {
			log.Warn().Err(err).Str("symbol", a.Ticker).Msg("history range failed")
			continue
		}

		var bars []historyBar
		start := time.Now()
		switch {
		case a.Full:
			// Explicit full backfill from the beginning of Yahoo history,
			// regardless of any partially stored data. No cooldown.
			bars, err = f.fetchChartRange(ctx, a.Ticker, fullHistoryStart, now)
		case latest == nil || earliest == nil || earliest.After(a.From.Add(30*24*time.Hour)):
			if last, ok := f.historyCooldown[a.ID]; ok && last.Add(f.refreshInterval).After(now) {
				continue
			}
			bars, err = f.fetchChartRange(ctx, a.Ticker, a.From, now)
			f.historyCooldown[a.ID] = now
		case latest.Add(f.refreshInterval).Before(now):
			bars, err = f.fetchChartRange(ctx, a.Ticker, *latest, now)
		default:
			continue
		}
		if err != nil {
			log.Warn().Err(err).Str("symbol", a.Ticker).Msg("history fetch failed")
			f.recordHealth(ctx, &a.ID, "history_fetch", "failure", issueCode(err), a.Ticker+": "+err.Error(), start)
			incomplete = true
			continue
		}
		// One multi-row upsert (chunked) instead of one statement per bar: the
		// per-bar round-trips were what pushed a long backfill past its
		// deadline. Dedupe by date first — Postgres rejects an ON CONFLICT
		// affecting the same row twice in a single INSERT.
		prices := make([]*model.Price, 0, len(bars))
		byDate := make(map[time.Time]int, len(bars))
		for _, b := range bars {
			p := &model.Price{AssetID: a.ID, Date: b.Date, Close: b.Close, Source: "yahoo"}
			if idx, ok := byDate[b.Date]; ok {
				prices[idx] = p
				continue
			}
			byDate[b.Date] = len(prices)
			prices = append(prices, p)
		}
		total := len(prices)
		written, saveErr := f.repos.Price.UpsertBatch(ctx, prices)
		failed := total - written
		if failed > 0 {
			// The duration is the whole per-asset operation, not just the save
			// phase: the save fails *because* the operation ran out of time, so
			// a save-only duration (a few hundred ms) would be misleading.
			f.recordHealth(ctx, &a.ID, "history_save", "failure", "error",
				fmt.Sprintf("%s: %d of %d bars failed to save: %v", a.Ticker, failed, total, saveErr), start)
			incomplete = true
		}
		markOK := true
		if a.Full {
			if err := f.repos.Asset.MarkHistoryBackfilled(ctx, a.ID); err != nil {
				log.Warn().Err(err).Str("symbol", a.Ticker).Msg("history backfill mark failed")
				f.recordHealth(ctx, &a.ID, "history_backfill", "failure", "error",
					a.Ticker+": mark history backfilled failed: "+err.Error(), start)
				markOK = false
				incomplete = true
			}
		}
		// Success event per asset: the health page lists every backfill call,
		// not just the failures. Assets skipped by cooldown/up-to-date never
		// reach here, so a row reflects real work.
		if failed == 0 && markOK {
			f.recordHealth(ctx, &a.ID, "history_backfill", "success", "",
				fmt.Sprintf("%s: %d bars saved", a.Ticker, total), start)
		}
	}
	if incomplete {
		return fmt.Errorf("%w: some history was not persisted", ErrSyncIncomplete)
	}
	return nil
}

// EnsureFXHistory backfills the per-date USD->X rate history from Yahoo for
// every enabled currency referenced by a transaction, so the series engine can
// resolve FX rates as of each portfolio day. Failures are logged and never
// fatal: without history the resolver falls back to the latest snapshot.
func (f *YahooFetcher) EnsureFXHistory(ctx context.Context) {
	f.mu.Lock()
	defer f.mu.Unlock()

	currencies, err := f.repos.Currency.ListEnabled(ctx)
	if err != nil {
		log.Warn().Err(err).Msg("fx history: list currencies failed")
		return
	}
	minByCurrency, err := f.repos.Transaction.MinDateByCurrency(ctx)
	if err != nil {
		log.Warn().Err(err).Msg("fx history: min transaction dates failed")
		return
	}

	now := time.Now().UTC()
	for _, cur := range currencies {
		quote := cur.Code
		if quote == "" || quote == "USD" {
			continue
		}
		minDate, ok := minByCurrency[quote]
		if !ok {
			continue
		}

		_, latest, err := f.repos.FX.MinMaxDate(ctx, "USD", quote)
		if err != nil {
			log.Warn().Err(err).Str("currency", quote).Msg("fx history range failed")
			continue
		}

		var from time.Time
		switch {
		case latest == nil:
			// Full backfill from the first transaction day of this currency.
			if last, ok := f.fxHistoryCooldown[quote]; ok && last.Add(f.refreshInterval).After(now) {
				continue
			}
			from = time.Date(minDate.Year(), minDate.Month(), minDate.Day(), 0, 0, 0, 0, time.UTC)
		case now.After(latest.Add(f.refreshInterval)):
			if last, ok := f.fxHistoryCooldown[quote]; ok && last.Add(f.refreshInterval).After(now) {
				continue
			}
			from = time.Date(latest.Year(), latest.Month(), latest.Day(), 0, 0, 0, 0, time.UTC)
		default:
			continue
		}
		f.fxHistoryCooldown[quote] = now

		start := time.Now()
		bars, err := f.fetchChartRange(ctx, "USD"+quote+"=X", from, now)
		if err != nil {
			log.Warn().Err(err).Str("currency", quote).Msg("fx history fetch failed")
			f.recordHealth(ctx, nil, "fx_history_fetch", "failure", issueCode(err), quote+": "+err.Error(), start)
			continue
		}
		// One aggregate save event per currency, never per bar.
		saveStart := time.Now()
		failed, total := 0, len(bars)
		var firstErr error
		for _, b := range bars {
			if b.Date.Before(from) {
				continue // bars may start earlier than the requested range
			}
			if err := f.repos.FX.UpsertHistory(ctx, "USD", quote, b.Date, b.Close, "yahoo"); err != nil {
				log.Warn().Err(err).Str("currency", quote).Str("date", b.Date.Format("2006-01-02")).Msg("fx history save failed")
				if firstErr == nil {
					firstErr = err
				}
				failed++
			}
		}
		if failed > 0 {
			f.recordHealth(ctx, nil, "fx_history_save", "failure", "error",
				fmt.Sprintf("%s: %d of %d rate bars failed to save: %v", quote, failed, total, firstErr), saveStart)
		}
		f.recordHealth(ctx, nil, "fx_history_fetch", "success", "", quote+" history updated", start)
		log.Info().Str("currency", quote).Int("bars", len(bars)).Msg("fx history updated")
	}
}

func (f *YahooFetcher) fetchSplits(ctx context.Context, ticker string) ([]model.Split, error) {
	url := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s?interval=1wk&events=split&range=max", ticker)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)")

	resp, err := f.doRequest(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("yahoo request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, resp.Body)
		return nil, &fetchStatusError{status: resp.StatusCode, message: fmt.Sprintf("yahoo returned status %d", resp.StatusCode)}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var result yahooChartResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal: %w (body: %s)", err, string(body))
	}
	if result.Chart.Error != nil {
		return nil, fmt.Errorf("yahoo error: %v", result.Chart.Error)
	}
	if len(result.Chart.Result) == 0 || result.Chart.Result[0].Events == nil {
		return []model.Split{}, nil
	}

	var splits []model.Split
	for _, e := range result.Chart.Result[0].Events.Splits {
		if e.Numerator == 0 || e.Denominator == 0 {
			continue
		}
		splits = append(splits, model.Split{
			Date:        time.Unix(e.Date, 0).UTC().Truncate(24 * time.Hour),
			Numerator:   decimal.NewFromFloat(e.Numerator),
			Denominator: decimal.NewFromFloat(e.Denominator),
		})
	}
	return splits, nil
}

// EnsureSplits fetches the stock split events for each asset and upserts them,
// re-checking the list on every call outside the per-asset cooldown. Upsert is
// idempotent on (asset_id, date) so re-fetching is safe.
func (f *YahooFetcher) EnsureSplits(ctx context.Context, assets []*model.Asset) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	now := time.Now().UTC()
	incomplete := false
	for _, a := range assets {
		if a.PriceSource != "" && a.PriceSource != "yahoo" {
			continue
		}
		if last, ok := f.splitCooldown[a.ID]; ok && last.Add(f.refreshInterval).After(now) {
			continue
		}
		start := time.Now()
		splits, err := f.fetchSplits(ctx, a.Ticker)
		f.splitCooldown[a.ID] = now
		if err != nil {
			log.Warn().Err(err).Str("symbol", a.Ticker).Msg("split fetch failed")
			f.recordHealth(ctx, &a.ID, "split_fetch", "failure", issueCode(err), a.Ticker+": "+err.Error(), start)
			incomplete = true
			continue
		}
		// Aggregate save failures into one event per asset; stop as soon as
		// the context is gone (the rest counts as not saved).
		failed, total := 0, len(splits)
		var firstErr error
		for i, sp := range splits {
			if ctx.Err() != nil {
				failed += len(splits) - i
				break
			}
			sp.AssetID = a.ID
			if err := f.repos.Split.Upsert(ctx, &sp); err != nil {
				log.Warn().Err(err).Str("symbol", a.Ticker).Str("date", sp.Date.Format("2006-01-02")).Msg("split save failed")
				if firstErr == nil {
					firstErr = err
				}
				failed++
			}
		}
		if failed > 0 {
			f.recordHealth(ctx, &a.ID, "split_save", "failure", "error",
				fmt.Sprintf("%s: %d of %d splits failed to save: %v", a.Ticker, failed, total, firstErr), start)
			incomplete = true
		} else {
			// Success event per asset so the health page lists every split
			// check, not just the failures.
			f.recordHealth(ctx, &a.ID, "split_fetch", "success", "",
				fmt.Sprintf("%s: %d splits", a.Ticker, total), start)
		}
	}
	if incomplete {
		return fmt.Errorf("%w: some splits were not persisted", ErrSyncIncomplete)
	}
	return nil
}

// RefreshStaleForPortfolio refreshes prices for assets currently held in a
// portfolio. Used by POST /prices/refresh on page open.
func (f *YahooFetcher) RefreshStaleForPortfolio(ctx context.Context, portfolioID uuid.UUID) (RefreshReport, error) {
	assets, err := f.repos.Portfolio.HeldAssets(ctx, portfolioID)
	if err != nil {
		return RefreshReport{}, fmt.Errorf("list held assets: %w", err)
	}
	return f.RefreshStale(ctx, assets)
}

// RefreshStale fetches quotes from Yahoo only for assets whose stored close is
// stale (old close date) and that haven't been fetched recently. This keeps the
// number of calls to Yahoo Finance as low as possible.
func (f *YahooFetcher) RefreshStale(ctx context.Context, assets []*model.Asset) (RefreshReport, error) {
	report := RefreshReport{Refreshed: []string{}, Issues: []FetchIssue{}}
	f.mu.Lock()
	defer f.mu.Unlock()

	// Only Yahoo-priced assets are ever queried; non-Yahoo assets (manual quotes,
	// unsupported tickers) are skipped entirely and never reported as stale.
	yahoo := make([]*model.Asset, 0, len(assets))
	for _, a := range assets {
		if a.PriceSource == "" || a.PriceSource == "yahoo" {
			yahoo = append(yahoo, a)
		}
	}
	assets = yahoo

	if len(assets) == 0 {
		return report, nil
	}

	ids := make([]uuid.UUID, 0, len(assets))
	for _, a := range assets {
		ids = append(ids, a.ID)
	}

	latest, err := f.repos.Price.FindLatestForAssets(ctx, ids)
	if err != nil {
		return report, fmt.Errorf("find latest prices: %w", err)
	}

	now := time.Now().UTC()
	expected := latestExpectedClose(now)
	var stale []*model.Asset
	for _, a := range assets {
		lp, ok := latest[a.ID]
		if !ok {
			// No price stored yet: fetch.
			stale = append(stale, a)
			continue
		}
		if a.PriceFetchedAt != nil && a.PriceFetchedAt.Add(f.refreshInterval).After(now) {
			// Fetched recently: skip to respect throttling.
			continue
		}
		if !lp.Date.Before(expected) {
			// Stored close is already fresh enough.
			continue
		}
		stale = append(stale, a)
	}

	if len(stale) == 0 {
		return report, nil
	}

	log.Info().
		Strs("symbols", tickers(stale)).
		Str("expected_close", expected.Format("2006-01-02")).
		Msg("refreshing stale prices")

	batchStart := time.Now()
	report.Refreshed, report.Issues = f.fetchQuotesBatch(ctx, stale)
	for _, iss := range report.Issues {
		if iss.Code == "rate_limited" {
			report.RateLimited = true
			log.Warn().Int("batch_size", len(stale)).Msg("yahoo rate limited during batch refresh")
			break
		}
	}
	// Health events: one success summary per batch plus a failure per issue.
	// The batch duration is shared by all events of the run: individual
	// per-symbol timings are not tracked to keep the fetch code simple.
	if len(report.Refreshed) > 0 {
		f.recordHealth(ctx, nil, "price_refresh", "success", "",
			fmt.Sprintf("refreshed %d/%d symbols: %s", len(report.Refreshed), len(stale), strings.Join(report.Refreshed, ", ")), batchStart)
	}
	for _, iss := range report.Issues {
		f.recordHealth(ctx, iss.AssetID, "price_refresh", "failure", iss.Code, iss.Message, batchStart)
	}
	if len(report.Refreshed) == 0 {
		return report, nil
	}

	if err := f.repos.Asset.MarkPricesFetched(ctx, assetIDs(stale), now); err != nil {
		log.Warn().Err(err).Msg("failed to mark prices as fetched")
	}

	return report, nil
}

// priceDate returns the UTC calendar day the quote refers to.
func priceDate(marketTime int64) time.Time {
	return time.Unix(marketTime, 0).UTC().Truncate(24 * time.Hour)
}

// latestExpectedClose returns the most recent weekday (Mon-Fri) at UTC midnight.
// Used as the "latest close we should already have stored" reference point.
func latestExpectedClose(now time.Time) time.Time {
	d := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	for d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
		d = d.AddDate(0, 0, -1)
	}
	return d
}

func assetIDs(assets []*model.Asset) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(assets))
	for _, a := range assets {
		ids = append(ids, a.ID)
	}
	return ids
}

func tickers(assets []*model.Asset) []string {
	ts := make([]string, 0, len(assets))
	for _, a := range assets {
		ts = append(ts, a.Ticker)
	}
	return ts
}
