package service

import (
	"context"
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"

	"github.com/alv67/peculium/internal/cache"
	"github.com/alv67/peculium/internal/geo"
	"github.com/alv67/peculium/internal/model"
	"github.com/alv67/peculium/internal/price"
	"github.com/alv67/peculium/internal/repository"
)

type fakePortfolioRepo struct {
	portfolio  *model.Portfolio
	portfolios []*model.Portfolio
	holdings   []*model.Holding
}

func (f *fakePortfolioRepo) Create(ctx context.Context, p *model.Portfolio) (*model.Portfolio, error) {
	return p, nil
}
func (f *fakePortfolioRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.Portfolio, error) {
	return f.portfolio, nil
}
func (f *fakePortfolioRepo) FindByUser(ctx context.Context, userID uuid.UUID) ([]*model.Portfolio, error) {
	return f.portfolios, nil
}
func (f *fakePortfolioRepo) Update(ctx context.Context, p *model.Portfolio) error { return nil }
func (f *fakePortfolioRepo) Delete(ctx context.Context, id uuid.UUID) error       { return nil }
func (f *fakePortfolioRepo) HeldAssets(ctx context.Context, portfolioID uuid.UUID) ([]*model.Asset, error) {
	return nil, nil
}
func (f *fakePortfolioRepo) HoldingsDetailed(ctx context.Context, portfolioIDs []uuid.UUID) ([]*model.Holding, error) {
	return f.holdings, nil
}
func (f *fakePortfolioRepo) FindAll(ctx context.Context) ([]uuid.UUID, error) { return nil, nil }

type fakeExposureRepo struct {
	regions   map[string][]model.ExposureRow
	sectors   map[string][]model.ExposureRow
	countries map[string][]model.ExposureRow
	// provenance mirrors the asset_exposure_provenance table, keyed by asset id
	// then by dimension ("countries"/"regions"/"sectors").
	provenance map[string]map[string]model.ExposureProvenance
	// replace*Calls record every dimension write so tests can assert which
	// operations persist (PUT/save) and which stay read-only previews.
	replaceRegionsCalls   int
	replaceSectorsCalls   int
	replaceCountriesCalls int
	setProvenanceCalls    int
}

func (f *fakeExposureRepo) FindRegions(ctx context.Context, assetID uuid.UUID) ([]model.ExposureRow, error) {
	return f.regions[assetID.String()], nil
}
func (f *fakeExposureRepo) FindSectors(ctx context.Context, assetID uuid.UUID) ([]model.ExposureRow, error) {
	return f.sectors[assetID.String()], nil
}
func (f *fakeExposureRepo) FindCountries(ctx context.Context, assetID uuid.UUID) ([]model.ExposureRow, error) {
	return f.countries[assetID.String()], nil
}
func (f *fakeExposureRepo) ReplaceRegions(ctx context.Context, assetID uuid.UUID, rows []model.ExposureRow) error {
	f.replaceRegionsCalls++
	if f.regions == nil {
		f.regions = map[string][]model.ExposureRow{}
	}
	f.regions[assetID.String()] = rows
	return nil
}
func (f *fakeExposureRepo) ReplaceSectors(ctx context.Context, assetID uuid.UUID, rows []model.ExposureRow) error {
	f.replaceSectorsCalls++
	if f.sectors == nil {
		f.sectors = map[string][]model.ExposureRow{}
	}
	f.sectors[assetID.String()] = rows
	return nil
}
func (f *fakeExposureRepo) ReplaceCountries(ctx context.Context, assetID uuid.UUID, rows []model.ExposureRow) error {
	f.replaceCountriesCalls++
	if f.countries == nil {
		f.countries = map[string][]model.ExposureRow{}
	}
	f.countries[assetID.String()] = rows
	return nil
}
func (f *fakeExposureRepo) FindRegionsByAssets(ctx context.Context, assetIDs []uuid.UUID) (map[string][]model.ExposureRow, error) {
	return f.regions, nil
}
func (f *fakeExposureRepo) FindSectorsByAssets(ctx context.Context, assetIDs []uuid.UUID) (map[string][]model.ExposureRow, error) {
	return f.sectors, nil
}
func (f *fakeExposureRepo) FindCountriesByAssets(ctx context.Context, assetIDs []uuid.UUID) (map[string][]model.ExposureRow, error) {
	return f.countries, nil
}
func (f *fakeExposureRepo) FindProvenance(ctx context.Context, assetID uuid.UUID) (map[string]model.ExposureProvenance, error) {
	if f.provenance == nil {
		return nil, nil
	}
	return f.provenance[assetID.String()], nil
}
func (f *fakeExposureRepo) SetProvenance(ctx context.Context, assetID uuid.UUID, dimension, source string) error {
	f.setProvenanceCalls++
	if f.provenance == nil {
		f.provenance = map[string]map[string]model.ExposureProvenance{}
	}
	key := assetID.String()
	if f.provenance[key] == nil {
		f.provenance[key] = map[string]model.ExposureProvenance{}
	}
	f.provenance[key][dimension] = model.ExposureProvenance{Source: source, UpdatedAt: time.Now().UTC()}
	return nil
}

type fakeFXRepo struct {
	rates   map[string]decimal.Decimal
	history map[string][]model.FXRatePoint
}

func (f *fakeFXRepo) Upsert(ctx context.Context, base, quote string, rate decimal.Decimal) error {
	return nil
}
func (f *fakeFXRepo) LatestByQuotes(ctx context.Context, quotes []string) (map[string]decimal.Decimal, error) {
	return f.rates, nil
}
func (f *fakeFXRepo) FetchedAt(ctx context.Context, quote string) (*time.Time, error) {
	return nil, nil
}
func (f *fakeFXRepo) UpsertHistory(ctx context.Context, base, quote string, date time.Time, rate decimal.Decimal, source string) error {
	return nil
}
func (f *fakeFXRepo) MinMaxDate(ctx context.Context, base, quote string) (*time.Time, *time.Time, error) {
	return nil, nil, nil
}
func (f *fakeFXRepo) RateForDate(ctx context.Context, base, quote string, date time.Time) (decimal.Decimal, error) {
	return decimal.Zero, nil
}
func (f *fakeFXRepo) History(ctx context.Context, base, quote string) ([]model.FXRatePoint, error) {
	return f.history[quote], nil
}

type fakeAssetRepo struct {
	asset *model.Asset
	err   error
	// assets backs FindByIDs so the history/sync paths can be tested with a
	// mix of price sources.
	assets []*model.Asset
	// updateCalls and lastUpdate record asset writes so tests can assert the
	// exposure fetches persist nothing but the auto-resolved ISIN.
	updateCalls int
	lastUpdate  *model.Asset
}

func (f *fakeAssetRepo) Create(ctx context.Context, asset *model.Asset) (*model.Asset, error) {
	return asset, nil
}
func (f *fakeAssetRepo) Update(ctx context.Context, asset *model.Asset) (*model.Asset, error) {
	f.updateCalls++
	f.lastUpdate = asset
	return asset, nil
}
func (f *fakeAssetRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.Asset, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.asset == nil || f.asset.ID != id {
		return nil, pgx.ErrNoRows
	}
	return f.asset, nil
}
func (f *fakeAssetRepo) FindByTicker(ctx context.Context, ticker string) (*model.Asset, error) {
	return nil, nil
}
func (f *fakeAssetRepo) FindByIDs(ctx context.Context, ids []uuid.UUID) ([]*model.Asset, error) {
	return f.assets, nil
}
func (f *fakeAssetRepo) Search(ctx context.Context, query string) ([]*model.Asset, error) {
	return nil, nil
}
func (f *fakeAssetRepo) List(ctx context.Context) ([]*model.Asset, error) {
	return nil, nil
}
func (f *fakeAssetRepo) ListYahoo(ctx context.Context) ([]*model.Asset, error) {
	return nil, nil
}
func (f *fakeAssetRepo) AllStocks(ctx context.Context) ([]*model.Asset, error) {
	return nil, nil
}
func (f *fakeAssetRepo) MarkPricesFetched(ctx context.Context, ids []uuid.UUID, at time.Time) error {
	return nil
}
func (f *fakeAssetRepo) MarkHistoryBackfilled(ctx context.Context, id uuid.UUID) error {
	return nil
}
func (f *fakeAssetRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}
func (f *fakeAssetRepo) Currencies(ctx context.Context) ([]string, error) {
	return nil, nil
}

// fakeLookupRepo is an in-memory stand-in for the Redis-backed
// repository.LookupRepository. It counts reads and writes so the provider
// exposure cache tests can assert hit/miss/refresh behavior; misses return
// redis.Nil like the real repository.
type fakeLookupRepo struct {
	data     map[string][]byte
	getCalls int
	setCalls int
}

func (f *fakeLookupRepo) Get(ctx context.Context, key string) ([]byte, error) {
	f.getCalls++
	v, ok := f.data[key]
	if !ok {
		return nil, redis.Nil
	}
	return v, nil
}

func (f *fakeLookupRepo) Set(ctx context.Context, key string, data []byte, ttl time.Duration) error {
	f.setCalls++
	if f.data == nil {
		f.data = map[string][]byte{}
	}
	f.data[key] = data
	return nil
}

func newTestService(t *testing.T, p *fakePortfolioRepo, e *fakeExposureRepo, f *fakeFXRepo) *Service {
	t.Helper()
	repos := &repository.Repository{
		Asset: &fakeAssetRepo{},
		// Dashboard reads use USD as the user's base currency so the
		// USD-centric fixtures keep asserting USD-pivoted numbers.
		User:      &fakeUserRepo{user: &model.User{ID: uuid.New(), BaseCurrency: "USD"}},
		Portfolio: p,
		Exposure:  e,
		FX:        f,
	}
	return New(repos, nil, nil, nil, time.Minute, time.Hour, cache.New(nil), 0, 0, nil)
}

func newTestServiceWithAsset(t *testing.T, a *fakeAssetRepo) *Service {
	t.Helper()
	repos := &repository.Repository{
		Asset:     a,
		Portfolio: &fakePortfolioRepo{},
		Exposure:  &fakeExposureRepo{},
		FX:        &fakeFXRepo{},
	}
	return New(repos, nil, nil, nil, time.Minute, time.Hour, cache.New(nil), 0, 0, nil)
}

// fakeYahooFetcher stubs the yahooFetcher seam; only the profile calls carry
// canned data, the rest are inert zero-value stubs.
type fakeYahooFetcher struct {
	sector   string
	industry string
	country  string
	// weightings is what the provider reports as sectorWeightings; gotISIN
	// style counters record the calls so tests can assert the fetch ran.
	weightings           []model.ExposureRow
	profileExtendedCalls int
	// historyCalls/splitCalls count the backfill entries; historyTickers and
	// splitTickers record the assets actually forwarded so tests can assert
	// non-Yahoo assets never reach the fetcher.
	historyCalls   int
	splitCalls     int
	historyTickers []string
	splitTickers   []string
}

func (f *fakeYahooFetcher) FetchAssetProfile(ctx context.Context, ticker string) (string, string, string, error) {
	return f.sector, f.industry, f.country, nil
}
func (f *fakeYahooFetcher) FetchAssetProfileExtended(ctx context.Context, ticker string) (string, string, string, []model.ExposureRow, error) {
	f.profileExtendedCalls++
	return f.sector, f.industry, f.country, f.weightings, nil
}
func (f *fakeYahooFetcher) FetchMeta(ctx context.Context, ticker string) (*price.AssetMeta, error) {
	return nil, nil
}
func (f *fakeYahooFetcher) FetchFXRate(ctx context.Context, quote string) (decimal.Decimal, error) {
	return decimal.Zero, nil
}
func (f *fakeYahooFetcher) RefreshFX(ctx context.Context) ([]price.FetchIssue, error) {
	return nil, nil
}
func (f *fakeYahooFetcher) RefreshStale(ctx context.Context, assets []*model.Asset) (price.RefreshReport, error) {
	return price.RefreshReport{}, nil
}
func (f *fakeYahooFetcher) RefreshStaleForPortfolio(ctx context.Context, portfolioID uuid.UUID) (price.RefreshReport, error) {
	return price.RefreshReport{}, nil
}
func (f *fakeYahooFetcher) EnsureHistory(ctx context.Context, assets []price.HistoryAsset) error {
	f.historyCalls++
	for _, a := range assets {
		f.historyTickers = append(f.historyTickers, a.Ticker)
	}
	return nil
}
func (f *fakeYahooFetcher) EnsureSplits(ctx context.Context, assets []*model.Asset) error {
	f.splitCalls++
	for _, a := range assets {
		f.splitTickers = append(f.splitTickers, a.Ticker)
	}
	return nil
}

// fakeETFFetcher stubs price.ETFFetcher and records the ISIN each fetch ran
// with so tests can assert the ticker→ISIN auto-resolution result is used. The
// *Calls counters record how often the provider was actually reached, which
// lets the cache tests verify that a hit skips the fetch entirely.
type fakeETFFetcher struct {
	exposure        *model.AssetExposure
	morningstar     *model.AssetExposure
	search          []price.EtfSearchResult
	exposureISIN    string
	morningstarISIN string
	searchTicker    string
	exposureCalls   int
	morningstarCall int
}

func (f *fakeETFFetcher) FetchExposure(ctx context.Context, isin string) (*model.AssetExposure, error) {
	f.exposureISIN = isin
	f.exposureCalls++
	return f.exposure, nil
}
func (f *fakeETFFetcher) FetchMorningstarExposure(ctx context.Context, isin string) (*model.AssetExposure, error) {
	f.morningstarISIN = isin
	f.morningstarCall++
	return f.morningstar, nil
}
func (f *fakeETFFetcher) SearchTicker(ctx context.Context, query string) ([]price.EtfSearchResult, error) {
	f.searchTicker = query
	return f.search, nil
}

func newFetchTestService(t *testing.T, a *fakeAssetRepo, e *fakeExposureRepo, lk *fakeLookupRepo, yf yahooFetcher, etf price.ETFFetcher) *Service {
	t.Helper()
	if lk == nil {
		lk = &fakeLookupRepo{}
	}
	repos := &repository.Repository{
		Asset:     a,
		Portfolio: &fakePortfolioRepo{},
		Exposure:  e,
		FX:        &fakeFXRepo{},
		Lookup:    lk,
	}
	return New(repos, nil, yf, etf, time.Minute, time.Hour, cache.New(nil), 0, 0, nil)
}

// fakeTransactionRepo is a minimal TransactionRepository stand-in for the
// history/split sync paths: MinDateByAsset returns canned first-transaction
// dates, the rest are inert stubs.
type fakeTransactionRepo struct {
	minDates map[uuid.UUID]time.Time
	// txs backs FindByPortfoliosAsc so the cash-flow paths (dashboard
	// performance) can be tested with a fixed transaction ledger. It also
	// backs the paginated read, which filters it (mirroring the SQL WHERE)
	// and slices it honoring limit/offset.
	txs []model.TransactionWithAsset
	// lastFilter records the filter of the most recent paged/count call so
	// tests can assert the service threads it through.
	lastFilter model.TransactionFilter
}

func (f *fakeTransactionRepo) Create(ctx context.Context, tx *model.Transaction) (*model.Transaction, error) {
	return tx, nil
}
func (f *fakeTransactionRepo) FindByPortfolio(ctx context.Context, portfolioID uuid.UUID) ([]model.TransactionWithAsset, error) {
	return nil, nil
}
func (f *fakeTransactionRepo) FindByPortfolioPage(ctx context.Context, portfolioID uuid.UUID, filter model.TransactionFilter, limit, offset int) ([]model.TransactionWithAsset, error) {
	f.lastFilter = filter
	matched := f.matchTxs(filter)
	if offset >= len(matched) {
		return nil, nil
	}
	end := offset + limit
	if end > len(matched) {
		end = len(matched)
	}
	return matched[offset:end], nil
}
func (f *fakeTransactionRepo) CountByPortfolio(ctx context.Context, portfolioID uuid.UUID, filter model.TransactionFilter) (int64, error) {
	f.lastFilter = filter
	return int64(len(f.matchTxs(filter))), nil
}

// matchTxs applies the filter the same way the SQL WHERE does: exact type
// and asset id, inclusive bounds on the transaction's calendar date.
func (f *fakeTransactionRepo) matchTxs(filter model.TransactionFilter) []model.TransactionWithAsset {
	var out []model.TransactionWithAsset
	for _, tx := range f.txs {
		if filter.Type != "" && string(tx.Type) != filter.Type {
			continue
		}
		if filter.AssetID != nil && tx.AssetID != *filter.AssetID {
			continue
		}
		day := startOfUTCDay(tx.Date)
		if filter.From != nil && day.Before(startOfUTCDay(*filter.From)) {
			continue
		}
		if filter.To != nil && day.After(startOfUTCDay(*filter.To)) {
			continue
		}
		out = append(out, tx)
	}
	return out
}

func startOfUTCDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
func (f *fakeTransactionRepo) FindByPortfoliosAsc(ctx context.Context, portfolioIDs []uuid.UUID) ([]model.TransactionWithAsset, error) {
	return f.txs, nil
}
func (f *fakeTransactionRepo) MinDateByAsset(ctx context.Context, assetIDs []uuid.UUID) (map[uuid.UUID]time.Time, error) {
	if f.minDates == nil {
		return map[uuid.UUID]time.Time{}, nil
	}
	return f.minDates, nil
}
func (f *fakeTransactionRepo) MinDateByCurrency(ctx context.Context) (map[string]time.Time, error) {
	return nil, nil
}
func (f *fakeTransactionRepo) CountByAsset(ctx context.Context, assetID uuid.UUID) (int64, error) {
	return 0, nil
}
func (f *fakeTransactionRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.Transaction, error) {
	return nil, nil
}
func (f *fakeTransactionRepo) Update(ctx context.Context, tx *model.Transaction) error { return nil }
func (f *fakeTransactionRepo) Delete(ctx context.Context, id uuid.UUID) error          { return nil }

func newSyncTestService(t *testing.T, a *fakeAssetRepo, tx *fakeTransactionRepo, yf yahooFetcher) *Service {
	t.Helper()
	repos := &repository.Repository{
		Asset:       a,
		Portfolio:   &fakePortfolioRepo{},
		Transaction: tx,
		Exposure:    &fakeExposureRepo{},
		FX:          &fakeFXRepo{},
		Lookup:      &fakeLookupRepo{},
	}
	return New(repos, nil, yf, nil, time.Minute, time.Hour, cache.New(nil), 0, 0, nil)
}

func holding(id, currency, country, sector string, typ model.AssetType, qty, lastClose decimal.Decimal) *model.Holding {
	return &model.Holding{
		AssetID:    id,
		Currency:   currency,
		Country:    country,
		Sector:     sector,
		Type:       typ,
		Qty:        qty,
		LastClose:  lastClose,
		HasPrice:   true,
		AssetClass: "equity",
	}
}

func equalDecimal(a, b decimal.Decimal) bool { return a.Equal(b) }

func TestGetPortfolioGeographyAllocation_ETF(t *testing.T) {
	assetID := uuid.New()
	pf := &fakePortfolioRepo{
		portfolio: &model.Portfolio{Currency: "EUR"},
		holdings: []*model.Holding{
			holding(assetID.String(), "EUR", "", "", model.AssetTypeETF, decimal.NewFromInt(10), decimal.NewFromInt(100)),
		},
	}
	ex := &fakeExposureRepo{
		regions: map[string][]model.ExposureRow{
			assetID.String(): {
				{Name: "North America", Weight: decimal.NewFromInt(60)},
				{Name: "Europe Developed", Weight: decimal.NewFromInt(40)},
			},
		},
	}
	svc := newTestService(t, pf, ex, &fakeFXRepo{})

	got, err := svc.GetPortfolioGeographyAllocation(context.Background(), assetID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Currency != "EUR" {
		t.Fatalf("currency = %q, want EUR", got.Currency)
	}
	if len(got.Regions) != len(geo.Regions) {
		t.Fatalf("regions len = %d, want %d", len(got.Regions), len(geo.Regions))
	}
	for i, r := range got.Regions {
		if r.Region != geo.Regions[i] {
			t.Fatalf("region[%d] = %q, want %q (canonical order)", i, r.Region, geo.Regions[i])
		}
	}
	wantVal := decimal.NewFromInt(1000) // 10 * 100
	if !equalDecimal(got.Regions[0].Value, wantVal.Mul(decimal.NewFromInt(60)).Div(decimal.NewFromInt(100))) {
		t.Fatalf("North America value = %v, want 600", got.Regions[0].Value)
	}
	if !equalDecimal(got.Regions[3].Value, wantVal.Mul(decimal.NewFromInt(40)).Div(decimal.NewFromInt(100))) {
		t.Fatalf("Europe Developed value = %v, want 400", got.Regions[3].Value)
	}
	sum := decimal.Zero
	for _, r := range got.Regions {
		sum = sum.Add(r.Weight)
	}
	if !equalDecimal(sum, decimal.NewFromInt(100)) {
		t.Fatalf("weights sum = %v, want 100", sum)
	}
	for _, r := range got.Regions {
		if r.Region == "Other" {
			t.Fatalf("unexpected Other bucket for fully-mapped ETF")
		}
	}
}

func TestGetPortfolioGeographyAllocation_StockFallback(t *testing.T) {
	assetID := uuid.New()
	pf := &fakePortfolioRepo{
		portfolio: &model.Portfolio{Currency: "USD"},
		holdings: []*model.Holding{
			holding(assetID.String(), "USD", "US", "Technology", model.AssetTypeStock, decimal.NewFromInt(5), decimal.NewFromInt(200)),
		},
	}
	svc := newTestService(t, pf, &fakeExposureRepo{}, &fakeFXRepo{})

	geoAlloc, err := svc.GetPortfolioGeographyAllocation(context.Background(), assetID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	na := geoAlloc.Regions[0]
	if na.Region != "North America" {
		t.Fatalf("region[0] = %q, want North America", na.Region)
	}
	if !equalDecimal(na.Value, decimal.NewFromInt(1000)) {
		t.Fatalf("North America value = %v, want 1000", na.Value)
	}
	if !equalDecimal(na.Weight, decimal.NewFromInt(100)) {
		t.Fatalf("North America weight = %v, want 100", na.Weight)
	}

	secAlloc, err := svc.GetPortfolioSectorAllocation(context.Background(), assetID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(secAlloc.Sectors) != len(geo.GICSSectors) {
		t.Fatalf("sectors len = %d, want %d", len(secAlloc.Sectors), len(geo.GICSSectors))
	}
	for i, s := range secAlloc.Sectors {
		if s.Sector != geo.GICSSectors[i] {
			t.Fatalf("sector[%d] = %q, want %q (canonical order)", i, s.Sector, geo.GICSSectors[i])
		}
	}
	if secAlloc.Sectors[7].Sector != "Information Technology" {
		t.Fatalf("sector[7] = %q, want Information Technology", secAlloc.Sectors[7].Sector)
	}
	if !equalDecimal(secAlloc.Sectors[7].Value, decimal.NewFromInt(1000)) {
		t.Fatalf("IT value = %v, want 1000", secAlloc.Sectors[7].Value)
	}
	if !equalDecimal(secAlloc.Sectors[7].Weight, decimal.NewFromInt(100)) {
		t.Fatalf("IT weight = %v, want 100", secAlloc.Sectors[7].Weight)
	}
}

func TestGetPortfolioGeographyAllocation_FXConversion(t *testing.T) {
	assetID := uuid.New()
	pf := &fakePortfolioRepo{
		portfolio: &model.Portfolio{Currency: "EUR"},
		holdings: []*model.Holding{
			holding(assetID.String(), "USD", "", "", model.AssetTypeETF, decimal.NewFromInt(10), decimal.NewFromInt(100)),
		},
	}
	ex := &fakeExposureRepo{
		regions: map[string][]model.ExposureRow{
			assetID.String(): {{Name: "North America", Weight: decimal.NewFromInt(100)}},
		},
	}
	fx := &fakeFXRepo{rates: map[string]decimal.Decimal{"USD": decimal.NewFromInt(1), "EUR": decimal.RequireFromString("1.1")}}
	svc := newTestService(t, pf, ex, fx)

	got, err := svc.GetPortfolioGeographyAllocation(context.Background(), assetID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := decimal.NewFromInt(1000).Mul(decimal.RequireFromString("1.1")) // 1100
	if !equalDecimal(got.Regions[0].Value, want) {
		t.Fatalf("North America value = %v, want %v", got.Regions[0].Value, want)
	}
	if !equalDecimal(got.Regions[0].Weight, decimal.NewFromInt(100)) {
		t.Fatalf("North America weight = %v, want 100", got.Regions[0].Weight)
	}
}

func TestGetPortfolioGeographyAllocation_EmptyPortfolio(t *testing.T) {
	pf := &fakePortfolioRepo{
		portfolio: &model.Portfolio{Currency: "USD"},
		holdings:  []*model.Holding{},
	}
	svc := newTestService(t, pf, &fakeExposureRepo{}, &fakeFXRepo{})

	geoAlloc, err := svc.GetPortfolioGeographyAllocation(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(geoAlloc.Regions) != len(geo.Regions) {
		t.Fatalf("regions len = %d, want %d", len(geoAlloc.Regions), len(geo.Regions))
	}
	if geoAlloc.Countries == nil || len(geoAlloc.Countries) != 0 {
		t.Fatalf("countries = %v, want empty non-nil slice", geoAlloc.Countries)
	}
	for i, r := range geoAlloc.Regions {
		if r.Region != geo.Regions[i] {
			t.Fatalf("region[%d] = %q, want %q", i, r.Region, geo.Regions[i])
		}
		if !r.Value.IsZero() || !r.Weight.IsZero() {
			t.Fatalf("region %q not zero: value=%v weight=%v", r.Region, r.Value, r.Weight)
		}
		if r.Region == "Other" {
			t.Fatalf("unexpected Other bucket in empty portfolio")
		}
	}

	secAlloc, err := svc.GetPortfolioSectorAllocation(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(secAlloc.Sectors) != len(geo.GICSSectors) {
		t.Fatalf("sectors len = %d, want %d", len(secAlloc.Sectors), len(geo.GICSSectors))
	}
	for i, s := range secAlloc.Sectors {
		if s.Sector != geo.GICSSectors[i] {
			t.Fatalf("sector[%d] = %q, want %q", i, s.Sector, geo.GICSSectors[i])
		}
		if !s.Value.IsZero() || !s.Weight.IsZero() {
			t.Fatalf("sector %q not zero: value=%v weight=%v", s.Sector, s.Value, s.Weight)
		}
	}
}

func TestGetPortfolioGeographyAllocation_TotalZeroNoDenominator(t *testing.T) {
	assetID := uuid.New()
	pf := &fakePortfolioRepo{
		portfolio: &model.Portfolio{Currency: "EUR"},
		holdings: []*model.Holding{
			holding(assetID.String(), "JPY", "", "", model.AssetTypeETF, decimal.NewFromInt(10), decimal.NewFromInt(100)),
		},
	}
	// No JPY rate (only USD and EUR present): the holding's factor is missing,
	// so total stays zero and no value is assigned to any bucket.
	svc := newTestService(t, pf, &fakeExposureRepo{}, &fakeFXRepo{rates: map[string]decimal.Decimal{"USD": decimal.NewFromInt(1), "EUR": decimal.NewFromInt(1)}})

	got, err := svc.GetPortfolioGeographyAllocation(context.Background(), assetID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, r := range got.Regions {
		if !r.Value.IsZero() {
			t.Fatalf("region %q value = %v, want zero", r.Region, r.Value)
		}
		if !r.Weight.IsZero() {
			t.Fatalf("region %q weight = %v, want zero", r.Region, r.Weight)
		}
	}
}

func TestGetPortfolioGeographyAllocation_OtherBucket(t *testing.T) {
	assetID := uuid.New()
	pf := &fakePortfolioRepo{
		portfolio: &model.Portfolio{Currency: "USD"},
		holdings: []*model.Holding{
			holding(assetID.String(), "USD", "", "", model.AssetTypeETF, decimal.NewFromInt(10), decimal.NewFromInt(100)),
		},
	}
	// Eligible equity ETF (asset_class "equity") with no stored rows and an
	// empty country/sector: for geography the region default is
	// "Other / Not Classified" but (not a stock) the fallback does not map it,
	// and for the sector NormalizeSector("") returns "" so no default and the
	// weight sum is 0 -> Other. The holding is eligible, so it is covered
	// rather than excluded.
	svc := newTestService(t, pf, &fakeExposureRepo{}, &fakeFXRepo{})

	geoAlloc, err := svc.GetPortfolioGeographyAllocation(context.Background(), assetID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(geoAlloc.Regions) != len(geo.Regions)+1 {
		t.Fatalf("regions len = %d, want %d", len(geoAlloc.Regions), len(geo.Regions)+1)
	}
	other := geoAlloc.Regions[len(geoAlloc.Regions)-1]
	if other.Region != "Other" {
		t.Fatalf("last region = %q, want Other", other.Region)
	}
	if !equalDecimal(other.Value, decimal.NewFromInt(1000)) {
		t.Fatalf("Other value = %v, want 1000", other.Value)
	}
	if !equalDecimal(other.Weight, decimal.NewFromInt(100)) {
		t.Fatalf("Other weight = %v, want 100", other.Weight)
	}
	for i := 0; i < len(geoAlloc.Regions)-1; i++ {
		if !geoAlloc.Regions[i].Value.IsZero() {
			t.Fatalf("region %q value = %v, want zero", geoAlloc.Regions[i].Region, geoAlloc.Regions[i].Value)
		}
	}
	sum := decimal.Zero
	for _, r := range geoAlloc.Regions {
		sum = sum.Add(r.Weight)
	}
	if !equalDecimal(sum, decimal.NewFromInt(100)) {
		t.Fatalf("weights sum = %v, want 100", sum)
	}
	if !equalDecimal(geoAlloc.Covered, decimal.NewFromInt(1000)) {
		t.Fatalf("covered = %v, want 1000", geoAlloc.Covered)
	}
	if !geoAlloc.Excluded.IsZero() {
		t.Fatalf("excluded = %v, want zero", geoAlloc.Excluded)
	}

	secAlloc, err := svc.GetPortfolioSectorAllocation(context.Background(), assetID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(secAlloc.Sectors) != len(geo.GICSSectors)+1 {
		t.Fatalf("sectors len = %d, want %d", len(secAlloc.Sectors), len(geo.GICSSectors)+1)
	}
	sOther := secAlloc.Sectors[len(secAlloc.Sectors)-1]
	if sOther.Sector != "Other" {
		t.Fatalf("last sector = %q, want Other", sOther.Sector)
	}
	if !equalDecimal(sOther.Value, decimal.NewFromInt(1000)) {
		t.Fatalf("sector Other value = %v, want 1000", sOther.Value)
	}
	if !equalDecimal(secAlloc.Covered, decimal.NewFromInt(1000)) {
		t.Fatalf("covered = %v, want 1000", secAlloc.Covered)
	}
	if !secAlloc.Excluded.IsZero() {
		t.Fatalf("excluded = %v, want zero", secAlloc.Excluded)
	}
}

func TestGetPortfolioGeographyAllocation_CountryExposure(t *testing.T) {
	stockID := uuid.New()
	etfID := uuid.New()
	bondID := uuid.New()
	fxID := uuid.New()
	pf := &fakePortfolioRepo{
		portfolio: &model.Portfolio{Currency: "EUR"},
		holdings: []*model.Holding{
			holding(stockID.String(), "EUR", "US", "Technology", model.AssetTypeStock, decimal.NewFromInt(1), decimal.NewFromInt(100)),
			holding(etfID.String(), "EUR", "", "", model.AssetTypeETF, decimal.NewFromInt(1), decimal.NewFromInt(400)),
			holding(bondID.String(), "EUR", "DE", "", model.AssetTypeBond, decimal.NewFromInt(1), decimal.NewFromInt(100)),
			holding(fxID.String(), "USD", "GB", "Financials", model.AssetTypeStock, decimal.NewFromInt(1), decimal.NewFromInt(100)),
		},
	}
	ex := &fakeExposureRepo{
		countries: map[string][]model.ExposureRow{
			etfID.String(): {
				{Name: "US", Weight: decimal.NewFromInt(25)},
				{Name: "JP", Weight: decimal.NewFromInt(75)},
			},
		},
	}
	fx := &fakeFXRepo{rates: map[string]decimal.Decimal{"USD": decimal.NewFromInt(1), "EUR": decimal.RequireFromString("1.1")}}
	svc := newTestService(t, pf, ex, fx)

	got, err := svc.GetPortfolioGeographyAllocation(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Currency != "EUR" {
		t.Fatalf("currency = %q, want EUR", got.Currency)
	}

	// Equity-only universe with FX conversion (portfolio EUR, USD holding at
	// 1.1): the US stock falls back to its own country (100), the equity ETF
	// splits 25/75 over US/JP (100/300), the USD GB stock converts to 110 and
	// the DE bond is excluded by nature. Only non-zero buckets are returned,
	// sorted by value descending (JP 300, US 200, GB 110) over a 610 total.
	if len(got.Countries) != 3 {
		t.Fatalf("countries = %+v, want the non-zero JP, US and GB buckets only", got.Countries)
	}
	if got.Countries[0].Country != "JP" || !equalDecimal(got.Countries[0].Value, decimal.NewFromInt(300)) {
		t.Fatalf("countries[0] = %+v, want JP 300 (descending)", got.Countries[0])
	}
	assertDecimalInDelta(t, got.Countries[0].Weight, decimal.RequireFromString("49.18"), "0.01", "JP weight")
	if got.Countries[1].Country != "US" || !equalDecimal(got.Countries[1].Value, decimal.NewFromInt(200)) {
		t.Fatalf("countries[1] = %+v, want US 200", got.Countries[1])
	}
	assertDecimalInDelta(t, got.Countries[1].Weight, decimal.RequireFromString("32.79"), "0.01", "US weight")
	if got.Countries[2].Country != "GB" || !equalDecimal(got.Countries[2].Value, decimal.NewFromInt(110)) {
		t.Fatalf("countries[2] = %+v, want GB 110 (USD converted at 1.1)", got.Countries[2])
	}
	assertDecimalInDelta(t, got.Countries[2].Weight, decimal.RequireFromString("18.03"), "0.01", "GB weight")
	countryWeightSum := decimal.Zero
	for _, c := range got.Countries {
		if !c.Value.IsPositive() {
			t.Fatalf("zero-value country bucket leaked: %+v", c)
		}
		if c.Country == "DE" {
			t.Fatalf("excluded bond country DE present: %+v", c)
		}
		countryWeightSum = countryWeightSum.Add(c.Weight)
	}
	if !equalDecimal(countryWeightSum, decimal.NewFromInt(100)) {
		t.Fatalf("countries weight sum = %v, want 100", countryWeightSum)
	}

	// Countries share the region pipeline's FX conversion: the same converted
	// GB stock lands 110 in the United Kingdom region too.
	if uk := regionByName(t, got.Regions, "United Kingdom"); !equalDecimal(uk.Value, decimal.NewFromInt(110)) {
		t.Fatalf("United Kingdom region value = %v, want 110", uk.Value)
	}
	if !equalDecimal(got.Covered, decimal.NewFromInt(610)) {
		t.Fatalf("covered = %v, want 610", got.Covered)
	}
	if !equalDecimal(got.Excluded, decimal.NewFromInt(100)) {
		t.Fatalf("excluded = %v, want 100", got.Excluded)
	}
}

func regionByName(t *testing.T, regions []*model.RegionAllocation, name string) *model.RegionAllocation {
	t.Helper()
	for _, r := range regions {
		if r.Region == name {
			return r
		}
	}
	t.Fatalf("region %q not found", name)
	return nil
}

func sectorByName(t *testing.T, sectors []*model.SectorAllocation, name string) *model.SectorAllocation {
	t.Helper()
	for _, s := range sectors {
		if s.Sector == name {
			return s
		}
	}
	t.Fatalf("sector %q not found", name)
	return nil
}

func assertDecimalInDelta(t *testing.T, got, want decimal.Decimal, delta, what string) {
	t.Helper()
	if got.Sub(want).Abs().GreaterThan(decimal.RequireFromString(delta)) {
		t.Fatalf("%s = %v, want %v ± %s", what, got, want, delta)
	}
}

func TestGetDashboardAllocation_AggregatesAcrossPortfolios(t *testing.T) {
	etfID := uuid.New()
	stockID := uuid.New()
	pf := &fakePortfolioRepo{
		portfolios: []*model.Portfolio{
			{Currency: "USD"},
			{Currency: "EUR"},
		},
		holdings: []*model.Holding{
			holding(etfID.String(), "EUR", "", "", model.AssetTypeETF, decimal.NewFromInt(1), decimal.NewFromInt(100)),
			holding(stockID.String(), "USD", "IT", "Financials", model.AssetTypeStock, decimal.NewFromInt(1), decimal.NewFromInt(100)),
		},
	}
	ex := &fakeExposureRepo{
		regions: map[string][]model.ExposureRow{
			etfID.String(): {{Name: "North America", Weight: decimal.NewFromInt(100)}},
		},
	}
	fx := &fakeFXRepo{rates: map[string]decimal.Decimal{"EUR": decimal.RequireFromString("0.9")}}
	svc := newTestService(t, pf, ex, fx)

	got, err := svc.GetDashboardAllocation(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Currency != "USD" {
		t.Fatalf("currency = %q, want USD", got.Currency)
	}

	na := regionByName(t, got.Regions, "North America")
	assertDecimalInDelta(t, na.Value, decimal.RequireFromString("111.11"), "0.01", "North America value")
	assertDecimalInDelta(t, na.Weight, decimal.RequireFromString("52.63"), "0.01", "North America weight")

	eu := regionByName(t, got.Regions, "Europe Developed")
	assertDecimalInDelta(t, eu.Value, decimal.NewFromInt(100), "0.01", "Europe value")
	assertDecimalInDelta(t, eu.Weight, decimal.RequireFromString("47.37"), "0.01", "Europe weight")

	weightSum := decimal.Zero
	for _, r := range got.Regions {
		weightSum = weightSum.Add(r.Weight)
	}
	assertDecimalInDelta(t, weightSum, decimal.NewFromInt(100), "0.01", "regions weight sum")

	fin := sectorByName(t, got.Sectors, "Financials")
	if !fin.Value.IsPositive() {
		t.Fatalf("Financials value = %v, want positive", fin.Value)
	}

	// Both holdings are priced equities across the two portfolios: the class
	// allocation merges them into a single 100% bucket in the base currency.
	if len(got.Classes) != 1 {
		t.Fatalf("classes = %+v, want a single merged equity bucket", got.Classes)
	}
	if got.Classes[0].Class != "equity" {
		t.Fatalf("classes[0].class = %q, want equity", got.Classes[0].Class)
	}
	assertDecimalInDelta(t, got.Classes[0].Value, decimal.RequireFromString("211.11"), "0.01", "equity class value")
	assertDecimalInDelta(t, got.Classes[0].Weight, decimal.NewFromInt(100), "0.01", "equity class weight")

	// Both holdings (ETF equity + stock) are eligible: fully covered, nothing excluded.
	assertDecimalInDelta(t, got.Covered, decimal.RequireFromString("211.11"), "0.01", "covered value")
	if !got.Excluded.IsZero() {
		t.Fatalf("excluded = %v, want zero", got.Excluded)
	}
}

func TestGetDashboardAllocation_EmptyUser(t *testing.T) {
	svc := newTestService(t, &fakePortfolioRepo{}, &fakeExposureRepo{}, &fakeFXRepo{})

	got, err := svc.GetDashboardAllocation(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Currency != "USD" {
		t.Fatalf("currency = %q, want USD", got.Currency)
	}
	if got.Regions == nil || len(got.Regions) != 0 {
		t.Fatalf("regions = %v, want empty non-nil slice", got.Regions)
	}
	if got.Sectors == nil || len(got.Sectors) != 0 {
		t.Fatalf("sectors = %v, want empty non-nil slice", got.Sectors)
	}
	if got.Classes == nil || len(got.Classes) != 0 {
		t.Fatalf("classes = %v, want empty non-nil slice", got.Classes)
	}
	if got.Countries == nil || len(got.Countries) != 0 {
		t.Fatalf("countries = %v, want empty non-nil slice", got.Countries)
	}
	if !got.Covered.IsZero() {
		t.Fatalf("covered = %v, want zero", got.Covered)
	}
	if !got.Excluded.IsZero() {
		t.Fatalf("excluded = %v, want zero", got.Excluded)
	}
}

func TestGetDashboardAllocation_ExcludesNonEquity(t *testing.T) {
	stockID := uuid.New()
	bondID := uuid.New()
	unclID := uuid.New()
	pf := &fakePortfolioRepo{
		portfolio: &model.Portfolio{Currency: "USD"},
		portfolios: []*model.Portfolio{
			{Currency: "USD"},
		},
		holdings: []*model.Holding{
			holding(stockID.String(), "USD", "IT", "Financials", model.AssetTypeStock, decimal.NewFromInt(1), decimal.NewFromInt(100)),
			holding(bondID.String(), "USD", "", "", model.AssetTypeBond, decimal.NewFromInt(2), decimal.NewFromInt(100)),
			{
				AssetID:    unclID.String(),
				Currency:   "USD",
				Country:    "US",
				Sector:     "Technology",
				Type:       model.AssetTypeETF,
				AssetClass: "other",
				Qty:        decimal.NewFromInt(3),
				LastClose:  decimal.NewFromInt(100),
				HasPrice:   true,
			},
		},
	}
	svc := newTestService(t, pf, &fakeExposureRepo{}, &fakeFXRepo{})

	got, err := svc.GetDashboardAllocation(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only the eligible stock (100) participates: everything else (bond 200 +
	// unclassified ETF 300) is excluded and must not flow into buckets/weights.
	if !equalDecimal(got.Covered, decimal.NewFromInt(100)) {
		t.Fatalf("covered = %v, want 100", got.Covered)
	}
	if !equalDecimal(got.Excluded, decimal.NewFromInt(500)) {
		t.Fatalf("excluded = %v, want 500", got.Excluded)
	}

	eu := regionByName(t, got.Regions, "Europe Developed")
	if !equalDecimal(eu.Value, decimal.NewFromInt(100)) {
		t.Fatalf("Europe Developed value = %v, want 100", eu.Value)
	}
	if !equalDecimal(eu.Weight, decimal.NewFromInt(100)) {
		t.Fatalf("Europe Developed weight = %v, want 100", eu.Weight)
	}

	fin := sectorByName(t, got.Sectors, "Financials")
	if !equalDecimal(fin.Value, decimal.NewFromInt(100)) {
		t.Fatalf("Financials value = %v, want 100", fin.Value)
	}
	if !equalDecimal(fin.Weight, decimal.NewFromInt(100)) {
		t.Fatalf("Financials weight = %v, want 100", fin.Weight)
	}

	// The excluded holdings must leave no trace: no literal "Other" bucket and
	// weights still sum to 100 over the eligible universe.
	for _, r := range got.Regions {
		if r.Region == "Other" && !r.Value.IsZero() {
			t.Fatalf("excluded value leaked into Other region: %v", r.Value)
		}
	}
	regionWeightSum := decimal.Zero
	for _, r := range got.Regions {
		regionWeightSum = regionWeightSum.Add(r.Weight)
	}
	if !equalDecimal(regionWeightSum, decimal.NewFromInt(100)) {
		t.Fatalf("regions weight sum = %v, want 100", regionWeightSum)
	}
	for _, s := range got.Sectors {
		if s.Sector == "Other" && !s.Value.IsZero() {
			t.Fatalf("excluded value leaked into Other sector: %v", s.Value)
		}
	}
	sectorWeightSum := decimal.Zero
	for _, s := range got.Sectors {
		sectorWeightSum = sectorWeightSum.Add(s.Weight)
	}
	if !equalDecimal(sectorWeightSum, decimal.NewFromInt(100)) {
		t.Fatalf("sectors weight sum = %v, want 100", sectorWeightSum)
	}
}

func TestGetDashboardAllocation_ClassAllocation(t *testing.T) {
	eqID := uuid.New()
	bondID := uuid.New()
	ocID := uuid.New()
	noPriceID := uuid.New()
	equity := holding(eqID.String(), "USD", "", "", model.AssetTypeStock, decimal.NewFromInt(5), decimal.NewFromInt(100))
	bond := holding(bondID.String(), "EUR", "", "", model.AssetTypeBond, decimal.NewFromInt(1), decimal.NewFromInt(100))
	bond.AssetClass = "bond"
	unclassified := holding(ocID.String(), "EUR", "", "", model.AssetTypeStock, decimal.NewFromInt(1), decimal.NewFromInt(50))
	unclassified.AssetClass = ""
	noPrice := holding(noPriceID.String(), "USD", "", "", model.AssetTypeStock, decimal.NewFromInt(10), decimal.NewFromInt(100))
	noPrice.HasPrice = false
	pf := &fakePortfolioRepo{
		portfolios: []*model.Portfolio{
			{Currency: "USD"},
			{Currency: "EUR"},
		},
		holdings: []*model.Holding{equity, bond, unclassified, noPrice},
	}
	fx := &fakeFXRepo{rates: map[string]decimal.Decimal{"EUR": decimal.RequireFromString("0.5")}}
	svc := newTestService(t, pf, &fakeExposureRepo{}, fx)

	got, err := svc.GetDashboardAllocation(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// EUR amounts are converted to the USD base (×2); the unpriced holding is
	// skipped and the empty class folds into "other": equity 500, bond 200,
	// other 100 of 800, sorted descending.
	if len(got.Classes) != 3 {
		t.Fatalf("classes = %+v, want 3 buckets", got.Classes)
	}
	if got.Classes[0].Class != "equity" || !equalDecimal(got.Classes[0].Value, decimal.NewFromInt(500)) {
		t.Fatalf("classes[0] = %+v, want equity 500", got.Classes[0])
	}
	assertDecimalInDelta(t, got.Classes[0].Weight, decimal.RequireFromString("62.5"), "0.01", "equity weight")
	if got.Classes[1].Class != "bond" || !equalDecimal(got.Classes[1].Value, decimal.NewFromInt(200)) {
		t.Fatalf("classes[1] = %+v, want bond 200", got.Classes[1])
	}
	assertDecimalInDelta(t, got.Classes[1].Weight, decimal.NewFromInt(25), "0.01", "bond weight")
	if got.Classes[2].Class != "other" || !equalDecimal(got.Classes[2].Value, decimal.NewFromInt(100)) {
		t.Fatalf("classes[2] = %+v, want other 100", got.Classes[2])
	}
	assertDecimalInDelta(t, got.Classes[2].Weight, decimal.RequireFromString("12.5"), "0.01", "other weight")
}

func TestGetDashboardAllocation_ClassAllocationSkipsMissingFX(t *testing.T) {
	eqID := uuid.New()
	jpyID := uuid.New()
	equity := holding(eqID.String(), "USD", "", "", model.AssetTypeStock, decimal.NewFromInt(1), decimal.NewFromInt(100))
	bond := holding(jpyID.String(), "JPY", "", "", model.AssetTypeBond, decimal.NewFromInt(1), decimal.NewFromInt(100))
	bond.AssetClass = "bond"
	pf := &fakePortfolioRepo{
		portfolios: []*model.Portfolio{{Currency: "USD"}},
		holdings:   []*model.Holding{equity, bond},
	}
	fx := &fakeFXRepo{rates: map[string]decimal.Decimal{"EUR": decimal.NewFromInt(1)}}
	svc := newTestService(t, pf, &fakeExposureRepo{}, fx)

	got, err := svc.GetDashboardAllocation(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Classes) != 1 || got.Classes[0].Class != "equity" {
		t.Fatalf("classes = %+v, want only the convertible equity bucket (the JPY holding has no FX rate)", got.Classes)
	}
	assertDecimalInDelta(t, got.Classes[0].Weight, decimal.NewFromInt(100), "0.01", "equity weight")
}

func TestGetDashboardAllocation_CountryExposure(t *testing.T) {
	stockID := uuid.New()
	etfID := uuid.New()
	bondID := uuid.New()
	pf := &fakePortfolioRepo{
		portfolios: []*model.Portfolio{{Currency: "USD"}},
		holdings: []*model.Holding{
			holding(stockID.String(), "USD", "US", "Technology", model.AssetTypeStock, decimal.NewFromInt(1), decimal.NewFromInt(100)),
			holding(etfID.String(), "USD", "", "", model.AssetTypeETF, decimal.NewFromInt(1), decimal.NewFromInt(400)),
			holding(bondID.String(), "USD", "DE", "", model.AssetTypeBond, decimal.NewFromInt(1), decimal.NewFromInt(100)),
		},
	}
	ex := &fakeExposureRepo{
		countries: map[string][]model.ExposureRow{
			etfID.String(): {
				{Name: "US", Weight: decimal.NewFromInt(25)},
				{Name: "JP", Weight: decimal.NewFromInt(75)},
			},
		},
	}
	svc := newTestService(t, pf, ex, &fakeFXRepo{})

	got, err := svc.GetDashboardAllocation(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Equity-only universe: the stock falls back to its own country (US 100)
	// and the ETF splits 25/75 over US/JP (100/300). The DE bond is excluded
	// by nature, so DE must never appear; zero countries are dropped entirely.
	if len(got.Countries) != 2 {
		t.Fatalf("countries = %+v, want only the non-zero JP and US buckets", got.Countries)
	}
	if got.Countries[0].Country != "JP" || !equalDecimal(got.Countries[0].Value, decimal.NewFromInt(300)) {
		t.Fatalf("countries[0] = %+v, want JP 300 (descending)", got.Countries[0])
	}
	assertDecimalInDelta(t, got.Countries[0].Weight, decimal.NewFromInt(60), "0.01", "JP weight")
	if got.Countries[1].Country != "US" || !equalDecimal(got.Countries[1].Value, decimal.NewFromInt(200)) {
		t.Fatalf("countries[1] = %+v, want US 200", got.Countries[1])
	}
	assertDecimalInDelta(t, got.Countries[1].Weight, decimal.NewFromInt(40), "0.01", "US weight")
	countryWeightSum := decimal.Zero
	for _, c := range got.Countries {
		if !c.Value.IsPositive() {
			t.Fatalf("zero-value country bucket leaked: %+v", c)
		}
		countryWeightSum = countryWeightSum.Add(c.Weight)
	}
	if !equalDecimal(countryWeightSum, decimal.NewFromInt(100)) {
		t.Fatalf("countries weight sum = %v, want 100", countryWeightSum)
	}
}

func classByName(t *testing.T, classes []*model.ClassAllocation, name string) *model.ClassAllocation {
	t.Helper()
	for _, c := range classes {
		if c.Class == name {
			return c
		}
	}
	t.Fatalf("class %q not found", name)
	return nil
}

func countryByName(t *testing.T, countries []*model.CountryAllocation, name string) *model.CountryAllocation {
	t.Helper()
	for _, c := range countries {
		if c.Country == name {
			return c
		}
	}
	t.Fatalf("country %q not found", name)
	return nil
}

func drillAsset(t *testing.T, assets []*model.AllocationDrillAsset, assetID string) *model.AllocationDrillAsset {
	t.Helper()
	for _, a := range assets {
		if a.AssetID == assetID {
			return a
		}
	}
	t.Fatalf("asset %s not in drill: %+v", assetID, assets)
	return nil
}

// assertDrillContribution checks the drill contract: contribution =
// value * weight / 100.
func assertDrillContribution(t *testing.T, a *model.AllocationDrillAsset, what string) {
	t.Helper()
	want := a.Value.Mul(a.Weight).Div(decimal.NewFromInt(100))
	if !equalDecimal(a.Contribution, want) {
		t.Fatalf("%s contribution = %v, want %v (value %v * weight %v / 100)", what, a.Contribution, want, a.Value, a.Weight)
	}
}

func TestGetPortfolioAllocationDrill_Class(t *testing.T) {
	stockID := uuid.New()
	etfID := uuid.New()
	bondID := uuid.New()
	pf := &fakePortfolioRepo{
		portfolio: &model.Portfolio{Currency: "EUR"},
		holdings: []*model.Holding{
			holding(stockID.String(), "EUR", "US", "Technology", model.AssetTypeStock, decimal.NewFromInt(2), decimal.NewFromInt(50)),
			holding(etfID.String(), "EUR", "", "", model.AssetTypeETF, decimal.NewFromInt(10), decimal.NewFromInt(100)),
		},
	}
	bond := holding(bondID.String(), "EUR", "", "", model.AssetTypeBond, decimal.NewFromInt(1), decimal.NewFromInt(200))
	bond.AssetClass = "bond"
	pf.holdings = append(pf.holdings, bond)
	svc := newTestService(t, pf, &fakeExposureRepo{}, &fakeFXRepo{})
	portfolioID := uuid.New()

	drill, err := svc.GetPortfolioAllocationDrill(context.Background(), portfolioID, "class", "equity")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if drill.Currency != "EUR" || drill.Dim != "class" || drill.Key != "equity" {
		t.Fatalf("drill = %+v, want class/equity in EUR", drill)
	}
	if len(drill.Assets) != 2 {
		t.Fatalf("assets = %+v, want the stock and the ETF", drill.Assets)
	}
	// Sorted by contribution descending: ETF 1000, stock 100, each at 100%.
	if drill.Assets[0].AssetID != etfID.String() || drill.Assets[1].AssetID != stockID.String() {
		t.Fatalf("assets not sorted by contribution: %+v", drill.Assets)
	}
	for _, a := range drill.Assets {
		if !equalDecimal(a.Weight, decimal.NewFromInt(100)) {
			t.Fatalf("asset %s weight = %v, want 100", a.AssetID, a.Weight)
		}
		assertDrillContribution(t, a, a.AssetID)
	}
	if !equalDecimal(drill.Assets[0].Contribution, decimal.NewFromInt(1000)) {
		t.Fatalf("ETF contribution = %v, want 1000", drill.Assets[0].Contribution)
	}
	// The bucket total must match the class allocation bucket exactly.
	classes, err := svc.GetPortfolioClassAllocation(context.Background(), portfolioID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !equalDecimal(drill.Total, classByName(t, classes.Classes, "equity").Value) {
		t.Fatalf("total = %v, want %v", drill.Total, classByName(t, classes.Classes, "equity").Value)
	}
	if !equalDecimal(drill.Total, decimal.NewFromInt(1100)) {
		t.Fatalf("total = %v, want 1100", drill.Total)
	}

	// The class dimension has no eligibility filter: the bond bucket drills
	// down to the bond holding (unlike the exposure dimensions).
	bondDrill, err := svc.GetPortfolioAllocationDrill(context.Background(), portfolioID, "class", "bond")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(bondDrill.Assets) != 1 || bondDrill.Assets[0].AssetID != bondID.String() {
		t.Fatalf("bond drill = %+v, want only the bond asset", bondDrill.Assets)
	}
	if !equalDecimal(bondDrill.Total, classByName(t, classes.Classes, "bond").Value) {
		t.Fatalf("bond total = %v, want %v", bondDrill.Total, classByName(t, classes.Classes, "bond").Value)
	}

	// An absent bucket is an empty (non-nil) result with a zero total.
	empty, err := svc.GetPortfolioAllocationDrill(context.Background(), portfolioID, "class", "crypto")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if empty.Assets == nil || len(empty.Assets) != 0 || !empty.Total.IsZero() {
		t.Fatalf("empty drill = %+v, want non-nil empty assets and zero total", empty)
	}
}

func TestGetPortfolioAllocationDrill_Country(t *testing.T) {
	stockID := uuid.New()
	etfID := uuid.New()
	bondID := uuid.New()
	pf := &fakePortfolioRepo{
		portfolio: &model.Portfolio{Currency: "EUR"},
		holdings: []*model.Holding{
			holding(stockID.String(), "EUR", "US", "Technology", model.AssetTypeStock, decimal.NewFromInt(1), decimal.NewFromInt(100)),
			holding(etfID.String(), "EUR", "", "", model.AssetTypeETF, decimal.NewFromInt(1), decimal.NewFromInt(400)),
			holding(bondID.String(), "EUR", "DE", "", model.AssetTypeBond, decimal.NewFromInt(1), decimal.NewFromInt(100)),
		},
	}
	ex := &fakeExposureRepo{
		countries: map[string][]model.ExposureRow{
			etfID.String(): {
				{Name: "US", Weight: decimal.NewFromInt(25)},
				{Name: "JP", Weight: decimal.NewFromInt(75)},
			},
		},
	}
	svc := newTestService(t, pf, ex, &fakeFXRepo{})
	portfolioID := uuid.New()

	us, err := svc.GetPortfolioAllocationDrill(context.Background(), portfolioID, "country", "US")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The stock falls back to its domicile (US 100) and the ETF contributes
	// 25% of 400; the DE bond is ineligible and never shows up.
	// The stock falls back to its domicile (US 100) and the ETF contributes
	// 25% of 400; the DE bond is ineligible and never shows up (the len
	// check above already pins the drill down to the two equity assets).
	if len(us.Assets) != 2 {
		t.Fatalf("US drill = %+v, want stock + ETF only", us.Assets)
	}
	stockEntry := drillAsset(t, us.Assets, stockID.String())
	if !equalDecimal(stockEntry.Weight, decimal.NewFromInt(100)) || !equalDecimal(stockEntry.Contribution, decimal.NewFromInt(100)) {
		t.Fatalf("stock entry = %+v, want weight 100 contribution 100", stockEntry)
	}
	etfEntry := drillAsset(t, us.Assets, etfID.String())
	if !equalDecimal(etfEntry.Weight, decimal.NewFromInt(25)) || !equalDecimal(etfEntry.Contribution, decimal.NewFromInt(100)) {
		t.Fatalf("ETF entry = %+v, want weight 25 contribution 100", etfEntry)
	}
	for _, a := range us.Assets {
		assertDrillContribution(t, a, a.AssetID)
	}

	jp, err := svc.GetPortfolioAllocationDrill(context.Background(), portfolioID, "country", "JP")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(jp.Assets) != 1 || jp.Assets[0].AssetID != etfID.String() {
		t.Fatalf("JP drill = %+v, want only the ETF", jp.Assets)
	}
	assertDrillContribution(t, jp.Assets[0], "JP ETF")

	// Totals must match the country buckets of the geography allocation.
	geoAlloc, err := svc.GetPortfolioGeographyAllocation(context.Background(), portfolioID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !equalDecimal(us.Total, countryByName(t, geoAlloc.Countries, "US").Value) {
		t.Fatalf("US total = %v, want %v", us.Total, countryByName(t, geoAlloc.Countries, "US").Value)
	}
	if !equalDecimal(jp.Total, countryByName(t, geoAlloc.Countries, "JP").Value) {
		t.Fatalf("JP total = %v, want %v", jp.Total, countryByName(t, geoAlloc.Countries, "JP").Value)
	}
	for _, c := range geoAlloc.Countries {
		if c.Country == "DE" {
			t.Fatalf("DE bucket unexpectedly present in the geography allocation: %+v", c)
		}
	}
	de, err := svc.GetPortfolioAllocationDrill(context.Background(), portfolioID, "country", "DE")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if de.Assets == nil || len(de.Assets) != 0 || !de.Total.IsZero() {
		t.Fatalf("DE drill = %+v, want non-nil empty assets and zero total", de)
	}
}

func TestGetPortfolioAllocationDrill_Region(t *testing.T) {
	etfID := uuid.New()
	jpyID := uuid.New()
	bondID := uuid.New()
	pf := &fakePortfolioRepo{
		portfolio: &model.Portfolio{Currency: "USD"},
		holdings: []*model.Holding{
			holding(etfID.String(), "USD", "", "", model.AssetTypeETF, decimal.NewFromInt(10), decimal.NewFromInt(100)),
			// No JPY rate in the fixture: not convertible, excluded like in
			// the aggregation. The bond is ineligible, excluded too.
			holding(jpyID.String(), "JPY", "JP", "Technology", model.AssetTypeStock, decimal.NewFromInt(1), decimal.NewFromInt(100)),
			holding(bondID.String(), "USD", "US", "", model.AssetTypeBond, decimal.NewFromInt(1), decimal.NewFromInt(500)),
		},
	}
	ex := &fakeExposureRepo{
		regions: map[string][]model.ExposureRow{
			etfID.String(): {
				{Name: "North America", Weight: decimal.NewFromInt(60)},
				{Name: "Europe Developed", Weight: decimal.NewFromInt(40)},
			},
		},
	}
	fx := &fakeFXRepo{rates: map[string]decimal.Decimal{"USD": decimal.NewFromInt(1)}}
	svc := newTestService(t, pf, ex, fx)
	portfolioID := uuid.New()

	na, err := svc.GetPortfolioAllocationDrill(context.Background(), portfolioID, "region", "North America")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(na.Assets) != 1 || na.Assets[0].AssetID != etfID.String() {
		t.Fatalf("North America drill = %+v, want only the ETF", na.Assets)
	}
	entry := na.Assets[0]
	if !equalDecimal(entry.Value, decimal.NewFromInt(1000)) || !equalDecimal(entry.Weight, decimal.NewFromInt(60)) {
		t.Fatalf("ETF entry = %+v, want value 1000 weight 60", entry)
	}
	assertDrillContribution(t, entry, "North America ETF")
	if !equalDecimal(entry.Contribution, decimal.NewFromInt(600)) {
		t.Fatalf("ETF contribution = %v, want 600", entry.Contribution)
	}
	if !equalDecimal(na.Total, decimal.NewFromInt(600)) {
		t.Fatalf("total = %v, want 600", na.Total)
	}

	// Key with spaces arrives verbatim and must match the region bucket.
	eu, err := svc.GetPortfolioAllocationDrill(context.Background(), portfolioID, "region", "Europe Developed")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(eu.Assets) != 1 || !equalDecimal(eu.Assets[0].Contribution, decimal.NewFromInt(400)) {
		t.Fatalf("Europe Developed drill = %+v, want ETF 400", eu.Assets)
	}

	geoAlloc, err := svc.GetPortfolioGeographyAllocation(context.Background(), portfolioID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !equalDecimal(na.Total, regionByName(t, geoAlloc.Regions, "North America").Value) {
		t.Fatalf("North America total = %v, want %v", na.Total, regionByName(t, geoAlloc.Regions, "North America").Value)
	}
	if !equalDecimal(eu.Total, regionByName(t, geoAlloc.Regions, "Europe Developed").Value) {
		t.Fatalf("Europe Developed total = %v, want %v", eu.Total, regionByName(t, geoAlloc.Regions, "Europe Developed").Value)
	}
}

func TestGetPortfolioAllocationDrill_Sector(t *testing.T) {
	stockID := uuid.New()
	etfID := uuid.New()
	pf := &fakePortfolioRepo{
		portfolio: &model.Portfolio{Currency: "USD"},
		holdings: []*model.Holding{
			holding(stockID.String(), "USD", "", "Technology", model.AssetTypeStock, decimal.NewFromInt(10), decimal.NewFromInt(100)),
			holding(etfID.String(), "USD", "", "", model.AssetTypeETF, decimal.NewFromInt(10), decimal.NewFromInt(100)),
		},
	}
	ex := &fakeExposureRepo{
		sectors: map[string][]model.ExposureRow{
			etfID.String(): {
				{Name: "Information Technology", Weight: decimal.NewFromInt(40)},
				{Name: "Energy", Weight: decimal.NewFromInt(10)},
			},
		},
	}
	svc := newTestService(t, pf, ex, &fakeFXRepo{})
	portfolioID := uuid.New()

	// The stock has no stored rows: "Technology" normalizes to
	// "Information Technology" and defaults it to 100%; the ETF contributes
	// 40% of its value.
	it, err := svc.GetPortfolioAllocationDrill(context.Background(), portfolioID, "sector", "Information Technology")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(it.Assets) != 2 {
		t.Fatalf("IT drill = %+v, want stock + ETF", it.Assets)
	}
	stockEntry := drillAsset(t, it.Assets, stockID.String())
	if !equalDecimal(stockEntry.Weight, decimal.NewFromInt(100)) || !equalDecimal(stockEntry.Contribution, decimal.NewFromInt(1000)) {
		t.Fatalf("stock entry = %+v, want weight 100 contribution 1000", stockEntry)
	}
	etfEntry := drillAsset(t, it.Assets, etfID.String())
	if !equalDecimal(etfEntry.Weight, decimal.NewFromInt(40)) || !equalDecimal(etfEntry.Contribution, decimal.NewFromInt(400)) {
		t.Fatalf("ETF entry = %+v, want weight 40 contribution 400", etfEntry)
	}
	for _, a := range it.Assets {
		assertDrillContribution(t, a, a.AssetID)
	}

	// The partial ETF mapping (40+10) leaves its residual out of every
	// bucket; the sector drill must mirror that per bucket.
	energy, err := svc.GetPortfolioAllocationDrill(context.Background(), portfolioID, "sector", "Energy")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(energy.Assets) != 1 || energy.Assets[0].AssetID != etfID.String() || !equalDecimal(energy.Total, decimal.NewFromInt(100)) {
		t.Fatalf("Energy drill = %+v, want ETF 100", energy.Assets)
	}

	secAlloc, err := svc.GetPortfolioSectorAllocation(context.Background(), portfolioID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !equalDecimal(it.Total, sectorByName(t, secAlloc.Sectors, "Information Technology").Value) {
		t.Fatalf("IT total = %v, want %v", it.Total, sectorByName(t, secAlloc.Sectors, "Information Technology").Value)
	}
	if !equalDecimal(energy.Total, sectorByName(t, secAlloc.Sectors, "Energy").Value) {
		t.Fatalf("Energy total = %v, want %v", energy.Total, sectorByName(t, secAlloc.Sectors, "Energy").Value)
	}
}

func TestGetPortfolioAllocationDrill_OtherBucket(t *testing.T) {
	etfID := uuid.New()
	bondID := uuid.New()
	pf := &fakePortfolioRepo{
		portfolio: &model.Portfolio{Currency: "USD"},
		holdings: []*model.Holding{
			holding(etfID.String(), "USD", "", "", model.AssetTypeETF, decimal.NewFromInt(10), decimal.NewFromInt(100)),
			holding(bondID.String(), "USD", "", "", model.AssetTypeBond, decimal.NewFromInt(1), decimal.NewFromInt(500)),
		},
	}
	// Eligible equity ETF with no stored rows and no country/sector: its
	// whole value falls into the literal "Other" bucket of the region and
	// sector dimensions, while the ineligible bond never appears.
	svc := newTestService(t, pf, &fakeExposureRepo{}, &fakeFXRepo{})
	portfolioID := uuid.New()

	for _, dim := range []string{"region", "sector", "country"} {
		drill, err := svc.GetPortfolioAllocationDrill(context.Background(), portfolioID, dim, "Other")
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", dim, err)
		}
		if len(drill.Assets) != 1 || drill.Assets[0].AssetID != etfID.String() {
			t.Fatalf("%s: Other drill = %+v, want only the ETF", dim, drill.Assets)
		}
		entry := drill.Assets[0]
		if !equalDecimal(entry.Weight, decimal.NewFromInt(100)) || !equalDecimal(entry.Contribution, decimal.NewFromInt(1000)) {
			t.Fatalf("%s: Other entry = %+v, want weight 100 contribution 1000", dim, entry)
		}
		assertDrillContribution(t, entry, dim+" Other")
		if !equalDecimal(drill.Total, decimal.NewFromInt(1000)) {
			t.Fatalf("%s: Other total = %v, want 1000", dim, drill.Total)
		}
	}

	geoAlloc, err := svc.GetPortfolioGeographyAllocation(context.Background(), portfolioID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	other, err := svc.GetPortfolioSectorAllocation(context.Background(), portfolioID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	regionOther := regionByName(t, geoAlloc.Regions, "Other")
	if !equalDecimal(regionOther.Value, decimal.NewFromInt(1000)) {
		t.Fatalf("geography Other region value = %v, want 1000", regionOther.Value)
	}
	sectorOther := sectorByName(t, other.Sectors, "Other")
	if !equalDecimal(sectorOther.Value, decimal.NewFromInt(1000)) {
		t.Fatalf("sector allocation Other value = %v, want 1000", sectorOther.Value)
	}

	// A mapped bucket on the unmapped asset is empty.
	na, err := svc.GetPortfolioAllocationDrill(context.Background(), portfolioID, "region", "North America")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if na.Assets == nil || len(na.Assets) != 0 || !na.Total.IsZero() {
		t.Fatalf("North America drill = %+v, want non-nil empty assets and zero total", na)
	}
}

func TestGetDashboardAllocationDrill_AggregatesAcrossPortfolios(t *testing.T) {
	etfID := uuid.New()
	stockID := uuid.New()
	pf := &fakePortfolioRepo{
		portfolios: []*model.Portfolio{
			{Currency: "USD"},
			{Currency: "USD"},
		},
		holdings: []*model.Holding{
			holding(etfID.String(), "USD", "", "", model.AssetTypeETF, decimal.NewFromInt(1), decimal.NewFromInt(100)),
			holding(etfID.String(), "USD", "", "", model.AssetTypeETF, decimal.NewFromInt(4), decimal.NewFromInt(100)),
			holding(stockID.String(), "USD", "US", "Technology", model.AssetTypeStock, decimal.NewFromInt(1), decimal.NewFromInt(100)),
		},
	}
	ex := &fakeExposureRepo{
		regions: map[string][]model.ExposureRow{
			etfID.String(): {{Name: "North America", Weight: decimal.NewFromInt(60)}},
		},
	}
	svc := newTestService(t, pf, ex, &fakeFXRepo{})
	userID := uuid.New()

	drill, err := svc.GetDashboardAllocationDrill(context.Background(), userID, "region", "North America")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if drill.Currency != "USD" || drill.Dim != "region" || drill.Key != "North America" {
		t.Fatalf("drill = %+v, want region/North America in USD", drill)
	}
	// The ETF held in both portfolios merges into one entry: value 500 at
	// 60% (contribution 300), next to the US stock at 100 (contribution
	// 100), sorted descending.
	if len(drill.Assets) != 2 {
		t.Fatalf("assets = %+v, want the merged ETF and the stock", drill.Assets)
	}
	if drill.Assets[0].AssetID != etfID.String() {
		t.Fatalf("assets[0] = %+v, want the ETF first", drill.Assets[0])
	}
	etfEntry := drill.Assets[0]
	if !equalDecimal(etfEntry.Value, decimal.NewFromInt(500)) || !equalDecimal(etfEntry.Weight, decimal.NewFromInt(60)) || !equalDecimal(etfEntry.Contribution, decimal.NewFromInt(300)) {
		t.Fatalf("ETF entry = %+v, want value 500 weight 60 contribution 300", etfEntry)
	}
	for _, a := range drill.Assets {
		assertDrillContribution(t, a, a.AssetID)
	}

	dashAlloc, err := svc.GetDashboardAllocation(context.Background(), userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !equalDecimal(drill.Total, regionByName(t, dashAlloc.Regions, "North America").Value) {
		t.Fatalf("total = %v, want %v", drill.Total, regionByName(t, dashAlloc.Regions, "North America").Value)
	}

	// The class dimension aggregates across portfolios as well.
	equity, err := svc.GetDashboardAllocationDrill(context.Background(), userID, "class", "equity")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(equity.Assets) != 2 || !equalDecimal(equity.Total, classByName(t, dashAlloc.Classes, "equity").Value) {
		t.Fatalf("equity drill = %+v, want 2 assets totalling %v", equity.Assets, classByName(t, dashAlloc.Classes, "equity").Value)
	}
}

func TestAllocationDrill_InvalidInput(t *testing.T) {
	pf := &fakePortfolioRepo{
		portfolio:  &model.Portfolio{Currency: "USD"},
		portfolios: []*model.Portfolio{{Currency: "USD"}},
	}
	svc := newTestService(t, pf, &fakeExposureRepo{}, &fakeFXRepo{})

	cases := []struct {
		dim string
		key string
	}{
		{dim: "asset", key: "US"},
		{dim: "Region", key: "North America"},
		{dim: "", key: "US"},
		{dim: "region", key: ""},
		{dim: "class", key: ""},
	}
	for _, tc := range cases {
		if _, err := svc.GetPortfolioAllocationDrill(context.Background(), uuid.New(), tc.dim, tc.key); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("portfolio drill dim=%q key=%q: err = %v, want ErrInvalidInput", tc.dim, tc.key, err)
		}
		if _, err := svc.GetDashboardAllocationDrill(context.Background(), uuid.New(), tc.dim, tc.key); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("dashboard drill dim=%q key=%q: err = %v, want ErrInvalidInput", tc.dim, tc.key, err)
		}
	}
}

func TestNormalizeCountries(t *testing.T) {
	cases := []struct {
		name string
		in   []model.ExposureRow
		want []model.ExposureRow
	}{
		{
			name: "ISO codes and full names normalize to canonical codes",
			in: []model.ExposureRow{
				{Name: "United States", Weight: decimal.NewFromInt(60)},
				{Name: "DE", Weight: decimal.NewFromInt(40)},
			},
			want: []model.ExposureRow{
				{Name: "US", Weight: decimal.NewFromInt(60)},
				{Name: "DE", Weight: decimal.NewFromInt(40)},
			},
		},
		{
			name: "non-canonical country names are dropped",
			in: []model.ExposureRow{
				{Name: "Atlantis", Weight: decimal.NewFromInt(50)},
				{Name: "IT", Weight: decimal.NewFromInt(50)},
			},
			want: []model.ExposureRow{
				{Name: "IT", Weight: decimal.NewFromInt(50)},
			},
		},
		{
			name: "empty and non-positive rows are dropped",
			in: []model.ExposureRow{
				{Name: "", Weight: decimal.NewFromInt(50)},
				{Name: "JP", Weight: decimal.NewFromInt(0)},
				{Name: "US", Weight: decimal.NewFromInt(100)},
			},
			want: []model.ExposureRow{
				{Name: "US", Weight: decimal.NewFromInt(100)},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeCountries(tc.in)
			if len(got) != len(tc.want) {
				t.Fatalf("len = %d, want %d: %+v", len(got), len(tc.want), got)
			}
			for i := range got {
				if got[i].Name != tc.want[i].Name || !got[i].Weight.Equal(tc.want[i].Weight) {
					t.Fatalf("row[%d] = %+v, want %+v", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestIsCanonicalCountry(t *testing.T) {
	if !isCanonicalCountry("US") {
		t.Fatal("US should be canonical")
	}
	if !isCanonicalCountry("ZA") {
		t.Fatal("ZA should be canonical")
	}
	if isCanonicalCountry("") {
		t.Fatal("empty should not be canonical")
	}
	if isCanonicalCountry("XYZ") {
		t.Fatal("XYZ should not be canonical")
	}
}

func TestDeriveRegions(t *testing.T) {
	assetID := uuid.New()
	svc := newTestServiceWithAsset(t, &fakeAssetRepo{asset: &model.Asset{ID: assetID, Type: model.AssetTypeETF}})

	regions, err := svc.DeriveRegions(context.Background(), assetID, []model.ExposureRow{
		{Name: "US", Weight: decimal.NewFromInt(60)},
		{Name: "GB", Weight: decimal.NewFromInt(25)},
		{Name: "JP", Weight: decimal.NewFromInt(15)},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(regions) != len(geo.Regions) {
		t.Fatalf("regions len = %d, want %d", len(regions), len(geo.Regions))
	}
	for i, name := range geo.Regions {
		if regions[i].Name != name {
			t.Fatalf("regions[%d] = %q, want %q (canonical order)", i, regions[i].Name, name)
		}
	}
	want := map[string]string{
		"North America":  "60",
		"United Kingdom": "25",
		"Japan":          "15",
	}
	for _, r := range regions {
		if w, ok := want[r.Name]; ok && !r.Weight.Equal(decimal.RequireFromString(w)) {
			t.Fatalf("region %q weight = %v, want %s", r.Name, r.Weight, w)
		}
	}
}

func TestDeriveRegions_AssetNotFound(t *testing.T) {
	svc := newTestServiceWithAsset(t, &fakeAssetRepo{})
	_, err := svc.DeriveRegions(context.Background(), uuid.New(), nil)
	if !errors.Is(err, ErrAssetNotFound) {
		t.Fatalf("err = %v, want ErrAssetNotFound", err)
	}
}

func TestDeriveRegions_EmptyCountries(t *testing.T) {
	assetID := uuid.New()
	svc := newTestServiceWithAsset(t, &fakeAssetRepo{asset: &model.Asset{ID: assetID, Type: model.AssetTypeETF}})

	regions, err := svc.DeriveRegions(context.Background(), assetID, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(regions) != len(geo.Regions) {
		t.Fatalf("regions len = %d, want %d", len(regions), len(geo.Regions))
	}
	for _, r := range regions {
		if !r.Weight.IsZero() {
			t.Fatalf("region %q weight = %v, want zero", r.Name, r.Weight)
		}
	}
}

func exposureRowsByName(rows []model.ExposureRow) map[string]decimal.Decimal {
	byName := map[string]decimal.Decimal{}
	for _, r := range rows {
		byName[r.Name] = r.Weight
	}
	return byName
}

func TestSaveExposureDimensions_CountriesDoNotTouchRegions(t *testing.T) {
	assetID := uuid.New()
	key := assetID.String()
	ex := &fakeExposureRepo{
		regions: map[string][]model.ExposureRow{
			key: {{Name: "North America", Weight: decimal.NewFromInt(100)}},
		},
	}
	repos := &repository.Repository{Exposure: ex}

	err := saveExposureDimensions(context.Background(), repos, assetID, &model.AssetExposure{
		Countries: []model.ExposureRow{
			{Name: "United States", Weight: decimal.NewFromInt(60)},
			{Name: "JP", Weight: decimal.NewFromInt(35)},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ex.replaceRegionsCalls != 0 {
		t.Fatalf("ReplaceRegions calls = %d, want 0 (saving countries must not rewrite the regions dimension)", ex.replaceRegionsCalls)
	}
	gotRegions := ex.regions[key]
	if len(gotRegions) != 1 || gotRegions[0].Name != "North America" || !gotRegions[0].Weight.Equal(decimal.NewFromInt(100)) {
		t.Fatalf("stored regions = %+v, want the pre-existing [North America 100]", gotRegions)
	}
	gotCountries := exposureRowsByName(ex.countries[key])
	if len(gotCountries) != 2 || !gotCountries["US"].Equal(decimal.NewFromInt(60)) || !gotCountries["JP"].Equal(decimal.NewFromInt(35)) {
		t.Fatalf("stored countries = %+v, want normalized US 60 + JP 35", ex.countries[key])
	}
	if _, ok := ex.sectors[key]; ok {
		t.Fatalf("sectors written for an absent dimension: %+v", ex.sectors[key])
	}
}

func TestSaveExposureDimensions_ExplicitRegionsStillSaved(t *testing.T) {
	assetID := uuid.New()
	key := assetID.String()
	ex := &fakeExposureRepo{
		regions: map[string][]model.ExposureRow{
			key: {{Name: "North America", Weight: decimal.NewFromInt(100)}},
		},
	}
	repos := &repository.Repository{Exposure: ex}

	err := saveExposureDimensions(context.Background(), repos, assetID, &model.AssetExposure{
		Regions: []model.ExposureRow{
			{Name: "North America", Weight: decimal.NewFromInt(60)},
			{Name: "United Kingdom", Weight: decimal.NewFromInt(25)},
			{Name: "Japan", Weight: decimal.NewFromInt(10)},
		},
		Countries: []model.ExposureRow{
			{Name: "US", Weight: decimal.NewFromInt(60)},
			{Name: "GB", Weight: decimal.NewFromInt(25)},
			{Name: "JP", Weight: decimal.NewFromInt(10)},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ex.replaceRegionsCalls != 1 {
		t.Fatalf("ReplaceRegions calls = %d, want 1 (explicit regions must persist)", ex.replaceRegionsCalls)
	}
	gotRegions := exposureRowsByName(ex.regions[key])
	if !gotRegions["North America"].Equal(decimal.NewFromInt(60)) || !gotRegions["United Kingdom"].Equal(decimal.NewFromInt(25)) ||
		!gotRegions["Japan"].Equal(decimal.NewFromInt(10)) {
		t.Fatalf("stored regions = %+v, want the explicit 60/25/10", ex.regions[key])
	}
	if !gotRegions[geo.OtherRegion].Equal(decimal.NewFromInt(5)) {
		t.Fatalf("residual %q = %v, want 5 (prepareRegions invariant)", geo.OtherRegion, gotRegions[geo.OtherRegion])
	}
	if gotCountries := exposureRowsByName(ex.countries[key]); len(gotCountries) != 3 || !gotCountries["GB"].Equal(decimal.NewFromInt(25)) {
		t.Fatalf("stored countries = %+v, want US/GB/JP 60/25/10", ex.countries[key])
	}
}

func TestSaveExposureDimensions_EmptyPayloadWritesNothing(t *testing.T) {
	assetID := uuid.New()
	key := assetID.String()
	ex := &fakeExposureRepo{
		regions:   map[string][]model.ExposureRow{key: {{Name: "Japan", Weight: decimal.NewFromInt(100)}}},
		sectors:   map[string][]model.ExposureRow{key: {{Name: "Health Care", Weight: decimal.NewFromInt(100)}}},
		countries: map[string][]model.ExposureRow{key: {{Name: "JP", Weight: decimal.NewFromInt(100)}}},
	}
	repos := &repository.Repository{Exposure: ex}

	if err := saveExposureDimensions(context.Background(), repos, assetID, &model.AssetExposure{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ex.replaceRegionsCalls != 0 {
		t.Fatalf("ReplaceRegions calls = %d, want 0", ex.replaceRegionsCalls)
	}
	if got := exposureRowsByName(ex.regions[key]); !got["Japan"].Equal(decimal.NewFromInt(100)) || len(got) != 1 {
		t.Fatalf("stored regions = %+v, want the pre-existing [Japan 100]", ex.regions[key])
	}
	if got := exposureRowsByName(ex.countries[key]); !got["JP"].Equal(decimal.NewFromInt(100)) || len(got) != 1 {
		t.Fatalf("stored countries = %+v, want the pre-existing [JP 100]", ex.countries[key])
	}
	if ex.setProvenanceCalls != 0 {
		t.Fatalf("SetProvenance calls = %d, want 0 (an empty payload must not write provenance)", ex.setProvenanceCalls)
	}
}

func TestSaveExposureDimensions_RecordsProvenance(t *testing.T) {
	assetID := uuid.New()
	key := assetID.String()
	ex := &fakeExposureRepo{
		provenance: map[string]map[string]model.ExposureProvenance{
			key: {
				model.ExposureDimensionRegions: {Source: "yahoo", UpdatedAt: time.Now().Add(-24 * time.Hour).UTC()},
			},
		},
	}
	repos := &repository.Repository{Exposure: ex}

	err := saveExposureDimensions(context.Background(), repos, assetID, &model.AssetExposure{
		Countries:       []model.ExposureRow{{Name: "US", Weight: decimal.NewFromInt(100)}},
		CountriesSource: "morningstar",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ex.setProvenanceCalls != 1 {
		t.Fatalf("SetProvenance calls = %d, want 1 (only the saved dimension)", ex.setProvenanceCalls)
	}
	got := ex.provenance[key]
	if got[model.ExposureDimensionCountries].Source != "morningstar" {
		t.Fatalf("countries provenance = %+v, want source morningstar", got[model.ExposureDimensionCountries])
	}
	if got[model.ExposureDimensionCountries].UpdatedAt.IsZero() {
		t.Fatalf("countries provenance updated_at is zero, want a timestamp")
	}
	if got[model.ExposureDimensionRegions].Source != "yahoo" {
		t.Fatalf("regions provenance = %+v, want the pre-existing yahoo (saving countries must not touch it)", got[model.ExposureDimensionRegions])
	}
	if _, ok := got[model.ExposureDimensionSectors]; ok {
		t.Fatalf("sectors provenance written for an absent dimension: %+v", got[model.ExposureDimensionSectors])
	}
}

func TestSaveExposureDimensions_DefaultsManualSource(t *testing.T) {
	assetID := uuid.New()
	key := assetID.String()
	ex := &fakeExposureRepo{}
	repos := &repository.Repository{Exposure: ex}

	err := saveExposureDimensions(context.Background(), repos, assetID, &model.AssetExposure{
		Regions: []model.ExposureRow{{Name: "North America", Weight: decimal.NewFromInt(100)}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := ex.provenance[key][model.ExposureDimensionRegions].Source; got != "manual" {
		t.Fatalf("regions provenance = %q, want manual (absent source means a manual edit)", got)
	}

	err = saveExposureDimensions(context.Background(), repos, assetID, &model.AssetExposure{
		Sectors:       []model.ExposureRow{{Name: "Technology", Weight: decimal.NewFromInt(100)}},
		SectorsSource: "justetf",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := ex.provenance[key][model.ExposureDimensionSectors].Source; got != "justetf" {
		t.Fatalf("sectors provenance = %q, want justetf (explicit source must win)", got)
	}
}

func TestGetAssetExposure_ReturnsStoredProvenance(t *testing.T) {
	assetID := uuid.New()
	key := assetID.String()
	updatedAt := time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)
	a := &fakeAssetRepo{asset: &model.Asset{ID: assetID, Ticker: "SXR8.DE", Type: model.AssetTypeETF, ISIN: "IE00B4L5Y983"}}
	ex := &fakeExposureRepo{
		countries: map[string][]model.ExposureRow{key: {{Name: "US", Weight: decimal.NewFromInt(100)}}},
		provenance: map[string]map[string]model.ExposureProvenance{
			key: {
				model.ExposureDimensionCountries: {Source: "morningstar", UpdatedAt: updatedAt},
			},
		},
	}
	svc := newFetchTestService(t, a, ex, nil, nil, nil)

	got, err := svc.GetAssetExposure(context.Background(), assetID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Provenance) != 1 {
		t.Fatalf("provenance = %+v, want only the stored countries entry", got.Provenance)
	}
	prov, ok := got.Provenance[model.ExposureDimensionCountries]
	if !ok || prov.Source != "morningstar" || !prov.UpdatedAt.Equal(updatedAt) {
		t.Fatalf("countries provenance = %+v, want morningstar @ %s", prov, updatedAt)
	}
}

func TestMapMorningstarExposure_UsesOfficialRegions(t *testing.T) {
	raw := &model.AssetExposure{
		Countries: []model.ExposureRow{
			{Name: "United States", Weight: decimal.NewFromInt(60)},
		},
		Regions: []model.ExposureRow{
			{Name: "North America", Weight: decimal.NewFromInt(60)},
			{Name: "United Kingdom", Weight: decimal.NewFromInt(25)},
			{Name: "Japan", Weight: decimal.NewFromInt(15)},
		},
		Sectors: []model.ExposureRow{
			{Name: "Technology", Weight: decimal.NewFromFloat(27.5)},
		},
	}
	mapped := mapMorningstarExposure(raw)

	byName := map[string]decimal.Decimal{}
	if len(mapped.Regions) != len(geo.Regions) {
		t.Fatalf("regions len = %d, want %d", len(mapped.Regions), len(geo.Regions))
	}
	for i, name := range geo.Regions {
		if mapped.Regions[i].Name != name {
			t.Fatalf("regions[%d] = %q, want %q (canonical order)", i, mapped.Regions[i].Name, name)
		}
		byName[name] = mapped.Regions[i].Weight
	}
	if !byName["North America"].Equal(decimal.NewFromInt(60)) {
		t.Fatalf("North America = %v, want 60", byName["North America"])
	}
	if !byName["United Kingdom"].Equal(decimal.NewFromInt(25)) {
		t.Fatalf("United Kingdom = %v, want 25", byName["United Kingdom"])
	}
	if !byName["Japan"].Equal(decimal.NewFromInt(15)) {
		t.Fatalf("Japan = %v, want 15", byName["Japan"])
	}
	if !byName["Asia Emerging"].IsZero() {
		t.Fatalf("Asia Emerging = %v, want zero (zero-filled)", byName["Asia Emerging"])
	}
	// Countries are normalized and sectors aggregated as before.
	if len(mapped.Countries) != 1 || mapped.Countries[0].Name != "US" ||
		!mapped.Countries[0].Weight.Equal(decimal.NewFromInt(60)) {
		t.Fatalf("countries = %+v, want [US 60]", mapped.Countries)
	}
	if len(mapped.Sectors) != 1 || mapped.Sectors[0].Name != "Information Technology" ||
		!mapped.Sectors[0].Weight.Equal(decimal.NewFromFloat(27.5)) {
		t.Fatalf("sectors = %+v, want [Information Technology 27.5]", mapped.Sectors)
	}
}

func TestMapMorningstarExposure_FallsBackToDerivation(t *testing.T) {
	raw := &model.AssetExposure{
		Countries: []model.ExposureRow{
			{Name: "US", Weight: decimal.NewFromInt(60)},
			{Name: "JP", Weight: decimal.NewFromInt(40)},
		},
	}
	mapped := mapMorningstarExposure(raw)

	byName := map[string]decimal.Decimal{}
	if len(mapped.Regions) != len(geo.Regions) {
		t.Fatalf("regions len = %d, want %d", len(mapped.Regions), len(geo.Regions))
	}
	for i, name := range geo.Regions {
		if mapped.Regions[i].Name != name {
			t.Fatalf("regions[%d] = %q, want %q (canonical order)", i, mapped.Regions[i].Name, name)
		}
		byName[name] = mapped.Regions[i].Weight
	}
	if !byName["North America"].Equal(decimal.NewFromInt(60)) {
		t.Fatalf("North America = %v, want 60", byName["North America"])
	}
	if !byName["Japan"].Equal(decimal.NewFromInt(40)) {
		t.Fatalf("Japan = %v, want 40", byName["Japan"])
	}
	if !byName["United Kingdom"].IsZero() {
		t.Fatalf("United Kingdom = %v, want zero", byName["United Kingdom"])
	}
}

func TestValidateExposureWeights(t *testing.T) {
	cases := []struct {
		name    string
		sum     string
		rule    weightSumRule
		wantErr bool
	}{
		{"exact: 100 is valid", "100", weightSumExact100, false},
		{"exact: 100.4 is within tolerance", "100.4", weightSumExact100, false},
		{"exact: 99.5 is within tolerance", "99.5", weightSumExact100, false},
		{"exact: 100.6 is rejected", "100.6", weightSumExact100, true},
		{"exact: 99.4 is rejected", "99.4", weightSumExact100, true},
		{"exact: 103 is rejected", "103", weightSumExact100, true},
		{"max: 100 is valid", "100", weightSumMax100, false},
		{"max: 100.5 is the accepted boundary", "100.5", weightSumMax100, false},
		{"max: 100.6 is rejected", "100.6", weightSumMax100, true},
		{"max: 103 is rejected", "103", weightSumMax100, true},
		{"max: a partial 95 is valid", "95", weightSumMax100, false},
		{"max: zero rows are valid", "0", weightSumMax100, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rows := []model.ExposureRow{{Name: "X", Weight: decimal.RequireFromString(tc.sum)}}
			err := validateExposureWeights(rows, tc.rule)
			if tc.wantErr {
				if !errors.Is(err, ErrInvalidWeights) {
					t.Fatalf("err = %v, want ErrInvalidWeights", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestPrepareRegions(t *testing.T) {
	cases := []struct {
		name    string
		in      []model.ExposureRow
		wantErr bool
		want    map[string]string
		wantSum string
	}{
		{
			name: "sum of 95 injects the residual into Other so the stored total is 100",
			in: []model.ExposureRow{
				{Name: "North America", Weight: decimal.NewFromInt(60)},
				{Name: "United Kingdom", Weight: decimal.NewFromInt(25)},
				{Name: "Japan", Weight: decimal.NewFromInt(10)},
			},
			want: map[string]string{
				"North America":          "60",
				"United Kingdom":         "25",
				"Japan":                  "10",
				"Other / Not Classified": "5",
			},
			wantSum: "100",
		},
		{
			name: "sum of 103 is rejected",
			in: []model.ExposureRow{
				{Name: "North America", Weight: decimal.NewFromInt(103)},
			},
			wantErr: true,
		},
		{
			name: "exact 100 without an Other row needs no injection",
			in: []model.ExposureRow{
				{Name: "North America", Weight: decimal.NewFromInt(95)},
				{Name: "Japan", Weight: decimal.NewFromInt(5)},
			},
			want: map[string]string{
				"North America": "95",
				"Japan":         "5",
			},
			wantSum: "100",
		},
		{
			name: "explicit zero Other row is dropped and not re-added",
			in: []model.ExposureRow{
				{Name: "North America", Weight: decimal.NewFromInt(100)},
				{Name: "Other / Not Classified", Weight: decimal.Zero},
			},
			want: map[string]string{
				"North America": "100",
			},
			wantSum: "100",
		},
		{
			name: "explicit positive Other row is kept as sent",
			in: []model.ExposureRow{
				{Name: "North America", Weight: decimal.NewFromInt(90)},
				{Name: geo.OtherRegion, Weight: decimal.NewFromInt(8)},
			},
			want: map[string]string{
				"North America":          "90",
				"Other / Not Classified": "8",
			},
			wantSum: "98",
		},
		{
			name: "residual within the 0.5 rounding tolerance is not injected",
			in: []model.ExposureRow{
				{Name: "North America", Weight: decimal.NewFromFloat(99.7)},
			},
			want: map[string]string{
				"North America": "99.7",
			},
			wantSum: "99.7",
		},
		{
			name:    "empty dimension closes entirely into Other",
			in:      []model.ExposureRow{},
			want:    map[string]string{"Other / Not Classified": "100"},
			wantSum: "100",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := prepareRegions(tc.in)
			if tc.wantErr {
				if !errors.Is(err, ErrInvalidWeights) {
					t.Fatalf("err = %v, want ErrInvalidWeights", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			byName := map[string]decimal.Decimal{}
			for _, r := range got {
				byName[r.Name] = r.Weight
			}
			if len(got) != len(tc.want) {
				t.Fatalf("rows = %+v, want %d rows", got, len(tc.want))
			}
			for name, w := range tc.want {
				if !byName[name].Equal(decimal.RequireFromString(w)) {
					t.Fatalf("region %q = %v, want %s", name, byName[name], w)
				}
			}
			if tc.wantSum != "" && !exposureWeightsSum(got).Equal(decimal.RequireFromString(tc.wantSum)) {
				t.Fatalf("stored sum = %v, want %s", exposureWeightsSum(got), tc.wantSum)
			}
		})
	}
}

func TestPrepareCountries(t *testing.T) {
	cases := []struct {
		name    string
		in      []model.ExposureRow
		wantErr bool
		want    []model.ExposureRow
	}{
		{
			name: "sum of 120 is rejected",
			in: []model.ExposureRow{
				{Name: "US", Weight: decimal.NewFromInt(70)},
				{Name: "JP", Weight: decimal.NewFromInt(50)},
			},
			wantErr: true,
		},
		{
			name: "sum of 95 is accepted with no lower bound",
			in: []model.ExposureRow{
				{Name: "US", Weight: decimal.NewFromInt(60)},
				{Name: "GB", Weight: decimal.NewFromInt(25)},
				{Name: "JP", Weight: decimal.NewFromInt(10)},
			},
			want: []model.ExposureRow{
				{Name: "US", Weight: decimal.NewFromInt(60)},
				{Name: "GB", Weight: decimal.NewFromInt(25)},
				{Name: "JP", Weight: decimal.NewFromInt(10)},
			},
		},
		{
			name: "non-canonical rows are dropped before the upper-bound check",
			in: []model.ExposureRow{
				{Name: "US", Weight: decimal.NewFromInt(60)},
				{Name: "Atlantis", Weight: decimal.NewFromInt(60)},
			},
			want: []model.ExposureRow{
				{Name: "US", Weight: decimal.NewFromInt(60)},
			},
		},
		{
			name: "names normalize to canonical ISO codes",
			in: []model.ExposureRow{
				{Name: "United States", Weight: decimal.NewFromInt(60)},
				{Name: "DE", Weight: decimal.NewFromInt(40)},
			},
			want: []model.ExposureRow{
				{Name: "US", Weight: decimal.NewFromInt(60)},
				{Name: "DE", Weight: decimal.NewFromInt(40)},
			},
		},
		{
			name: "empty and non-positive rows are dropped",
			in: []model.ExposureRow{
				{Name: "", Weight: decimal.NewFromInt(50)},
				{Name: "JP", Weight: decimal.Zero},
				{Name: "US", Weight: decimal.NewFromInt(100)},
			},
			want: []model.ExposureRow{
				{Name: "US", Weight: decimal.NewFromInt(100)},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := prepareCountries(tc.in)
			if tc.wantErr {
				if !errors.Is(err, ErrInvalidWeights) {
					t.Fatalf("err = %v, want ErrInvalidWeights", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("rows = %+v, want %+v", got, tc.want)
			}
			for i := range got {
				if got[i].Name != tc.want[i].Name || !got[i].Weight.Equal(tc.want[i].Weight) {
					t.Fatalf("row[%d] = %+v, want %+v", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestPrepareSectorsKeepsExactRule(t *testing.T) {
	sectors := normalizeExposureRows([]model.ExposureRow{
		{Name: "Technology", Weight: decimal.NewFromInt(103)},
	})
	if err := validateExposureWeights(sectors, weightSumExact100); !errors.Is(err, ErrInvalidWeights) {
		t.Fatalf("103%% sectors err = %v, want ErrInvalidWeights", err)
	}
	sectors = normalizeExposureRows([]model.ExposureRow{
		{Name: "Technology", Weight: decimal.NewFromInt(100)},
	})
	if err := validateExposureWeights(sectors, weightSumExact100); err != nil {
		t.Fatalf("100%% sectors err = %v, want nil", err)
	}
	sectors = normalizeExposureRows([]model.ExposureRow{
		{Name: "Technology", Weight: decimal.NewFromInt(95)},
	})
	if err := validateExposureWeights(sectors, weightSumExact100); !errors.Is(err, ErrInvalidWeights) {
		t.Fatalf("95%% sectors err = %v, want ErrInvalidWeights (sectors keep the exact-100 rule)", err)
	}
}

// assertNoExposureWrites checks that a preview fetch wrote no exposure
// dimension and no provenance at all.
func assertNoExposureWrites(t *testing.T, e *fakeExposureRepo, what string) {
	t.Helper()
	if e.replaceCountriesCalls != 0 || e.replaceRegionsCalls != 0 || e.replaceSectorsCalls != 0 {
		t.Fatalf("%s persisted exposure: ReplaceCountries=%d ReplaceRegions=%d ReplaceSectors=%d, want all 0",
			what, e.replaceCountriesCalls, e.replaceRegionsCalls, e.replaceSectorsCalls)
	}
	if e.setProvenanceCalls != 0 {
		t.Fatalf("%s persisted provenance: SetProvenance=%d, want 0", what, e.setProvenanceCalls)
	}
}

func TestFetchETFExposure_PreviewDoesNotPersist(t *testing.T) {
	assetID := uuid.New()
	a := &fakeAssetRepo{asset: &model.Asset{ID: assetID, Ticker: "SXR8.DE", Type: model.AssetTypeETF, ISIN: "IE00B4L5Y983"}}
	e := &fakeExposureRepo{}
	etf := &fakeETFFetcher{exposure: &model.AssetExposure{
		Countries: []model.ExposureRow{
			{Name: "United States", Weight: decimal.NewFromInt(60)},
			{Name: "Germany", Weight: decimal.NewFromInt(40)},
		},
		Sectors: []model.ExposureRow{
			{Name: "Technology", Weight: decimal.NewFromInt(50)},
			{Name: "Healthcare", Weight: decimal.NewFromInt(50)},
		},
	}}
	svc := newFetchTestService(t, a, e, nil, nil, etf)

	got, err := svc.FetchETFExposure(context.Background(), assetID, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertNoExposureWrites(t, e, "FetchETFExposure")
	if a.updateCalls != 0 {
		t.Fatalf("asset.Update calls = %d, want 0 (an asset with an ISIN must not be touched)", a.updateCalls)
	}
	if etf.exposureISIN != "IE00B4L5Y983" {
		t.Fatalf("provider called with ISIN %q, want IE00B4L5Y983", etf.exposureISIN)
	}
	if got.ISIN != "IE00B4L5Y983" {
		t.Fatalf("response ISIN = %q, want IE00B4L5Y983", got.ISIN)
	}
	if len(got.Countries) != len(geo.Countries) || len(got.Regions) != len(geo.Regions) || len(got.Sectors) != len(geo.GICSSectors) {
		t.Fatalf("preview not canonical: countries=%d regions=%d sectors=%d", len(got.Countries), len(got.Regions), len(got.Sectors))
	}
	countries := exposureRowsByName(got.Countries)
	if !countries["US"].Equal(decimal.NewFromInt(60)) || !countries["DE"].Equal(decimal.NewFromInt(40)) {
		t.Fatalf("preview countries = %+v, want normalized US 60 + DE 40", got.Countries)
	}
	regions := exposureRowsByName(got.Regions)
	if !regions["North America"].Equal(decimal.NewFromInt(60)) || !regions["Europe Developed"].Equal(decimal.NewFromInt(40)) {
		t.Fatalf("preview regions = %+v, want aggregated North America 60 + Europe Developed 40", got.Regions)
	}
	sectors := exposureRowsByName(got.Sectors)
	if !sectors["Information Technology"].Equal(decimal.NewFromInt(50)) || !sectors["Health Care"].Equal(decimal.NewFromInt(50)) {
		t.Fatalf("preview sectors = %+v, want aggregated Info Tech 50 + Health Care 50", got.Sectors)
	}
}

func TestFetchETFExposure_PreviewPersistsOnlyResolvedISIN(t *testing.T) {
	assetID := uuid.New()
	a := &fakeAssetRepo{asset: &model.Asset{ID: assetID, Ticker: "CW9", Name: "iShares Core MSCI World UCITS ETF USD (Acc)", Type: model.AssetTypeETF}}
	e := &fakeExposureRepo{}
	etf := &fakeETFFetcher{
		search: []price.EtfSearchResult{{ISIN: "IE00B4L5Y983", Ticker: "CW9", Name: "iShares Core MSCI World UCITS ETF USD (Acc)"}},
		exposure: &model.AssetExposure{
			Countries: []model.ExposureRow{{Name: "US", Weight: decimal.NewFromInt(100)}},
			Sectors:   []model.ExposureRow{{Name: "Technology", Weight: decimal.NewFromInt(100)}},
		},
	}
	svc := newFetchTestService(t, a, e, nil, nil, etf)

	got, err := svc.FetchETFExposure(context.Background(), assetID, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.updateCalls != 1 {
		t.Fatalf("asset.Update calls = %d, want 1 (only the resolved ISIN is persisted)", a.updateCalls)
	}
	if a.lastUpdate == nil || a.lastUpdate.ISIN != "IE00B4L5Y983" {
		t.Fatalf("updated asset = %+v, want ISIN IE00B4L5Y983", a.lastUpdate)
	}
	if etf.exposureISIN != "IE00B4L5Y983" {
		t.Fatalf("provider called with ISIN %q, want the resolved IE00B4L5Y983", etf.exposureISIN)
	}
	assertNoExposureWrites(t, e, "FetchETFExposure")
	if got.ISIN != "IE00B4L5Y983" {
		t.Fatalf("response ISIN = %q, want IE00B4L5Y983", got.ISIN)
	}
}

func TestFetchMorningstarExposure_PreviewDoesNotPersist(t *testing.T) {
	assetID := uuid.New()
	a := &fakeAssetRepo{asset: &model.Asset{ID: assetID, Ticker: "SXR8.DE", Type: model.AssetTypeETF, ISIN: "IE00B4L5Y983"}}
	e := &fakeExposureRepo{}
	etf := &fakeETFFetcher{morningstar: &model.AssetExposure{
		Countries: []model.ExposureRow{
			{Name: "United States", Weight: decimal.NewFromInt(60)},
			{Name: "Germany", Weight: decimal.NewFromInt(40)},
		},
		Regions: []model.ExposureRow{
			{Name: "North America", Weight: decimal.NewFromInt(60)},
			{Name: "Europe Developed", Weight: decimal.NewFromInt(40)},
		},
		Sectors: []model.ExposureRow{
			{Name: "Technology", Weight: decimal.NewFromInt(100)},
		},
	}}
	svc := newFetchTestService(t, a, e, nil, nil, etf)

	got, err := svc.FetchMorningstarExposure(context.Background(), assetID, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertNoExposureWrites(t, e, "FetchMorningstarExposure")
	if a.updateCalls != 0 {
		t.Fatalf("asset.Update calls = %d, want 0 (an asset with an ISIN must not be touched)", a.updateCalls)
	}
	if etf.morningstarISIN != "IE00B4L5Y983" {
		t.Fatalf("provider called with ISIN %q, want IE00B4L5Y983", etf.morningstarISIN)
	}
	countries := exposureRowsByName(got.Countries)
	if !countries["US"].Equal(decimal.NewFromInt(60)) || !countries["DE"].Equal(decimal.NewFromInt(40)) {
		t.Fatalf("preview countries = %+v, want normalized US 60 + DE 40", got.Countries)
	}
	regions := exposureRowsByName(got.Regions)
	if !regions["North America"].Equal(decimal.NewFromInt(60)) || !regions["Europe Developed"].Equal(decimal.NewFromInt(40)) {
		t.Fatalf("preview regions = %+v, want official North America 60 + Europe Developed 40", got.Regions)
	}
	sectors := exposureRowsByName(got.Sectors)
	if !sectors["Information Technology"].Equal(decimal.NewFromInt(100)) {
		t.Fatalf("preview sectors = %+v, want aggregated Information Technology 100", got.Sectors)
	}
	if len(got.Regions) != len(geo.Regions) || len(got.Sectors) != len(geo.GICSSectors) {
		t.Fatalf("preview not canonical: regions=%d sectors=%d", len(got.Regions), len(got.Sectors))
	}
}

func TestFetchMorningstarExposure_PreviewPersistsOnlyResolvedISIN(t *testing.T) {
	assetID := uuid.New()
	a := &fakeAssetRepo{asset: &model.Asset{ID: assetID, Ticker: "CW9", Name: "iShares Core MSCI World UCITS ETF USD (Acc)", Type: model.AssetTypeETF}}
	e := &fakeExposureRepo{}
	etf := &fakeETFFetcher{
		search: []price.EtfSearchResult{{ISIN: "IE00B4L5Y983", Ticker: "CW9", Name: "iShares Core MSCI World UCITS ETF USD (Acc)"}},
		morningstar: &model.AssetExposure{
			Countries: []model.ExposureRow{{Name: "US", Weight: decimal.NewFromInt(100)}},
		},
	}
	svc := newFetchTestService(t, a, e, nil, nil, etf)

	got, err := svc.FetchMorningstarExposure(context.Background(), assetID, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.updateCalls != 1 {
		t.Fatalf("asset.Update calls = %d, want 1 (only the resolved ISIN is persisted)", a.updateCalls)
	}
	if a.lastUpdate == nil || a.lastUpdate.ISIN != "IE00B4L5Y983" {
		t.Fatalf("updated asset = %+v, want ISIN IE00B4L5Y983", a.lastUpdate)
	}
	if etf.morningstarISIN != "IE00B4L5Y983" {
		t.Fatalf("provider called with ISIN %q, want the resolved IE00B4L5Y983", etf.morningstarISIN)
	}
	assertNoExposureWrites(t, e, "FetchMorningstarExposure")
	if got.ISIN != "IE00B4L5Y983" {
		t.Fatalf("response ISIN = %q, want IE00B4L5Y983", got.ISIN)
	}
}

func TestFetchAssetExposure_PreviewDoesNotPersist(t *testing.T) {
	assetID := uuid.New()
	key := assetID.String()
	a := &fakeAssetRepo{asset: &model.Asset{ID: assetID, Ticker: "AAPL", Type: model.AssetTypeStock, Country: "US", Sector: "Utilities"}}
	e := &fakeExposureRepo{
		regions:   map[string][]model.ExposureRow{key: {{Name: "North America", Weight: decimal.NewFromInt(100)}}},
		countries: map[string][]model.ExposureRow{key: {{Name: "US", Weight: decimal.NewFromInt(100)}}},
	}
	yf := &fakeYahooFetcher{
		sector:   "Technology",
		industry: "Consumer Electronics",
		country:  "United States",
		weightings: []model.ExposureRow{
			{Name: "Information Technology", Weight: decimal.NewFromInt(60)},
			{Name: "Health Care", Weight: decimal.NewFromInt(40)},
		},
	}
	svc := newFetchTestService(t, a, e, nil, yf, nil)

	got, err := svc.FetchAssetExposure(context.Background(), assetID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if yf.profileExtendedCalls != 1 {
		t.Fatalf("provider calls = %d, want 1", yf.profileExtendedCalls)
	}
	assertNoExposureWrites(t, e, "FetchAssetExposure")
	if a.updateCalls != 0 {
		t.Fatalf("asset.Update calls = %d, want 0 (profile fields are preview-only)", a.updateCalls)
	}
	if a.asset.Sector != "Utilities" || a.asset.Country != "US" {
		t.Fatalf("asset profile mutated by preview: sector=%q country=%q", a.asset.Sector, a.asset.Country)
	}
	sectors := exposureRowsByName(got.Sectors)
	if !sectors["Information Technology"].Equal(decimal.NewFromInt(60)) || !sectors["Health Care"].Equal(decimal.NewFromInt(40)) {
		t.Fatalf("preview sectors = %+v, want the provider's Info Tech 60 + Health Care 40", got.Sectors)
	}
	if len(got.Sectors) != len(geo.GICSSectors) {
		t.Fatalf("preview sectors len = %d, want %d (canonical)", len(got.Sectors), len(geo.GICSSectors))
	}
	// The stored regions/countries dimensions are read back into the preview.
	regions := exposureRowsByName(got.Regions)
	if !regions["North America"].Equal(decimal.NewFromInt(100)) {
		t.Fatalf("preview regions = %+v, want the stored North America 100", got.Regions)
	}
	countries := exposureRowsByName(got.Countries)
	if !countries["US"].Equal(decimal.NewFromInt(100)) {
		t.Fatalf("preview countries = %+v, want the stored US 100", got.Countries)
	}
}

func TestFetchETFExposure_CachesFetchOnMissAndServesFromCache(t *testing.T) {
	assetID := uuid.New()
	a := &fakeAssetRepo{asset: &model.Asset{ID: assetID, Ticker: "SXR8.DE", Type: model.AssetTypeETF, ISIN: "IE00B4L5Y983"}}
	e := &fakeExposureRepo{}
	lk := &fakeLookupRepo{}
	etf := &fakeETFFetcher{exposure: &model.AssetExposure{
		Countries: []model.ExposureRow{
			{Name: "United States", Weight: decimal.NewFromInt(60)},
			{Name: "Germany", Weight: decimal.NewFromInt(40)},
		},
		Sectors: []model.ExposureRow{{Name: "Technology", Weight: decimal.NewFromInt(100)}},
	}}
	svc := newFetchTestService(t, a, e, lk, nil, etf)

	first, err := svc.FetchETFExposure(context.Background(), assetID, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if etf.exposureCalls != 1 {
		t.Fatalf("provider calls = %d, want 1 on cache miss", etf.exposureCalls)
	}
	if lk.setCalls != 1 {
		t.Fatalf("cache Set calls = %d, want 1 after a miss", lk.setCalls)
	}
	if _, ok := lk.data[exposureCacheKey("justetf", "IE00B4L5Y983")]; !ok {
		t.Fatalf("payload not stored under the justetf cache key: %v", lk.data)
	}

	// A different provider payload must not surface: the second call is served
	// from the cached countries, so the fetcher is never reached again.
	etf.exposure = &model.AssetExposure{
		Countries: []model.ExposureRow{{Name: "Japan", Weight: decimal.NewFromInt(100)}},
	}
	second, err := svc.FetchETFExposure(context.Background(), assetID, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if etf.exposureCalls != 1 {
		t.Fatalf("provider calls = %d, want 1 (cache hit must skip the fetcher)", etf.exposureCalls)
	}
	if lk.setCalls != 1 {
		t.Fatalf("cache Set calls = %d, want 1 (a hit must not rewrite the cache)", lk.setCalls)
	}
	countries := exposureRowsByName(second.Countries)
	if !countries["US"].Equal(decimal.NewFromInt(60)) || !countries["DE"].Equal(decimal.NewFromInt(40)) {
		t.Fatalf("cached preview countries = %+v, want US 60 + DE 40 from the first fetch", second.Countries)
	}
	if !countries["JP"].IsZero() {
		t.Fatalf("cached preview leaked the fresh provider payload: %+v", second.Countries)
	}
	if len(first.Countries) != len(geo.Countries) {
		t.Fatalf("first preview not canonical: countries=%d", len(first.Countries))
	}
}

func TestFetchETFExposure_RefreshBypassesCache(t *testing.T) {
	assetID := uuid.New()
	a := &fakeAssetRepo{asset: &model.Asset{ID: assetID, Ticker: "SXR8.DE", Type: model.AssetTypeETF, ISIN: "IE00B4L5Y983"}}
	lk := &fakeLookupRepo{}
	etf := &fakeETFFetcher{exposure: &model.AssetExposure{
		Countries: []model.ExposureRow{{Name: "Germany", Weight: decimal.NewFromInt(100)}},
	}}
	svc := newFetchTestService(t, a, &fakeExposureRepo{}, lk, nil, etf)

	if _, err := svc.FetchETFExposure(context.Background(), assetID, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	getsAfterPrime := lk.getCalls

	etf.exposure = &model.AssetExposure{
		Countries: []model.ExposureRow{{Name: "United States", Weight: decimal.NewFromInt(100)}},
	}
	got, err := svc.FetchETFExposure(context.Background(), assetID, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if etf.exposureCalls != 2 {
		t.Fatalf("provider calls = %d, want 2 (refresh=true must re-fetch)", etf.exposureCalls)
	}
	if lk.getCalls != getsAfterPrime {
		t.Fatalf("cache Get calls = %d, want %d (refresh=true must not read the cache)", lk.getCalls, getsAfterPrime)
	}
	if lk.setCalls != 2 {
		t.Fatalf("cache Set calls = %d, want 2 (refresh=true must rewrite the cache)", lk.setCalls)
	}
	countries := exposureRowsByName(got.Countries)
	if !countries["US"].Equal(decimal.NewFromInt(100)) {
		t.Fatalf("preview countries = %+v, want the refreshed US 100 payload", got.Countries)
	}

	// The rewritten payload is what subsequent non-refresh calls serve.
	if _, err := svc.FetchETFExposure(context.Background(), assetID, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if etf.exposureCalls != 2 {
		t.Fatalf("provider calls = %d, want 2 (the refreshed payload must be cached)", etf.exposureCalls)
	}
}

func TestFetchETFExposure_EmptyResultNotCached(t *testing.T) {
	assetID := uuid.New()
	a := &fakeAssetRepo{asset: &model.Asset{ID: assetID, Ticker: "SXR8.DE", Type: model.AssetTypeETF, ISIN: "IE00B4L5Y983"}}
	lk := &fakeLookupRepo{}
	etf := &fakeETFFetcher{exposure: &model.AssetExposure{}}
	svc := newFetchTestService(t, a, &fakeExposureRepo{}, lk, nil, etf)

	if _, err := svc.FetchETFExposure(context.Background(), assetID, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lk.setCalls != 0 {
		t.Fatalf("cache Set calls = %d, want 0 (a country-less payload must not be cached)", lk.setCalls)
	}
	if len(lk.data) != 0 {
		t.Fatalf("cache data = %v, want empty", lk.data)
	}
	if _, err := svc.FetchETFExposure(context.Background(), assetID, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if etf.exposureCalls != 2 {
		t.Fatalf("provider calls = %d, want 2 (nothing was cached, so the fetcher runs again)", etf.exposureCalls)
	}
}

func TestFetchMorningstarExposure_CachesUnderItsOwnKey(t *testing.T) {
	assetID := uuid.New()
	a := &fakeAssetRepo{asset: &model.Asset{ID: assetID, Ticker: "SXR8.DE", Type: model.AssetTypeETF, ISIN: "IE00B4L5Y983"}}
	lk := &fakeLookupRepo{}
	etf := &fakeETFFetcher{
		exposure: &model.AssetExposure{
			Countries: []model.ExposureRow{{Name: "Germany", Weight: decimal.NewFromInt(100)}},
		},
		morningstar: &model.AssetExposure{
			Countries: []model.ExposureRow{{Name: "United States", Weight: decimal.NewFromInt(60)}},
			Regions:   []model.ExposureRow{{Name: "North America", Weight: decimal.NewFromInt(60)}},
			Sectors:   []model.ExposureRow{{Name: "Technology", Weight: decimal.NewFromInt(100)}},
		},
	}
	svc := newFetchTestService(t, a, &fakeExposureRepo{}, lk, nil, etf)

	// Priming the justetf cache must not feed the morningstar source.
	if _, err := svc.FetchETFExposure(context.Background(), assetID, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	first, err := svc.FetchMorningstarExposure(context.Background(), assetID, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if etf.morningstarCall != 1 {
		t.Fatalf("morningstar provider calls = %d, want 1 (separate cache key per source)", etf.morningstarCall)
	}
	if _, ok := lk.data[exposureCacheKey("morningstar", "IE00B4L5Y983")]; !ok {
		t.Fatalf("payload not stored under the morningstar cache key: %v", lk.data)
	}

	etf.morningstar = &model.AssetExposure{
		Countries: []model.ExposureRow{{Name: "Japan", Weight: decimal.NewFromInt(100)}},
	}
	second, err := svc.FetchMorningstarExposure(context.Background(), assetID, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if etf.morningstarCall != 1 {
		t.Fatalf("morningstar provider calls = %d, want 1 (cache hit must skip the fetcher)", etf.morningstarCall)
	}
	countries := exposureRowsByName(second.Countries)
	if !countries["US"].Equal(decimal.NewFromInt(60)) || !countries["JP"].IsZero() {
		t.Fatalf("cached preview countries = %+v, want US 60 from the first fetch", second.Countries)
	}
	regions := exposureRowsByName(second.Regions)
	if !regions["North America"].Equal(decimal.NewFromInt(60)) {
		t.Fatalf("cached preview regions = %+v, want the cached official North America 60", second.Regions)
	}
	if len(first.Regions) != len(geo.Regions) {
		t.Fatalf("first preview not canonical: regions=%d", len(first.Regions))
	}
}

func TestRefreshPrices_SetsFinishedAt(t *testing.T) {
	svc := newFetchTestService(t, &fakeAssetRepo{}, &fakeExposureRepo{}, nil, &fakeYahooFetcher{}, nil)
	portfolioID := uuid.New()

	for _, pid := range []*uuid.UUID{nil, &portfolioID} {
		start := time.Now().UTC().Add(-time.Second)
		report, err := svc.RefreshPrices(context.Background(), pid)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if report.FinishedAt.IsZero() {
			t.Fatal("finished_at is zero, want the refresh completion timestamp")
		}
		if report.FinishedAt.Before(start) || report.FinishedAt.After(time.Now().UTC()) {
			t.Fatalf("finished_at = %v, want a timestamp between the call start and now", report.FinishedAt)
		}
	}
}

func TestIsYahooPriced(t *testing.T) {
	cases := []struct {
		source string
		want   bool
	}{
		{"yahoo", true},
		{"", true}, // legacy default: the column postdates the first assets
		{"manual", false},
		{"none", false},
	}
	for _, tc := range cases {
		a := &model.Asset{Ticker: "X", PriceSource: tc.source}
		if got := isYahooPriced(a); got != tc.want {
			t.Fatalf("isYahooPriced(price_source=%q) = %v, want %v", tc.source, got, tc.want)
		}
	}
}

func TestFilterYahooAssets(t *testing.T) {
	assets := []*model.Asset{
		{ID: uuid.New(), Ticker: "AAPL", PriceSource: "yahoo"},
		{ID: uuid.New(), Ticker: "LEGACY", PriceSource: ""},
		{ID: uuid.New(), Ticker: "BTC", PriceSource: "manual"},
		{ID: uuid.New(), Ticker: "CASH", PriceSource: "none"},
	}
	got := filterYahooAssets(assets)
	if len(got) != 2 || got[0].Ticker != "AAPL" || got[1].Ticker != "LEGACY" {
		t.Fatalf("filtered = %+v, want [AAPL LEGACY] in input order", got)
	}
	if out := filterYahooAssets(nil); len(out) != 0 {
		t.Fatalf("filter(nil) = %+v, want empty", out)
	}
}

// TestSyncAssets_OnlyForwardsYahooPriced covers the GetPortfolioHistory
// filtering too: that path builds its EnsureHistory/EnsureSplits inputs with
// the same filterYahooAssets helper (a full GetPortfolioHistory test would
// need the whole series/Split repository surface stubbed).
func TestSyncAssets_OnlyForwardsYahooPriced(t *testing.T) {
	assets := []*model.Asset{
		{ID: uuid.New(), Ticker: "AAPL", PriceSource: "yahoo"},
		{ID: uuid.New(), Ticker: "BTC", PriceSource: "manual"},
		{ID: uuid.New(), Ticker: "LEGACY", PriceSource: ""},
		{ID: uuid.New(), Ticker: "CASH", PriceSource: "none"},
	}
	ids := make([]uuid.UUID, 0, len(assets))
	for _, a := range assets {
		ids = append(ids, a.ID)
	}
	yf := &fakeYahooFetcher{}
	svc := newSyncTestService(t, &fakeAssetRepo{assets: assets}, &fakeTransactionRepo{}, yf)

	if err := svc.syncAssets(context.Background(), ids); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if yf.historyCalls != 1 || yf.splitCalls != 1 {
		t.Fatalf("fetcher calls = history %d splits %d, want 1 and 1", yf.historyCalls, yf.splitCalls)
	}
	want := []string{"AAPL", "LEGACY"}
	if len(yf.historyTickers) != len(want) || yf.historyTickers[0] != want[0] || yf.historyTickers[1] != want[1] {
		t.Fatalf("EnsureHistory tickers = %v, want %v (manual/none must not reach Yahoo)", yf.historyTickers, want)
	}
	if len(yf.splitTickers) != len(want) || yf.splitTickers[0] != want[0] || yf.splitTickers[1] != want[1] {
		t.Fatalf("EnsureSplits tickers = %v, want %v (manual/none must not reach Yahoo)", yf.splitTickers, want)
	}
}

func TestSyncAssets_AllNonYahooSkipsFetcher(t *testing.T) {
	assets := []*model.Asset{
		{ID: uuid.New(), Ticker: "BTC", PriceSource: "manual"},
		{ID: uuid.New(), Ticker: "CASH", PriceSource: "none"},
	}
	ids := []uuid.UUID{assets[0].ID, assets[1].ID}
	yf := &fakeYahooFetcher{}
	svc := newSyncTestService(t, &fakeAssetRepo{assets: assets}, &fakeTransactionRepo{}, yf)

	if err := svc.syncAssets(context.Background(), ids); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if yf.historyCalls != 0 || yf.splitCalls != 0 {
		t.Fatalf("fetcher calls = history %d splits %d, want 0 and 0", yf.historyCalls, yf.splitCalls)
	}
}

func TestBackfillAssetHistory_SkipsNonYahooAssets(t *testing.T) {
	for _, source := range []string{"manual", "none"} {
		asset := &model.Asset{ID: uuid.New(), Ticker: "BTC", PriceSource: source}
		yf := &fakeYahooFetcher{}
		svc := newFetchTestService(t, &fakeAssetRepo{asset: asset}, &fakeExposureRepo{}, nil, yf, nil)

		if err := svc.BackfillAssetHistory(context.Background(), asset.ID); err != nil {
			t.Fatalf("price_source=%q: unexpected error: %v", source, err)
		}
		if yf.historyCalls != 0 {
			t.Fatalf("price_source=%q: EnsureHistory calls = %d, want 0 (silent no-op)", source, yf.historyCalls)
		}
	}
}

// fakeUserRepo is a single-user stand-in for repository.UserRepository:
// FindByID serves the stored record whatever the requested id (the service
// tests always pass the id the fake owns) and Update persists a snapshot so
// the refresh-after-write flow is observable.
type fakeUserRepo struct {
	user        *model.User
	updateCalls int
}

func (f *fakeUserRepo) Create(ctx context.Context, email, name, password string) (*model.User, error) {
	return f.user, nil
}
func (f *fakeUserRepo) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	if f.user == nil || f.user.Email != email {
		return nil, pgx.ErrNoRows
	}
	u := *f.user
	return &u, nil
}
func (f *fakeUserRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	if f.user == nil {
		return nil, pgx.ErrNoRows
	}
	u := *f.user
	return &u, nil
}
func (f *fakeUserRepo) Update(ctx context.Context, user *model.User) error {
	f.updateCalls++
	u := *user
	f.user = &u
	return nil
}
func (f *fakeUserRepo) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	return nil
}

// fakeCurrencyRepo is an in-memory whitelist stand-in for
// repository.CurrencyRepository; only the enabled-code surface carries data.
type fakeCurrencyRepo struct {
	enabled []string
}

func (f *fakeCurrencyRepo) ListEnabled(ctx context.Context) ([]model.Currency, error) {
	out := make([]model.Currency, 0, len(f.enabled))
	for _, c := range f.enabled {
		out = append(out, model.Currency{Code: c, Enabled: true})
	}
	return out, nil
}
func (f *fakeCurrencyRepo) ListAll(ctx context.Context) ([]model.Currency, error) {
	return f.ListEnabled(ctx)
}
func (f *fakeCurrencyRepo) Get(ctx context.Context, code string) (*model.Currency, error) {
	return nil, nil
}
func (f *fakeCurrencyRepo) Create(ctx context.Context, c *model.Currency) error { return nil }
func (f *fakeCurrencyRepo) Delete(ctx context.Context, code string) error       { return nil }
func (f *fakeCurrencyRepo) EnabledByCodes(ctx context.Context, codes []string) ([]string, error) {
	var out []string
	for _, c := range codes {
		for _, e := range f.enabled {
			if e == c {
				out = append(out, c)
				break
			}
		}
	}
	return out, nil
}
func (f *fakeCurrencyRepo) CountInUse(ctx context.Context, code string) (int, error) {
	return 0, nil
}

// fakeSeriesRepo is a canned series stand-in for
// repository.SeriesRepository: FindPortfolio serves preloaded per-asset
// series, the write paths are inert.
type fakeSeriesRepo struct {
	assets map[uuid.UUID][]model.AssetPositionSeries
}

func (f *fakeSeriesRepo) ReplacePortfolio(ctx context.Context, portfolioID uuid.UUID, agg []model.PositionPoint, assets []model.AssetPositionSeries) error {
	return nil
}
func (f *fakeSeriesRepo) FindPortfolioAgg(ctx context.Context, portfolioID uuid.UUID) ([]model.PositionPoint, error) {
	return nil, nil
}
func (f *fakeSeriesRepo) FindPortfolio(ctx context.Context, portfolioID uuid.UUID) ([]model.AssetPositionSeries, error) {
	return f.assets[portfolioID], nil
}
func (f *fakeSeriesRepo) HasPortfolio(ctx context.Context, portfolioID uuid.UUID) (bool, error) {
	return false, nil
}

func newDashboardTestService(t *testing.T, p *fakePortfolioRepo, fx *fakeFXRepo, baseCurrency string, assets map[uuid.UUID][]model.AssetPositionSeries) *Service {
	t.Helper()
	return newDashboardTestServiceWithTxs(t, p, fx, baseCurrency, assets, &fakeTransactionRepo{})
}

func newDashboardTestServiceWithTxs(t *testing.T, p *fakePortfolioRepo, fx *fakeFXRepo, baseCurrency string, assets map[uuid.UUID][]model.AssetPositionSeries, txs *fakeTransactionRepo) *Service {
	t.Helper()
	repos := &repository.Repository{
		Asset:       &fakeAssetRepo{},
		User:        &fakeUserRepo{user: &model.User{ID: uuid.New(), Email: "u@example.com", BaseCurrency: baseCurrency}},
		Portfolio:   p,
		Exposure:    &fakeExposureRepo{},
		FX:          fx,
		Series:      &fakeSeriesRepo{assets: assets},
		Transaction: txs,
	}
	return New(repos, nil, nil, nil, time.Minute, time.Hour, cache.New(nil), 0, 0, nil)
}

func newProfileTestService(t *testing.T, u *fakeUserRepo, c *fakeCurrencyRepo) *Service {
	t.Helper()
	repos := &repository.Repository{
		User:     u,
		Currency: c,
	}
	return New(repos, nil, nil, nil, time.Minute, time.Hour, cache.New(nil), 0, 0, nil)
}

// dashboardHolding is a full holding fixture for the base-currency summary:
// unlike the `holding` helper it carries the portfolio id, the portfolio
// currency cost basis and realized P&L the summary converts.
func dashboardHolding(portfolioID, assetID, currency string, qty, lastClose, cost, realized decimal.Decimal) *model.Holding {
	h := holding(assetID, currency, "US", "Technology", model.AssetTypeStock, qty, lastClose)
	h.PortfolioID = portfolioID
	h.Cost = cost
	h.Realized = realized
	return h
}

// dashboardClosedHolding extends dashboardHolding with the closed-lot figures
// (AVCO cost of sold lots and net sale proceeds) and the cumulative dividends:
// the active/closed dashboard breakdown classifies the dividends by the
// position's remaining quantity.
func dashboardClosedHolding(portfolioID, assetID, currency string, qty, lastClose, cost, closedCost, proceeds, dividends decimal.Decimal) *model.Holding {
	h := dashboardHolding(portfolioID, assetID, currency, qty, lastClose, cost, decimal.Zero)
	h.ClosedCost = closedCost
	h.Proceeds = proceeds
	h.Dividends = dividends
	return h
}

// seriesHolding is the minimal priced holding fixture the performance chart
// reads to know an asset is priced: it only carries the portfolio/asset ids
// and the HasPrice flag, the figures come from the stored series.
func seriesHolding(portfolioID uuid.UUID, assetID string) *model.Holding {
	h := dashboardHolding(portfolioID.String(), assetID, "USD",
		decimal.NewFromInt(1), decimal.NewFromInt(1), decimal.NewFromInt(1), decimal.Zero)
	return h
}

func TestUpdateProfile_InvalidBaseCurrencyRejected(t *testing.T) {
	uid := uuid.New()
	u := &fakeUserRepo{user: &model.User{ID: uid, Email: "a@b.co", Name: "Old", BaseCurrency: "USD"}}
	c := &fakeCurrencyRepo{enabled: []string{"EUR", "USD"}}
	svc := newProfileTestService(t, u, c)

	_, err := svc.UpdateProfile(context.Background(), uid, "New", "a@b.co", "zzz")
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("err = %v, want ErrInvalidInput", err)
	}
	if u.updateCalls != 0 {
		t.Fatalf("Update calls = %d, want 0 (a rejected currency must not persist anything)", u.updateCalls)
	}
	if u.user.BaseCurrency != "USD" {
		t.Fatalf("stored base currency = %q, want the untouched USD", u.user.BaseCurrency)
	}
}

func TestUpdateProfile_NormalizesAndPersistsBaseCurrency(t *testing.T) {
	uid := uuid.New()
	u := &fakeUserRepo{user: &model.User{ID: uid, Email: "a@b.co", Name: "Old", BaseCurrency: "USD"}}
	c := &fakeCurrencyRepo{enabled: []string{"EUR", "USD"}}
	svc := newProfileTestService(t, u, c)

	got, err := svc.UpdateProfile(context.Background(), uid, "New", "a@b.co", " eur ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.updateCalls != 1 {
		t.Fatalf("Update calls = %d, want 1", u.updateCalls)
	}
	if got.Name != "New" || got.BaseCurrency != "EUR" {
		t.Fatalf("refreshed user = %+v, want name New and normalized EUR base currency", got)
	}
}

func TestUpdateProfile_EmptyBaseCurrencyKeepsExisting(t *testing.T) {
	uid := uuid.New()
	u := &fakeUserRepo{user: &model.User{ID: uid, Email: "a@b.co", Name: "Old", BaseCurrency: "USD"}}
	c := &fakeCurrencyRepo{enabled: []string{"EUR", "USD"}}
	svc := newProfileTestService(t, u, c)

	got, err := svc.UpdateProfile(context.Background(), uid, "New", "a@b.co", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.BaseCurrency != "USD" {
		t.Fatalf("base currency = %q, want the existing USD (an empty input must not overwrite)", got.BaseCurrency)
	}
}

func TestGetDashboard_SummaryInBaseCurrency(t *testing.T) {
	pfUSD := uuid.New()
	pfEUR := uuid.New()
	pfCHF := uuid.New()
	pf := &fakePortfolioRepo{
		portfolios: []*model.Portfolio{
			{ID: pfUSD, Currency: "USD"},
			{ID: pfEUR, Currency: "EUR"},
			{ID: pfCHF, Currency: "CHF"},
		},
		holdings: []*model.Holding{
			// USD portfolio: everything converts through USD->EUR 0.9.
			dashboardHolding(pfUSD.String(), uuid.New().String(), "USD", decimal.NewFromInt(10), decimal.NewFromInt(100), decimal.NewFromInt(800), decimal.NewFromInt(50)),
			// EUR portfolio, EUR asset: identity conversion.
			dashboardHolding(pfEUR.String(), uuid.New().String(), "EUR", decimal.NewFromInt(5), decimal.NewFromInt(10), decimal.NewFromInt(40), decimal.NewFromInt(10)),
			// EUR portfolio, JPY asset: no JPY rate, the value is flagged raw.
			dashboardHolding(pfEUR.String(), uuid.New().String(), "JPY", decimal.NewFromInt(1), decimal.NewFromInt(1000), decimal.Zero, decimal.Zero),
			// CHF portfolio: no CHF rate, the unconvertible cost is flagged raw.
			dashboardHolding(pfCHF.String(), uuid.New().String(), "EUR", decimal.NewFromInt(1), decimal.NewFromInt(20), decimal.NewFromInt(15), decimal.NewFromInt(5)),
		},
	}
	fx := &fakeFXRepo{rates: map[string]decimal.Decimal{"EUR": decimal.RequireFromString("0.9")}}
	svc := newDashboardTestService(t, pf, fx, "EUR", nil)

	got, err := svc.GetDashboard(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.BaseCurrency != "EUR" {
		t.Fatalf("base_currency = %q, want EUR", got.BaseCurrency)
	}
	if got.Summary == nil {
		t.Fatal("summary = nil, want the base-currency roll-up")
	}
	s := got.Summary
	if s.Currency != "EUR" {
		t.Fatalf("summary currency = %q, want EUR", s.Currency)
	}
	// active invested = 800*0.9 + 40; active value = 10*100*0.9 + 5*10 + 1*20
	if !equalDecimal(s.Active.Invested, decimal.NewFromInt(760)) {
		t.Fatalf("active.invested = %v, want 760", s.Active.Invested)
	}
	if !equalDecimal(s.Active.Value, decimal.NewFromInt(970)) {
		t.Fatalf("active.value = %v, want 970", s.Active.Value)
	}
	if !equalDecimal(s.Active.GainLoss, decimal.NewFromInt(210)) {
		t.Fatalf("active.gain_loss = %v, want 210", s.Active.GainLoss)
	}
	assertDecimalInDelta(t, s.Active.GainLossPct, decimal.RequireFromString("27.63"), "0.01", "active.gain_loss_pct")
	// No sold lots or dividends in this fixture: the closed group stays zero
	// and the open positions contribute no active dividends.
	if !equalDecimal(s.Active.Dividends, decimal.Zero) {
		t.Fatalf("active.dividends = %v, want 0", s.Active.Dividends)
	}
	if !equalDecimal(s.Closed.Invested, decimal.Zero) || !equalDecimal(s.Closed.Proceeds, decimal.Zero) ||
		!equalDecimal(s.Closed.Realized, decimal.Zero) || !equalDecimal(s.Closed.RealizedPct, decimal.Zero) {
		t.Fatalf("closed = %+v, want all zeros", s.Closed)
	}
	// 1 unconvertible JPY value + the CHF portfolio's unconvertible cost: two
	// skipped conversions, raw totals 1000+15 (zero closed amounts are not
	// counted as missing).
	if s.FXMissingCount != 2 {
		t.Fatalf("fx_missing_count = %d, want 2", s.FXMissingCount)
	}
	if !equalDecimal(s.FXMissingValue, decimal.NewFromInt(1015)) {
		t.Fatalf("fx_missing_value = %v, want 1015", s.FXMissingValue)
	}
}

func TestGetDashboard_ActiveClosedBreakdown(t *testing.T) {
	pfUSD := uuid.New()
	pfEUR := uuid.New()
	pf := &fakePortfolioRepo{
		portfolios: []*model.Portfolio{
			{ID: pfUSD, Currency: "USD"},
			{ID: pfEUR, Currency: "EUR"},
		},
		holdings: []*model.Holding{
			// USD portfolio, fully closed position: only sold lots and
			// dividends, the latter folded into the closed proceeds.
			dashboardClosedHolding(pfUSD.String(), uuid.New().String(), "USD",
				decimal.Zero, decimal.NewFromInt(120), decimal.Zero,
				decimal.NewFromInt(100), decimal.NewFromInt(120), decimal.NewFromInt(10)),
			// USD portfolio, partially sold: open lots (and their dividends)
			// feed active, sold lots closed.
			dashboardClosedHolding(pfUSD.String(), uuid.New().String(), "USD",
				decimal.NewFromInt(5), decimal.NewFromInt(110), decimal.NewFromInt(250),
				decimal.NewFromInt(250), decimal.NewFromInt(300), decimal.NewFromInt(15)),
			// EUR portfolio, EUR asset: amounts convert EUR->USD at 2 (USD->EUR 0.5).
			dashboardClosedHolding(pfEUR.String(), uuid.New().String(), "EUR",
				decimal.NewFromInt(2), decimal.NewFromInt(50), decimal.NewFromInt(80),
				decimal.NewFromInt(20), decimal.NewFromInt(110), decimal.NewFromInt(5)),
			// EUR portfolio, GBP asset: no GBP rate, only the value is unconvertible.
			dashboardClosedHolding(pfEUR.String(), uuid.New().String(), "GBP",
				decimal.NewFromInt(1), decimal.NewFromInt(100), decimal.Zero,
				decimal.Zero, decimal.Zero, decimal.Zero),
		},
	}
	fx := &fakeFXRepo{rates: map[string]decimal.Decimal{"EUR": decimal.RequireFromString("0.5")}}
	svc := newDashboardTestService(t, pf, fx, "USD", nil)

	got, err := svc.GetDashboard(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := got.Summary
	if s == nil {
		t.Fatal("summary = nil, want the base-currency roll-up")
	}
	// active invested = 250 + 80*2; active value = 5*110 + 2*50*2 (GBP dropped)
	if !equalDecimal(s.Active.Invested, decimal.NewFromInt(410)) {
		t.Fatalf("active.invested = %v, want 410", s.Active.Invested)
	}
	if !equalDecimal(s.Active.Value, decimal.NewFromInt(750)) {
		t.Fatalf("active.value = %v, want 750", s.Active.Value)
	}
	if !equalDecimal(s.Active.GainLoss, decimal.NewFromInt(340)) {
		t.Fatalf("active.gain_loss = %v, want 340", s.Active.GainLoss)
	}
	assertDecimalInDelta(t, s.Active.GainLossPct, decimal.RequireFromString("82.93"), "0.01", "active.gain_loss_pct")
	// active dividends = 15 (partially sold USD, still open) + 5*2 (EUR one)
	if !equalDecimal(s.Active.Dividends, decimal.NewFromInt(25)) {
		t.Fatalf("active.dividends = %v, want 25", s.Active.Dividends)
	}
	// closed invested = 100 + 250 + 20*2; proceeds = (120+10) + 300 + 110*2
	// (the fully closed USD position folds its dividends into the proceeds);
	// realized = proceeds - invested
	if !equalDecimal(s.Closed.Invested, decimal.NewFromInt(390)) {
		t.Fatalf("closed.invested = %v, want 390", s.Closed.Invested)
	}
	if !equalDecimal(s.Closed.Proceeds, decimal.NewFromInt(650)) {
		t.Fatalf("closed.proceeds = %v, want 650", s.Closed.Proceeds)
	}
	if !equalDecimal(s.Closed.Realized, decimal.NewFromInt(260)) {
		t.Fatalf("closed.realized = %v, want 260", s.Closed.Realized)
	}
	assertDecimalInDelta(t, s.Closed.RealizedPct, decimal.RequireFromString("66.67"), "0.01", "closed.realized_pct")
	if s.FXMissingCount != 1 || !equalDecimal(s.FXMissingValue, decimal.NewFromInt(100)) {
		t.Fatalf("fx missing = (%d, %v), want (1, 100) for the GBP value only", s.FXMissingCount, s.FXMissingValue)
	}

	if len(got.Portfolios) != 2 {
		t.Fatalf("portfolios len = %d, want 2", len(got.Portfolios))
	}
	usd := got.Portfolios[0]
	if usd.PortfolioID != pfUSD.String() {
		t.Fatalf("portfolios[0] = %s, want the USD portfolio", usd.PortfolioID)
	}
	if !equalDecimal(usd.Active.Invested, decimal.NewFromInt(250)) || !equalDecimal(usd.Active.Value, decimal.NewFromInt(550)) ||
		!equalDecimal(usd.Active.GainLoss, decimal.NewFromInt(300)) || !equalDecimal(usd.Active.Dividends, decimal.NewFromInt(15)) {
		t.Fatalf("USD portfolio active = %+v, want invested 250 value 550 gain 300 dividends 15", usd.Active)
	}
	assertDecimalInDelta(t, usd.Active.GainLossPct, decimal.NewFromInt(120), "0.01", "USD active.gain_loss_pct")
	// The fully closed position's 10 dividends join the 120 sale proceeds.
	if !equalDecimal(usd.Closed.Invested, decimal.NewFromInt(350)) || !equalDecimal(usd.Closed.Proceeds, decimal.NewFromInt(430)) ||
		!equalDecimal(usd.Closed.Realized, decimal.NewFromInt(80)) {
		t.Fatalf("USD portfolio closed = %+v, want invested 350 proceeds 430 realized 80", usd.Closed)
	}
	assertDecimalInDelta(t, usd.Closed.RealizedPct, decimal.RequireFromString("22.86"), "0.01", "USD closed.realized_pct")
	if usd.AssetCount != 2 || usd.FXMissing != 0 {
		t.Fatalf("USD portfolio asset_count/fx_missing = (%d, %d), want (2, 0)", usd.AssetCount, usd.FXMissing)
	}

	// EUR portfolio amounts stay in the portfolio currency, no FX applied.
	eur := got.Portfolios[1]
	if !equalDecimal(eur.Active.Invested, decimal.NewFromInt(80)) || !equalDecimal(eur.Active.Value, decimal.NewFromInt(100)) ||
		!equalDecimal(eur.Active.GainLoss, decimal.NewFromInt(20)) || !equalDecimal(eur.Active.Dividends, decimal.NewFromInt(5)) {
		t.Fatalf("EUR portfolio active = %+v, want invested 80 value 100 gain 20 dividends 5", eur.Active)
	}
	assertDecimalInDelta(t, eur.Active.GainLossPct, decimal.NewFromInt(25), "0.01", "EUR active.gain_loss_pct")
	if !equalDecimal(eur.Closed.Invested, decimal.NewFromInt(20)) || !equalDecimal(eur.Closed.Proceeds, decimal.NewFromInt(110)) ||
		!equalDecimal(eur.Closed.Realized, decimal.NewFromInt(90)) {
		t.Fatalf("EUR portfolio closed = %+v, want invested 20 proceeds 110 realized 90", eur.Closed)
	}
	assertDecimalInDelta(t, eur.Closed.RealizedPct, decimal.NewFromInt(450), "0.01", "EUR closed.realized_pct")
	if eur.AssetCount != 2 || eur.FXMissing != 1 {
		t.Fatalf("EUR portfolio asset_count/fx_missing = (%d, %d), want (2, 1)", eur.AssetCount, eur.FXMissing)
	}
}

func TestGetDashboard_ClosedHoldingResidualCostStaysOutOfActive(t *testing.T) {
	pfUSD := uuid.New()
	pf := &fakePortfolioRepo{
		portfolios: []*model.Portfolio{{ID: pfUSD, Currency: "USD"}},
		holdings: []*model.Holding{
			// Fully closed position still carrying an AVCO division residue in
			// its cost basis (2e-16 left across stock splits): with no open
			// lots the active group must stay at zero, not report -100%.
			dashboardClosedHolding(pfUSD.String(), uuid.New().String(), "USD",
				decimal.Zero, decimal.NewFromInt(120), decimal.RequireFromString("0.0000000000000002"),
				decimal.NewFromInt(1000), decimal.NewFromInt(1200), decimal.NewFromInt(50)),
		},
	}
	fx := &fakeFXRepo{}
	svc := newDashboardTestService(t, pf, fx, "USD", nil)

	got, err := svc.GetDashboard(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := got.Summary
	if !equalDecimal(s.Active.Invested, decimal.Zero) {
		t.Fatalf("active.invested = %v, want 0 (a closed position has no open cost basis)", s.Active.Invested)
	}
	if !equalDecimal(s.Active.Value, decimal.Zero) || !equalDecimal(s.Active.GainLoss, decimal.Zero) ||
		!equalDecimal(s.Active.GainLossPct, decimal.Zero) || !equalDecimal(s.Active.Dividends, decimal.Zero) {
		t.Fatalf("active = %+v, want all zeros", s.Active)
	}
	// The residue must not leak anywhere else: the closed group holds only
	// the sold lots' cost and proceeds (dividends folded into the proceeds).
	if !equalDecimal(s.Closed.Invested, decimal.NewFromInt(1000)) {
		t.Fatalf("closed.invested = %v, want 1000", s.Closed.Invested)
	}
	if !equalDecimal(s.Closed.Proceeds, decimal.NewFromInt(1250)) {
		t.Fatalf("closed.proceeds = %v, want 1250", s.Closed.Proceeds)
	}
	if !equalDecimal(s.Closed.Realized, decimal.NewFromInt(250)) {
		t.Fatalf("closed.realized = %v, want 250", s.Closed.Realized)
	}
	if s.FXMissingCount != 0 || !equalDecimal(s.FXMissingValue, decimal.Zero) {
		t.Fatalf("fx missing = (%d, %v), want (0, 0): the skipped cost must not be counted", s.FXMissingCount, s.FXMissingValue)
	}
	ps := got.Portfolios[0]
	if !equalDecimal(ps.Active.Invested, decimal.Zero) || !equalDecimal(ps.Active.GainLoss, decimal.Zero) ||
		!equalDecimal(ps.Active.GainLossPct, decimal.Zero) {
		t.Fatalf("portfolio active = %+v, want all zeros", ps.Active)
	}
	if !equalDecimal(ps.Closed.Realized, decimal.NewFromInt(250)) {
		t.Fatalf("portfolio closed.realized = %v, want 250", ps.Closed.Realized)
	}
}

func TestGetDashboard_UnpricedOpenPositionStaysOutOfActive(t *testing.T) {
	pfUSD := uuid.New()
	stockID := uuid.New().String()
	bondID := uuid.New().String()
	soldBondID := uuid.New().String()

	stock := dashboardHolding(pfUSD.String(), stockID, "USD",
		decimal.NewFromInt(10), decimal.NewFromInt(100), decimal.NewFromInt(800), decimal.Zero)
	// Open bond ETF position with price_source none: its 12663 cost has no
	// market value to be compared with, so it must stay out of the active
	// invested (otherwise the row would report a fake -100% loss); its
	// dividends are real and keep feeding the active group.
	bond := dashboardClosedHolding(pfUSD.String(), bondID, "USD",
		decimal.NewFromInt(5), decimal.Zero, decimal.NewFromInt(12663),
		decimal.Zero, decimal.Zero, decimal.NewFromInt(100))
	bond.HasPrice = false
	// A fully closed unpriced position keeps its real figures in the closed
	// group: the sold lots' cost and the sale proceeds (a 100 net loss here).
	soldBond := dashboardClosedHolding(pfUSD.String(), soldBondID, "USD",
		decimal.Zero, decimal.Zero, decimal.Zero,
		decimal.NewFromInt(1000), decimal.NewFromInt(900), decimal.Zero)
	soldBond.HasPrice = false

	pf := &fakePortfolioRepo{
		portfolios: []*model.Portfolio{{ID: pfUSD, Currency: "USD"}},
		holdings:   []*model.Holding{stock, bond, soldBond},
	}
	fx := &fakeFXRepo{}
	svc := newDashboardTestService(t, pf, fx, "USD", nil)

	got, err := svc.GetDashboard(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := got.Summary
	if s == nil {
		t.Fatal("summary = nil, want the base-currency roll-up")
	}
	// Only the priced stock feeds active invested/value; the unpriced bond
	// contributes its open-position dividends.
	if !equalDecimal(s.Active.Invested, decimal.NewFromInt(800)) {
		t.Fatalf("active.invested = %v, want 800 (the unpriced bond's 12663 cost must be excluded)", s.Active.Invested)
	}
	if !equalDecimal(s.Active.Value, decimal.NewFromInt(1000)) {
		t.Fatalf("active.value = %v, want 1000", s.Active.Value)
	}
	if !equalDecimal(s.Active.GainLoss, decimal.NewFromInt(200)) {
		t.Fatalf("active.gain_loss = %v, want 200", s.Active.GainLoss)
	}
	assertDecimalInDelta(t, s.Active.GainLossPct, decimal.NewFromInt(25), "0.01", "active.gain_loss_pct")
	if !equalDecimal(s.Active.Dividends, decimal.NewFromInt(100)) {
		t.Fatalf("active.dividends = %v, want 100 (dividends of the open unpriced position still count)", s.Active.Dividends)
	}
	// The closed unpriced position's realized stays in the closed group.
	if !equalDecimal(s.Closed.Invested, decimal.NewFromInt(1000)) {
		t.Fatalf("closed.invested = %v, want 1000", s.Closed.Invested)
	}
	if !equalDecimal(s.Closed.Proceeds, decimal.NewFromInt(900)) {
		t.Fatalf("closed.proceeds = %v, want 900", s.Closed.Proceeds)
	}
	if !equalDecimal(s.Closed.Realized, decimal.NewFromInt(-100)) {
		t.Fatalf("closed.realized = %v, want -100", s.Closed.Realized)
	}
	if s.FXMissingCount != 0 || !equalDecimal(s.FXMissingValue, decimal.Zero) {
		t.Fatalf("fx missing = (%d, %v), want (0, 0)", s.FXMissingCount, s.FXMissingValue)
	}

	ps := got.Portfolios[0]
	if !equalDecimal(ps.Active.Invested, decimal.NewFromInt(800)) || !equalDecimal(ps.Active.Value, decimal.NewFromInt(1000)) ||
		!equalDecimal(ps.Active.GainLoss, decimal.NewFromInt(200)) || !equalDecimal(ps.Active.Dividends, decimal.NewFromInt(100)) {
		t.Fatalf("portfolio active = %+v, want invested 800 value 1000 gain 200 dividends 100", ps.Active)
	}
	assertDecimalInDelta(t, ps.Active.GainLossPct, decimal.NewFromInt(25), "0.01", "portfolio active.gain_loss_pct")
	if !equalDecimal(ps.Closed.Realized, decimal.NewFromInt(-100)) {
		t.Fatalf("portfolio closed.realized = %v, want -100", ps.Closed.Realized)
	}
	if ps.FXMissing != 0 {
		t.Fatalf("portfolio fx_missing = %d, want 0 (the excluded cost must not be flagged)", ps.FXMissing)
	}
}

func TestGetDashboard_BreakdownAmountsAreRounded(t *testing.T) {
	pfUSD := uuid.New()
	pf := &fakePortfolioRepo{
		portfolios: []*model.Portfolio{{ID: pfUSD, Currency: "USD"}},
		holdings: []*model.Holding{
			// Closed lot whose AVCO cost carries a division residue: the
			// aggregated figures must come out as clean 2e-8 money.
			dashboardClosedHolding(pfUSD.String(), uuid.New().String(), "USD",
				decimal.Zero, decimal.NewFromInt(120), decimal.RequireFromString("0.0000000000000002"),
				decimal.RequireFromString("21063.5799999999999998"), decimal.RequireFromString("27381.22"), decimal.Zero),
			// Open position with a residue cost basis and dividends.
			dashboardClosedHolding(pfUSD.String(), uuid.New().String(), "USD",
				decimal.NewFromInt(5), decimal.NewFromInt(10), decimal.RequireFromString("1234.5699999999999999"),
				decimal.Zero, decimal.Zero, decimal.RequireFromString("0.0000000000000005")),
		},
	}
	fx := &fakeFXRepo{}
	svc := newDashboardTestService(t, pf, fx, "USD", nil)

	got, err := svc.GetDashboard(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := got.Summary
	if !equalDecimal(s.Closed.Invested, decimal.RequireFromString("21063.58")) {
		t.Fatalf("closed.invested = %v, want 21063.58", s.Closed.Invested)
	}
	if !equalDecimal(s.Closed.Proceeds, decimal.RequireFromString("27381.22")) {
		t.Fatalf("closed.proceeds = %v, want 27381.22", s.Closed.Proceeds)
	}
	if !equalDecimal(s.Closed.Realized, decimal.RequireFromString("6317.64")) {
		t.Fatalf("closed.realized = %v, want 6317.64", s.Closed.Realized)
	}
	assertDecimalInDelta(t, s.Closed.RealizedPct, decimal.NewFromInt(30), "0.01", "closed.realized_pct")
	if !equalDecimal(s.Active.Invested, decimal.RequireFromString("1234.57")) {
		t.Fatalf("active.invested = %v, want 1234.57", s.Active.Invested)
	}
	if !equalDecimal(s.Active.Value, decimal.NewFromInt(50)) || !equalDecimal(s.Active.Dividends, decimal.Zero) {
		t.Fatalf("active.value/dividends = (%v, %v), want (50, 0)", s.Active.Value, s.Active.Dividends)
	}
	if !equalDecimal(s.Active.GainLoss, decimal.RequireFromString("-1184.57")) {
		t.Fatalf("active.gain_loss = %v, want -1184.57", s.Active.GainLoss)
	}
	assertDecimalInDelta(t, s.Active.GainLossPct, decimal.RequireFromString("-95.95"), "0.01", "active.gain_loss_pct")
	ps := got.Portfolios[0]
	if !equalDecimal(ps.Active.Invested, decimal.RequireFromString("1234.57")) ||
		!equalDecimal(ps.Closed.Realized, decimal.RequireFromString("6317.64")) {
		t.Fatalf("portfolio active.invested/closed.realized = (%v, %v), want (1234.57, 6317.64)", ps.Active.Invested, ps.Closed.Realized)
	}
}

func TestGetPortfolioSummary_ActiveClosedBreakdown(t *testing.T) {
	pfID := uuid.New()
	unpricedID := uuid.New().String()
	closedID := uuid.New().String()
	unpriced := dashboardClosedHolding(pfID.String(), unpricedID, "USD",
		decimal.NewFromInt(4), decimal.Zero, decimal.NewFromInt(12663),
		decimal.Zero, decimal.Zero, decimal.NewFromInt(50))
	unpriced.HasPrice = false
	closed := dashboardClosedHolding(pfID.String(), closedID, "USD",
		decimal.Zero, decimal.Zero, decimal.RequireFromString("0.0000000000000002"),
		decimal.NewFromInt(1000), decimal.NewFromInt(1120), decimal.NewFromInt(30))
	closed.HasPrice = false
	pf := &fakePortfolioRepo{
		portfolio: &model.Portfolio{ID: pfID, Name: "Main", Currency: "USD"},
		holdings: []*model.Holding{
			// Fully open priced position: cost and dividends feed active.
			dashboardClosedHolding(pfID.String(), uuid.New().String(), "USD",
				decimal.NewFromInt(10), decimal.NewFromInt(12), decimal.NewFromInt(100),
				decimal.Zero, decimal.Zero, decimal.NewFromInt(5)),
			// Partially sold: the 5 remaining shares and their dividends stay
			// in active, the sold lots' 100 cost and 120 proceeds go to closed.
			dashboardClosedHolding(pfID.String(), uuid.New().String(), "USD",
				decimal.NewFromInt(5), decimal.NewFromInt(20), decimal.NewFromInt(250),
				decimal.NewFromInt(100), decimal.NewFromInt(120), decimal.NewFromInt(10)),
			// Open unpriced position: no comparable market value, so its cost
			// stays out of active; its dividends are real and follow the
			// still-open position.
			unpriced,
			// Fully closed position: sold lots in closed, dividends folded
			// into the proceeds, the AVCO residue cost must not leak into
			// active.
			closed,
		},
	}
	fx := &fakeFXRepo{}
	svc := newTestService(t, pf, &fakeExposureRepo{}, fx)

	got, err := svc.GetPortfolioSummary(context.Background(), pfID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// active invested = 100 + 250 (the unpriced cost is excluded);
	// active value = 10*12 + 5*20; dividends = 5 + 10 + 50
	if !equalDecimal(got.Active.Invested, decimal.NewFromInt(350)) {
		t.Fatalf("active.invested = %v, want 350", got.Active.Invested)
	}
	if !equalDecimal(got.Active.Value, decimal.NewFromInt(220)) {
		t.Fatalf("active.value = %v, want 220", got.Active.Value)
	}
	if !equalDecimal(got.Active.GainLoss, decimal.NewFromInt(-130)) {
		t.Fatalf("active.gain_loss = %v, want -130", got.Active.GainLoss)
	}
	assertDecimalInDelta(t, got.Active.GainLossPct, decimal.RequireFromString("-37.14"), "0.01", "active.gain_loss_pct")
	if !equalDecimal(got.Active.Dividends, decimal.NewFromInt(65)) {
		t.Fatalf("active.dividends = %v, want 65", got.Active.Dividends)
	}
	// closed invested = 100 + 1000; proceeds = 120 + (1120 + 30) (the fully
	// closed position folds its dividends into the proceeds); realized = 170
	if !equalDecimal(got.Closed.Invested, decimal.NewFromInt(1100)) {
		t.Fatalf("closed.invested = %v, want 1100", got.Closed.Invested)
	}
	if !equalDecimal(got.Closed.Proceeds, decimal.NewFromInt(1270)) {
		t.Fatalf("closed.proceeds = %v, want 1270", got.Closed.Proceeds)
	}
	if !equalDecimal(got.Closed.Realized, decimal.NewFromInt(170)) {
		t.Fatalf("closed.realized = %v, want 170", got.Closed.Realized)
	}
	assertDecimalInDelta(t, got.Closed.RealizedPct, decimal.RequireFromString("15.45"), "0.01", "closed.realized_pct")

	// The legacy flat figures keep their meaning: only priced open positions
	// feed the totals.
	if !equalDecimal(got.TotalCost, decimal.NewFromInt(350)) || !equalDecimal(got.TotalValue, decimal.NewFromInt(220)) {
		t.Fatalf("total cost/value = (%v, %v), want (350, 220)", got.TotalCost, got.TotalValue)
	}
	if !equalDecimal(got.GainLoss, decimal.NewFromInt(-130)) || !equalDecimal(got.UnrealizedGL, decimal.NewFromInt(-130)) || !equalDecimal(got.RealizedGL, decimal.Zero) {
		t.Fatalf("gain_loss/unrealized/realized = (%v, %v, %v), want (-130, -130, 0)", got.GainLoss, got.UnrealizedGL, got.RealizedGL)
	}
	if got.AssetCount != 4 || len(got.Holdings) != 4 {
		t.Fatalf("asset_count/holdings = (%d, %d), want (4, 4)", got.AssetCount, len(got.Holdings))
	}
	if got.FXMissingCount != 0 {
		t.Fatalf("fx_missing_count = %d, want 0", got.FXMissingCount)
	}
}

func TestGetPortfolioSummary_EmptyPortfolioHasZeroedBreakdowns(t *testing.T) {
	pfID := uuid.New()
	pf := &fakePortfolioRepo{portfolio: &model.Portfolio{ID: pfID, Name: "Empty", Currency: "USD"}}
	svc := newTestService(t, pf, &fakeExposureRepo{}, &fakeFXRepo{})

	got, err := svc.GetPortfolioSummary(context.Background(), pfID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !equalDecimal(got.Active.Invested, decimal.Zero) || !equalDecimal(got.Active.Value, decimal.Zero) ||
		!equalDecimal(got.Active.GainLoss, decimal.Zero) || !equalDecimal(got.Active.GainLossPct, decimal.Zero) ||
		!equalDecimal(got.Active.Dividends, decimal.Zero) {
		t.Fatalf("active = %+v, want all zeros", got.Active)
	}
	if !equalDecimal(got.Closed.Invested, decimal.Zero) || !equalDecimal(got.Closed.Proceeds, decimal.Zero) ||
		!equalDecimal(got.Closed.Realized, decimal.Zero) || !equalDecimal(got.Closed.RealizedPct, decimal.Zero) {
		t.Fatalf("closed = %+v, want all zeros", got.Closed)
	}
	if !equalDecimal(got.TotalCost, decimal.Zero) || !equalDecimal(got.TotalValue, decimal.Zero) {
		t.Fatalf("total cost/value = (%v, %v), want (0, 0)", got.TotalCost, got.TotalValue)
	}
}

func TestGetDashboard_InvestedAssetsAggregatesAcrossPortfolios(t *testing.T) {
	pfUSD := uuid.New()
	pfEUR := uuid.New()
	vwraID := uuid.New().String()
	bondID := uuid.New().String()
	equityID := uuid.New().String()
	jpyID := uuid.New().String()
	closedID := uuid.New().String()

	// Same USD asset held in both portfolios: one row, base-currency sums.
	vwraUSD := dashboardHolding(pfUSD.String(), vwraID, "USD",
		decimal.NewFromInt(10), decimal.NewFromInt(100), decimal.NewFromInt(800), decimal.Zero)
	vwraUSD.Ticker = "VWRA"
	vwraUSD.Name = "Vanguard All-World"
	vwraEUR := dashboardHolding(pfEUR.String(), vwraID, "USD",
		decimal.NewFromInt(5), decimal.NewFromInt(100), decimal.NewFromInt(450), decimal.Zero)
	vwraEUR.Ticker = "VWRA"
	// Open unpriced bond: carried at cost with has_price false.
	bond := dashboardHolding(pfEUR.String(), bondID, "EUR",
		decimal.NewFromInt(5), decimal.Zero, decimal.NewFromInt(12663), decimal.Zero)
	bond.Ticker = "BOND"
	bond.HasPrice = false
	// Priced EUR asset: identity conversion.
	equity := dashboardHolding(pfEUR.String(), equityID, "EUR",
		decimal.NewFromInt(1), decimal.NewFromInt(5000), decimal.NewFromInt(4000), decimal.Zero)
	equity.Ticker = "EQ"
	// Priced JPY asset with no JPY rate: the value cannot convert, so it
	// falls back to the (convertible) invested cost: no fake loss.
	jpy := dashboardHolding(pfEUR.String(), jpyID, "JPY",
		decimal.NewFromInt(1), decimal.NewFromInt(1000), decimal.NewFromInt(8), decimal.Zero)
	jpy.Ticker = "JPYEQ"
	// Fully closed position: excluded.
	closed := dashboardHolding(pfUSD.String(), closedID, "USD",
		decimal.Zero, decimal.NewFromInt(120), decimal.Zero, decimal.NewFromInt(20))
	closed.Ticker = "GONE"

	pf := &fakePortfolioRepo{
		portfolios: []*model.Portfolio{
			{ID: pfUSD, Currency: "USD"},
			{ID: pfEUR, Currency: "EUR"},
		},
		holdings: []*model.Holding{vwraUSD, vwraEUR, bond, equity, jpy, closed},
	}
	fx := &fakeFXRepo{rates: map[string]decimal.Decimal{"EUR": decimal.RequireFromString("0.9")}}
	svc := newDashboardTestService(t, pf, fx, "EUR", nil)

	got, err := svc.GetDashboard(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Value-descending order: BOND 12663 > EQ 5000 > VWRA 1350 > JPYEQ 8;
	// the closed GONE position never appears.
	wantOrder := []string{bondID, equityID, vwraID, jpyID}
	if len(got.InvestedAssets) != len(wantOrder) {
		t.Fatalf("invested_assets len = %d, want %d: %+v", len(got.InvestedAssets), len(wantOrder), got.InvestedAssets)
	}
	for i, id := range wantOrder {
		if got.InvestedAssets[i].AssetID != id {
			t.Fatalf("invested_assets[%d].asset_id = %s, want %s (value-descending order)", i, got.InvestedAssets[i].AssetID, id)
		}
	}
	byID := map[string]model.InvestedAsset{}
	for _, ia := range got.InvestedAssets {
		byID[ia.AssetID] = ia
	}
	if _, ok := byID[closedID]; ok {
		t.Fatal("closed position must not appear in invested_assets")
	}

	// VWRA: invested 800*0.9 + 450, value 10*100*0.9 + 5*100*0.9.
	v := byID[vwraID]
	if v.Ticker != "VWRA" || v.Name != "Vanguard All-World" || v.Currency != "USD" || !v.HasPrice {
		t.Fatalf("VWRA row = %+v, want ticker VWRA, name Vanguard All-World, currency USD, has_price true", v)
	}
	if !equalDecimal(v.Invested, decimal.NewFromInt(1170)) || !equalDecimal(v.Value, decimal.NewFromInt(1350)) {
		t.Fatalf("VWRA invested/value = (%v, %v), want (1170, 1350)", v.Invested, v.Value)
	}
	if !equalDecimal(v.GainLoss, decimal.NewFromInt(180)) {
		t.Fatalf("VWRA gain_loss = %v, want 180", v.GainLoss)
	}
	assertDecimalInDelta(t, v.GainLossPct, decimal.RequireFromString("15.38"), "0.01", "VWRA gain_loss_pct")

	b := byID[bondID]
	if b.HasPrice {
		t.Fatalf("BOND has_price = true, want false (unpriced asset)")
	}
	if !equalDecimal(b.Invested, decimal.NewFromInt(12663)) || !equalDecimal(b.Value, decimal.NewFromInt(12663)) ||
		!equalDecimal(b.GainLoss, decimal.Zero) || !equalDecimal(b.GainLossPct, decimal.Zero) {
		t.Fatalf("BOND row = %+v, want invested and value carried at cost 12663 with zero gain", b)
	}

	e := byID[equityID]
	if !equalDecimal(e.Invested, decimal.NewFromInt(4000)) || !equalDecimal(e.Value, decimal.NewFromInt(5000)) ||
		!equalDecimal(e.GainLoss, decimal.NewFromInt(1000)) {
		t.Fatalf("EQ row = %+v, want invested 4000 value 5000 gain 1000", e)
	}
	assertDecimalInDelta(t, e.GainLossPct, decimal.NewFromInt(25), "0.01", "EQ gain_loss_pct")

	j := byID[jpyID]
	if !j.HasPrice {
		t.Fatalf("JPYEQ has_price = false, want true (priced, only the FX is missing)")
	}
	if !equalDecimal(j.Invested, decimal.NewFromInt(8)) || !equalDecimal(j.Value, decimal.NewFromInt(8)) ||
		!equalDecimal(j.GainLoss, decimal.Zero) {
		t.Fatalf("JPYEQ row = %+v, want the unconvertible value carried at the invested cost 8", j)
	}
}

func TestGetDashboard_NoPortfoliosReturnsBaseCurrencyWithoutSummary(t *testing.T) {
	svc := newDashboardTestService(t, &fakePortfolioRepo{}, &fakeFXRepo{}, "EUR", nil)

	got, err := svc.GetDashboard(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.BaseCurrency != "EUR" {
		t.Fatalf("base_currency = %q, want EUR", got.BaseCurrency)
	}
	if got.Summary != nil {
		t.Fatalf("summary = %+v, want nil for an empty vault", got.Summary)
	}
	if got.InvestedAssets == nil || len(got.InvestedAssets) != 0 {
		t.Fatalf("invested_assets = %v, want an empty (non-nil) slice for an empty vault", got.InvestedAssets)
	}
}

func TestGetDashboardPerformance_MonthlyBucketsWithoutFlows(t *testing.T) {
	pfUSD := uuid.New()
	stockID := uuid.New().String()
	buy := time.Date(2025, 1, 5, 0, 0, 0, 0, time.UTC)
	d1 := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2025, 2, 28, 0, 0, 0, 0, time.UTC)
	pf := &fakePortfolioRepo{
		portfolios: []*model.Portfolio{{ID: pfUSD, Currency: "USD"}},
		holdings:   []*model.Holding{seriesHolding(pfUSD, stockID)},
	}
	assets := map[uuid.UUID][]model.AssetPositionSeries{
		pfUSD: {
			{AssetID: stockID, Series: []model.PositionPoint{
				{Date: d1, MarketValue: decimal.NewFromInt(1000), CostBasis: decimal.NewFromInt(1000)},
				{Date: d2, MarketValue: decimal.NewFromInt(1000), CostBasis: decimal.NewFromInt(1000)},
			}},
		},
	}
	txs := &fakeTransactionRepo{txs: []model.TransactionWithAsset{
		perfTx(pfUSD, stockID, model.TxBuy, buy, 10, 100, 0),
	}}
	svc := newDashboardTestServiceWithTxs(t, pf, &fakeFXRepo{}, "USD", assets, txs)

	got, err := svc.GetDashboardPerformance(context.Background(), uuid.New(), "month")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Currency != "USD" || got.Granularity != "month" {
		t.Fatalf("currency/granularity = (%q, %q), want (USD, month)", got.Currency, got.Granularity)
	}
	// The January deposit is the vault's first flow with no previous value
	// (V(d−1) = 0): its day and the January observation are both skipped, so
	// the first bucket reports 0% while invested and value are populated.
	// February holds flat (V = 1000 with no flows): every daily return, and
	// hence the bucket return and the compounded twr, is 0.
	assertPerfBuckets(t, got.Buckets, []model.PerformanceBucket{
		{Period: "2025-01", Return: decimal.Zero, TWR: decimal.Zero, Invested: decimal.NewFromInt(1000), Value: decimal.NewFromInt(1000)},
		{Period: "2025-02", Return: decimal.Zero, TWR: decimal.Zero, Invested: decimal.NewFromInt(1000), Value: decimal.NewFromInt(1000)},
	})
}

func TestGetDashboardPerformance_CanonicalMidPeriodSale(t *testing.T) {
	pfUSD := uuid.New()
	stockID := uuid.New().String()
	buy := time.Date(2025, 1, 5, 0, 0, 0, 0, time.UTC)
	sell := time.Date(2025, 2, 14, 0, 0, 0, 0, time.UTC)
	d1 := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2025, 2, 14, 0, 0, 0, 0, time.UTC)
	d3 := time.Date(2025, 2, 28, 0, 0, 0, 0, time.UTC)
	pf := &fakePortfolioRepo{
		portfolios: []*model.Portfolio{{ID: pfUSD, Currency: "USD"}},
		holdings:   []*model.Holding{seriesHolding(pfUSD, stockID)},
	}
	// 100 shares at an average cost of 3: the month starts at MV 1000
	// (price 10), 50 shares are sold mid-February at 12 (MV 600 after the
	// sale, the realized 450 the series also carries must NOT enter V), and
	// the month ends at MV 750 (the remaining 50 shares at price 15).
	assets := map[uuid.UUID][]model.AssetPositionSeries{
		pfUSD: {
			{AssetID: stockID, Series: []model.PositionPoint{
				{Date: d1, MarketValue: decimal.NewFromInt(1000), CostBasis: decimal.NewFromInt(300)},
				{Date: d2, MarketValue: decimal.NewFromInt(600), CostBasis: decimal.NewFromInt(150), Realized: decimal.NewFromInt(450)},
				{Date: d3, MarketValue: decimal.NewFromInt(750), CostBasis: decimal.NewFromInt(150), Realized: decimal.NewFromInt(450)},
			}},
		},
	}
	txs := &fakeTransactionRepo{txs: []model.TransactionWithAsset{
		perfTx(pfUSD, stockID, model.TxBuy, buy, 100, 10, 0),
		perfTx(pfUSD, stockID, model.TxSell, sell, 50, 12, 0),
	}}
	svc := newDashboardTestServiceWithTxs(t, pf, &fakeFXRepo{}, "USD", assets, txs)

	got, err := svc.GetDashboardPerformance(context.Background(), uuid.New(), "month")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The pure TWR of the 10 → 12 → 15 price path: the sale day measures
	// (600 − 1000 + 600)/1000 = +20% on the full pre-sale value and the
	// rest of February (750 − 600)/600 = +25% on the remainder, linked
	// geometrically to 1.2 · 1.25 − 1 = +50%. invested nets the withdrawal
	// (1000 − 600) and value is the market value only (750, no realized).
	assertPerfBuckets(t, got.Buckets, []model.PerformanceBucket{
		{Period: "2025-01", Return: decimal.Zero, TWR: decimal.Zero, Invested: decimal.NewFromInt(1000), Value: decimal.NewFromInt(1000)},
		{Period: "2025-02", Return: decimal.NewFromInt(50), TWR: decimal.NewFromInt(50), Invested: decimal.NewFromInt(400), Value: decimal.NewFromInt(750)},
	})
}

func TestGetDashboardPerformance_MidPeriodBuyIsTimeWeighted(t *testing.T) {
	pfUSD := uuid.New()
	stockID := uuid.New().String()
	buy1 := time.Date(2025, 1, 5, 0, 0, 0, 0, time.UTC)
	buy2 := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)
	d1 := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)
	d3 := time.Date(2025, 2, 9, 0, 0, 0, 0, time.UTC)
	pf := &fakePortfolioRepo{
		portfolios: []*model.Portfolio{{ID: pfUSD, Currency: "USD"}},
		holdings:   []*model.Holding{seriesHolding(pfUSD, stockID)},
	}
	// February starts at MV 1000 (100 shares at price 10), on the 1st the
	// price drops to 9 and 10 more shares are bought (MV 990 after the
	// deposit, +90 flow), and the bucket ends on the 9th at price 8:
	// MV = 110 shares * 8 = 880.
	assets := map[uuid.UUID][]model.AssetPositionSeries{
		pfUSD: {
			{AssetID: stockID, Series: []model.PositionPoint{
				{Date: d1, MarketValue: decimal.NewFromInt(1000), CostBasis: decimal.NewFromInt(1000)},
				{Date: d2, MarketValue: decimal.NewFromInt(990), CostBasis: decimal.NewFromInt(1090)},
				{Date: d3, MarketValue: decimal.NewFromInt(880), CostBasis: decimal.NewFromInt(1090)},
			}},
		},
	}
	txs := &fakeTransactionRepo{txs: []model.TransactionWithAsset{
		perfTx(pfUSD, stockID, model.TxBuy, buy1, 100, 10, 0),
		perfTx(pfUSD, stockID, model.TxBuy, buy2, 10, 9, 0),
	}}
	svc := newDashboardTestServiceWithTxs(t, pf, &fakeFXRepo{}, "USD", assets, txs)

	got, err := svc.GetDashboardPerformance(context.Background(), uuid.New(), "month")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The deposit is isolated out of the return: the buy day measures
	// (990 − 1000 − 90)/1000 = −10% (the drop on the pre-buy value) and the
	// rest of the month (880 − 990)/990 = −11⅑%, so the linked February
	// return is the true TWR of the 10 → 9 → 8 path: 0.9 · 8/9 − 1 = −20%.
	// invested still tracks the full cash in (1000 + 90).
	assertPerfBuckets(t, got.Buckets, []model.PerformanceBucket{
		{Period: "2025-01", Return: decimal.Zero, TWR: decimal.Zero, Invested: decimal.NewFromInt(1000), Value: decimal.NewFromInt(1000)},
		{Period: "2025-02", Return: decimal.NewFromInt(-20), TWR: decimal.NewFromInt(-20), Invested: decimal.NewFromInt(1090), Value: decimal.NewFromInt(880)},
	})
}

func TestGetDashboardPerformance_DividendsAddYieldWithoutChangingInvested(t *testing.T) {
	pfUSD := uuid.New()
	stockID := uuid.New().String()
	buy := time.Date(2025, 1, 5, 0, 0, 0, 0, time.UTC)
	div := time.Date(2025, 2, 10, 0, 0, 0, 0, time.UTC)
	d1 := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2025, 2, 10, 0, 0, 0, 0, time.UTC)
	d3 := time.Date(2025, 2, 28, 0, 0, 0, 0, time.UTC)
	pf := &fakePortfolioRepo{
		portfolios: []*model.Portfolio{{ID: pfUSD, Currency: "USD"}},
		holdings:   []*model.Holding{seriesHolding(pfUSD, stockID)},
	}
	// The market value is unaffected by the distribution: February ends
	// where it started at MV 1000.
	assets := map[uuid.UUID][]model.AssetPositionSeries{
		pfUSD: {
			{AssetID: stockID, Series: []model.PositionPoint{
				{Date: d1, MarketValue: decimal.NewFromInt(1000), CostBasis: decimal.NewFromInt(1000)},
				{Date: d2, MarketValue: decimal.NewFromInt(1000), CostBasis: decimal.NewFromInt(1000)},
				{Date: d3, MarketValue: decimal.NewFromInt(1000), CostBasis: decimal.NewFromInt(1000)},
			}},
		},
	}
	txs := &fakeTransactionRepo{txs: []model.TransactionWithAsset{
		perfTx(pfUSD, stockID, model.TxBuy, buy, 100, 10, 0),
		perfTx(pfUSD, stockID, model.TxDividend, div, 10, 2, 0),
	}}
	svc := newDashboardTestServiceWithTxs(t, pf, &fakeFXRepo{}, "USD", assets, txs)

	got, err := svc.GetDashboardPerformance(context.Background(), uuid.New(), "month")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The dividend is income taken out of the vault: a −20 withdrawal that
	// the value never receives back, so the pay day measures
	// (1000 − 1000 + 20)/1000 = +2% of pure yield — a positive return
	// contribution with a flat market value — while the capital invested
	// stays at the 1000 originally deposited.
	assertPerfBuckets(t, got.Buckets, []model.PerformanceBucket{
		{Period: "2025-01", Return: decimal.Zero, TWR: decimal.Zero, Invested: decimal.NewFromInt(1000), Value: decimal.NewFromInt(1000)},
		{Period: "2025-02", Return: decimal.NewFromInt(2), TWR: decimal.NewFromInt(2), Invested: decimal.NewFromInt(1000), Value: decimal.NewFromInt(1000)},
	})
}

func TestGetDashboardPerformance_UnpricedBondCarriedAtCost(t *testing.T) {
	pfUSD := uuid.New()
	stockID := uuid.New().String()
	bondID := uuid.New().String()
	buyStock := time.Date(2025, 1, 5, 0, 0, 0, 0, time.UTC)
	buyBond := time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC)
	sellBond := time.Date(2025, 2, 10, 0, 0, 0, 0, time.UTC)
	d1 := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2025, 2, 5, 0, 0, 0, 0, time.UTC)
	d3 := time.Date(2025, 2, 10, 0, 0, 0, 0, time.UTC)
	d4 := time.Date(2025, 2, 28, 0, 0, 0, 0, time.UTC)
	stock := seriesHolding(pfUSD, stockID)
	bond := seriesHolding(pfUSD, bondID)
	bond.HasPrice = false
	pf := &fakePortfolioRepo{
		portfolios: []*model.Portfolio{{ID: pfUSD, Currency: "USD"}},
		holdings:   []*model.Holding{stock, bond},
	}
	// The bond has no price row: it is carried at its 900 cost basis while
	// held (MV 1000 + cost 900 = 1900 before and on February the 5th), then
	// 5 of the 10 units are sold at 100 on February the 10th: the −500
	// withdrawal meets a 450 remaining cost, so the 50 gain (proceeds −
	// sold cost) surfaces exactly on the sale day and never before.
	assets := map[uuid.UUID][]model.AssetPositionSeries{
		pfUSD: {
			{AssetID: stockID, Series: []model.PositionPoint{
				{Date: d1, MarketValue: decimal.NewFromInt(1000), CostBasis: decimal.NewFromInt(1000)},
				{Date: d2, MarketValue: decimal.NewFromInt(1000), CostBasis: decimal.NewFromInt(1000)},
				{Date: d3, MarketValue: decimal.NewFromInt(1000), CostBasis: decimal.NewFromInt(1000)},
				{Date: d4, MarketValue: decimal.NewFromInt(1000), CostBasis: decimal.NewFromInt(1000)},
			}},
			{AssetID: bondID, Series: []model.PositionPoint{
				{Date: d1, CostBasis: decimal.NewFromInt(900)},
				{Date: d2, CostBasis: decimal.NewFromInt(900)},
				{Date: d3, CostBasis: decimal.NewFromInt(450), Realized: decimal.NewFromInt(50)},
				{Date: d4, CostBasis: decimal.NewFromInt(450), Realized: decimal.NewFromInt(50)},
			}},
		},
	}
	txs := &fakeTransactionRepo{txs: []model.TransactionWithAsset{
		perfTx(pfUSD, stockID, model.TxBuy, buyStock, 100, 10, 0),
		perfTx(pfUSD, bondID, model.TxBuy, buyBond, 10, 90, 0),
		perfTx(pfUSD, bondID, model.TxSell, sellBond, 5, 100, 0),
	}}
	svc := newDashboardTestServiceWithTxs(t, pf, &fakeFXRepo{}, "USD", assets, txs)

	got, err := svc.GetDashboardPerformance(context.Background(), uuid.New(), "month")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// January: the first bucket (V(d−1) = 0 everywhere) is 0 with invested
	// 1900 and value at the full market value, cost of the bond included.
	// February: flat 0 through February the 5th, then the sale day alone
	// moves the return: (1450 − 1900 + 500)/1900 = 50/1900 = +2.6316%.
	assertPerfBuckets(t, got.Buckets, []model.PerformanceBucket{
		{Period: "2025-01", Return: decimal.Zero, TWR: decimal.Zero, Invested: decimal.NewFromInt(1900), Value: decimal.NewFromInt(1900)},
		{Period: "2025-02", Return: decimal.RequireFromString("2.6316"), TWR: decimal.RequireFromString("2.6316"), Invested: decimal.NewFromInt(1400), Value: decimal.NewFromInt(1450)},
	})
}

func TestGetDashboardPerformance_FullLiquidationThenReopen(t *testing.T) {
	pfUSD := uuid.New()
	stockID := uuid.New().String()
	buy := time.Date(2025, 1, 5, 0, 0, 0, 0, time.UTC)
	sell := time.Date(2025, 6, 10, 0, 0, 0, 0, time.UTC)
	reopen := time.Date(2025, 9, 15, 0, 0, 0, 0, time.UTC)
	d1 := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2025, 6, 9, 0, 0, 0, 0, time.UTC)
	d3 := time.Date(2025, 6, 10, 0, 0, 0, 0, time.UTC)
	d4 := time.Date(2025, 6, 30, 0, 0, 0, 0, time.UTC)
	d5 := time.Date(2025, 9, 15, 0, 0, 0, 0, time.UTC)
	d6 := time.Date(2025, 9, 30, 0, 0, 0, 0, time.UTC)
	d7 := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	pf := &fakePortfolioRepo{
		portfolios: []*model.Portfolio{{ID: pfUSD, Currency: "USD"}},
		holdings:   []*model.Holding{seriesHolding(pfUSD, stockID)},
	}
	// The AAPL pattern: 10 shares bought at 100, the whole position sold at
	// 105 in June (the 50 gain is priced in on the run-up, the 1050
	// withdrawal is a flow), the vault sits empty, then the same position is
	// reopened at 105 in September and closes the year at 110. The series
	// carries the 50 of realized from the June sale: V must ignore it.
	assets := map[uuid.UUID][]model.AssetPositionSeries{
		pfUSD: {
			{AssetID: stockID, Series: []model.PositionPoint{
				{Date: d1, MarketValue: decimal.NewFromInt(1000), CostBasis: decimal.NewFromInt(1000)},
				{Date: d2, MarketValue: decimal.NewFromInt(1050), CostBasis: decimal.NewFromInt(1000)},
				{Date: d3, MarketValue: decimal.Zero, CostBasis: decimal.Zero, Realized: decimal.NewFromInt(50)},
				{Date: d4, MarketValue: decimal.Zero, CostBasis: decimal.Zero, Realized: decimal.NewFromInt(50)},
				{Date: d5, MarketValue: decimal.NewFromInt(1050), CostBasis: decimal.NewFromInt(1050), Realized: decimal.NewFromInt(50)},
				{Date: d6, MarketValue: decimal.NewFromInt(1050), CostBasis: decimal.NewFromInt(1050), Realized: decimal.NewFromInt(50)},
				{Date: d7, MarketValue: decimal.NewFromInt(1100), CostBasis: decimal.NewFromInt(1050), Realized: decimal.NewFromInt(50)},
			}},
		},
	}
	txs := &fakeTransactionRepo{txs: []model.TransactionWithAsset{
		perfTx(pfUSD, stockID, model.TxBuy, buy, 10, 100, 0),
		perfTx(pfUSD, stockID, model.TxSell, sell, 10, 105, 0),
		perfTx(pfUSD, stockID, model.TxBuy, reopen, 10, 105, 0),
	}}
	svc := newDashboardTestServiceWithTxs(t, pf, &fakeFXRepo{}, "USD", assets, txs)

	got, err := svc.GetDashboardPerformance(context.Background(), uuid.New(), "month")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// No ±170%/−49% residue from the liquidation/reopen: selling at market
	// is return-neutral ((0 − 1050 + 1050)/1050 = 0 on the sale day), the
	// empty gap measures nothing (V(d−1) = 0 → skipped) and the reopen is
	// isolated the same way. Only the real 100 → 105 → 110 price path is
	// reported: +5% in June, 0% in September, 50/1050 = +4.7619% in
	// December, compounded to a 1.05 · 1.047619 − 1 = +10% cumulative twr.
	assertPerfBuckets(t, got.Buckets, []model.PerformanceBucket{
		{Period: "2025-01", Return: decimal.Zero, TWR: decimal.Zero, Invested: decimal.NewFromInt(1000), Value: decimal.NewFromInt(1000)},
		{Period: "2025-06", Return: decimal.NewFromInt(5), TWR: decimal.NewFromInt(5), Invested: decimal.NewFromInt(-50), Value: decimal.Zero},
		{Period: "2025-09", Return: decimal.Zero, TWR: decimal.NewFromInt(5), Invested: decimal.NewFromInt(1000), Value: decimal.NewFromInt(1050)},
		{Period: "2025-12", Return: decimal.RequireFromString("4.7619"), TWR: decimal.NewFromInt(10), Invested: decimal.NewFromInt(1000), Value: decimal.NewFromInt(1100)},
	})
	for _, b := range got.Buckets {
		if b.Return.GreaterThan(decimal.NewFromInt(10)) || b.Return.LessThan(decimal.NewFromInt(-10)) {
			t.Fatalf("bucket %s return = %v, want within ±10%% (no liquidation artifacts)", b.Period, b.Return)
		}
	}
}

func TestGetDashboardPerformance_TWRCompoundsAcrossBuckets(t *testing.T) {
	pfUSD := uuid.New()
	stockID := uuid.New().String()
	buy := time.Date(2025, 1, 5, 0, 0, 0, 0, time.UTC)
	d1 := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2025, 2, 28, 0, 0, 0, 0, time.UTC)
	d3 := time.Date(2025, 3, 31, 0, 0, 0, 0, time.UTC)
	pf := &fakePortfolioRepo{
		portfolios: []*model.Portfolio{{ID: pfUSD, Currency: "USD"}},
		holdings:   []*model.Holding{seriesHolding(pfUSD, stockID)},
	}
	assets := map[uuid.UUID][]model.AssetPositionSeries{
		pfUSD: {
			{AssetID: stockID, Series: []model.PositionPoint{
				{Date: d1, MarketValue: decimal.NewFromInt(1000), CostBasis: decimal.NewFromInt(1000)},
				{Date: d2, MarketValue: decimal.NewFromInt(1500), CostBasis: decimal.NewFromInt(1000)},
				{Date: d3, MarketValue: decimal.NewFromInt(1650), CostBasis: decimal.NewFromInt(1000)},
			}},
		},
	}
	txs := &fakeTransactionRepo{txs: []model.TransactionWithAsset{
		perfTx(pfUSD, stockID, model.TxBuy, buy, 100, 10, 0),
	}}
	svc := newDashboardTestServiceWithTxs(t, pf, &fakeFXRepo{}, "USD", assets, txs)

	got, err := svc.GetDashboardPerformance(context.Background(), uuid.New(), "month")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// No flows after the first deposit: February returns +50% and March
	// +10% on the grown value, and the cumulative twr compounds them:
	// (1.5 * 1.1 - 1) = +65%, not 50 + 10.
	assertPerfBuckets(t, got.Buckets, []model.PerformanceBucket{
		{Period: "2025-01", Return: decimal.Zero, TWR: decimal.Zero, Invested: decimal.NewFromInt(1000), Value: decimal.NewFromInt(1000)},
		{Period: "2025-02", Return: decimal.NewFromInt(50), TWR: decimal.NewFromInt(50), Invested: decimal.NewFromInt(1000), Value: decimal.NewFromInt(1500)},
		{Period: "2025-03", Return: decimal.NewFromInt(10), TWR: decimal.NewFromInt(65), Invested: decimal.NewFromInt(1000), Value: decimal.NewFromInt(1650)},
	})
}

func TestGetDashboardPerformance_FirstBucketAndStandaloneFees(t *testing.T) {
	pfUSD := uuid.New()
	stockID := uuid.New().String()
	buy := time.Date(2025, 2, 10, 0, 0, 0, 0, time.UTC)
	lotFee := time.Date(2025, 2, 12, 0, 0, 0, 0, time.UTC)
	flatFee := time.Date(2025, 2, 20, 0, 0, 0, 0, time.UTC)
	d1 := time.Date(2025, 2, 28, 0, 0, 0, 0, time.UTC)
	pf := &fakePortfolioRepo{
		portfolios: []*model.Portfolio{{ID: pfUSD, Currency: "USD"}},
		holdings:   []*model.Holding{seriesHolding(pfUSD, stockID)},
	}
	assets := map[uuid.UUID][]model.AssetPositionSeries{
		pfUSD: {
			{AssetID: stockID, Series: []model.PositionPoint{
				{Date: d1, MarketValue: decimal.NewFromInt(600), CostBasis: decimal.NewFromInt(500)},
			}},
		},
	}
	// 500 in, plus a per-lot fee (qty 2 at 5 → +10) and a flat fee
	// (qty 0 → the price alone, +20).
	txs := &fakeTransactionRepo{txs: []model.TransactionWithAsset{
		perfTx(pfUSD, stockID, model.TxBuy, buy, 10, 50, 0),
		perfTx(pfUSD, stockID, model.TxFee, lotFee, 2, 5, 0),
		perfTx(pfUSD, stockID, model.TxFee, flatFee, 0, 20, 0),
	}}
	svc := newDashboardTestServiceWithTxs(t, pf, &fakeFXRepo{}, "USD", assets, txs)

	got, err := svc.GetDashboardPerformance(context.Background(), uuid.New(), "month")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The only bucket is the first one: every day has V(d−1) = 0 and is
	// skipped, so even the +20% appreciation of this single deposit period
	// reports return/twr 0 while invested (500 + 10 + 20) and value are
	// populated.
	assertPerfBuckets(t, got.Buckets, []model.PerformanceBucket{
		{Period: "2025-02", Return: decimal.Zero, TWR: decimal.Zero, Invested: decimal.NewFromInt(530), Value: decimal.NewFromInt(600)},
	})
}

func TestGetDashboardPerformance_YearlyBuckets(t *testing.T) {
	pfUSD := uuid.New()
	stockID := uuid.New().String()
	buy := time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC)
	d1 := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2025, 6, 10, 0, 0, 0, 0, time.UTC)
	pf := &fakePortfolioRepo{
		portfolios: []*model.Portfolio{{ID: pfUSD, Currency: "USD"}},
		holdings:   []*model.Holding{seriesHolding(pfUSD, stockID)},
	}
	assets := map[uuid.UUID][]model.AssetPositionSeries{
		pfUSD: {
			{AssetID: stockID, Series: []model.PositionPoint{
				{Date: d1, MarketValue: decimal.NewFromInt(1200), CostBasis: decimal.NewFromInt(1000), Realized: decimal.NewFromInt(60)},
				{Date: d2, MarketValue: decimal.NewFromInt(1400), CostBasis: decimal.NewFromInt(1000), Realized: decimal.NewFromInt(60)},
			}},
		},
	}
	txs := &fakeTransactionRepo{txs: []model.TransactionWithAsset{
		perfTx(pfUSD, stockID, model.TxBuy, buy, 100, 10, 0),
	}}
	svc := newDashboardTestServiceWithTxs(t, pf, &fakeFXRepo{}, "USD", assets, txs)

	got, err := svc.GetDashboardPerformance(context.Background(), uuid.New(), "year")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Granularity != "year" {
		t.Fatalf("granularity = %q, want year", got.Granularity)
	}
	// 2024 is the first bucket (V(d−1) = 0 → every day skipped → return 0).
	// 2025: the market value alone drives the return, the stored realized
	// stays out of V: (1400 − 1200)/1200 = +16.6667%.
	assertPerfBuckets(t, got.Buckets, []model.PerformanceBucket{
		{Period: "2024", Return: decimal.Zero, TWR: decimal.Zero, Invested: decimal.NewFromInt(1000), Value: decimal.NewFromInt(1200)},
		{Period: "2025", Return: decimal.RequireFromString("16.6667"), TWR: decimal.RequireFromString("16.6667"), Invested: decimal.NewFromInt(1000), Value: decimal.NewFromInt(1400)},
	})
}

func TestGetDashboardPerformance_AggregatesPortfoliosInBaseCurrency(t *testing.T) {
	pfUSD := uuid.New()
	pfGBP := uuid.New()
	usdAssetID := uuid.New().String()
	gbpAssetID := uuid.New().String()
	buyUSD := time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC)
	buyGBP := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	sellUSD := time.Date(2025, 2, 10, 0, 0, 0, 0, time.UTC)
	d1 := time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2025, 2, 10, 0, 0, 0, 0, time.UTC)
	d3 := time.Date(2025, 2, 28, 0, 0, 0, 0, time.UTC)
	gbpHolding := seriesHolding(pfGBP, gbpAssetID)
	gbpHolding.Currency = "GBP"
	pf := &fakePortfolioRepo{
		portfolios: []*model.Portfolio{
			{ID: pfUSD, Currency: "USD"},
			{ID: pfGBP, Currency: "GBP"},
		},
		holdings: []*model.Holding{seriesHolding(pfUSD, usdAssetID), gbpHolding},
	}
	fx := &fakeFXRepo{
		rates: map[string]decimal.Decimal{
			"EUR": decimal.RequireFromString("0.5"),
			"GBP": decimal.RequireFromString("0.25"),
		},
	}
	// The USD asset is priced through every observation; the GBP asset has
	// no point on February the 10th, so its last converted value (the 400
	// of January the 20th) carries over to that day.
	assets := map[uuid.UUID][]model.AssetPositionSeries{
		pfUSD: {
			{AssetID: usdAssetID, Series: []model.PositionPoint{
				{Date: d1, MarketValue: decimal.NewFromInt(1000), CostBasis: decimal.NewFromInt(1000)},
				{Date: d2, MarketValue: decimal.NewFromInt(900), CostBasis: decimal.NewFromInt(900)},
				{Date: d3, MarketValue: decimal.NewFromInt(990), CostBasis: decimal.NewFromInt(900)},
			}},
		},
		pfGBP: {
			{AssetID: gbpAssetID, Series: []model.PositionPoint{
				{Date: d1, MarketValue: decimal.NewFromInt(200), CostBasis: decimal.NewFromInt(200)},
				{Date: d3, MarketValue: decimal.NewFromInt(240), CostBasis: decimal.NewFromInt(200)},
			}},
		},
	}
	// Flows are converted in the asset currency at the FX of the
	// transaction date: +1000 USD → +500, +200 GBP → +400 (GBP→EUR cross
	// 0.5/0.25 = 2), −100 USD → −50.
	txs := &fakeTransactionRepo{txs: []model.TransactionWithAsset{
		perfTx(pfUSD, usdAssetID, model.TxBuy, buyUSD, 10, 100, 0),
		perfTx(pfGBP, gbpAssetID, model.TxBuy, buyGBP, 10, 20, 0),
		perfTx(pfUSD, usdAssetID, model.TxSell, sellUSD, 1, 100, 0),
	}}
	svc := newDashboardTestServiceWithTxs(t, pf, fx, "EUR", assets, txs)

	got, err := svc.GetDashboardPerformance(context.Background(), uuid.New(), "month")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Currency != "EUR" {
		t.Fatalf("currency = %q, want EUR", got.Currency)
	}
	// January: MV 1000*0.5 + 200*2 = 900 = invested (500 + 400), first
	// bucket → return 0. February: the sale day is return-neutral
	// ((850 − 900 + 50)/900 = 0, sold at market) and the month ends at
	// MV 900*0.5 + 240*2 = 975 → (975 − 850)/850 = +14.7059%, invested 850.
	assertPerfBuckets(t, got.Buckets, []model.PerformanceBucket{
		{Period: "2025-01", Return: decimal.Zero, TWR: decimal.Zero, Invested: decimal.NewFromInt(900), Value: decimal.NewFromInt(900)},
		{Period: "2025-02", Return: decimal.RequireFromString("14.7059"), TWR: decimal.RequireFromString("14.7059"), Invested: decimal.NewFromInt(850), Value: decimal.NewFromInt(975)},
	})
}

func TestGetDashboardPerformance_MissingFXForwardFillsToZero(t *testing.T) {
	pfUSD := uuid.New()
	stockID := uuid.New().String()
	buyJan := time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC)
	buyFeb := time.Date(2025, 2, 5, 0, 0, 0, 0, time.UTC)
	feb1 := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)
	d1 := time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2025, 2, 10, 0, 0, 0, 0, time.UTC)
	pf := &fakePortfolioRepo{
		portfolios: []*model.Portfolio{{ID: pfUSD, Currency: "USD"}},
		holdings:   []*model.Holding{seriesHolding(pfUSD, stockID)},
	}
	// No EUR snapshot at all and the first history point only covers
	// February: January cannot be converted, its totals stay zero and its
	// flow is skipped entirely (it never enters invested, not even later).
	fx := &fakeFXRepo{
		history: map[string][]model.FXRatePoint{
			"EUR": {{Date: feb1, Rate: decimal.RequireFromString("0.9")}},
		},
	}
	assets := map[uuid.UUID][]model.AssetPositionSeries{
		pfUSD: {
			{AssetID: stockID, Series: []model.PositionPoint{
				{Date: d1, MarketValue: decimal.NewFromInt(1000), CostBasis: decimal.NewFromInt(1000)},
				{Date: d2, MarketValue: decimal.NewFromInt(1100), CostBasis: decimal.NewFromInt(1000), Realized: decimal.NewFromInt(20)},
			}},
		},
	}
	txs := &fakeTransactionRepo{txs: []model.TransactionWithAsset{
		perfTx(pfUSD, stockID, model.TxBuy, buyJan, 10, 100, 0),
		perfTx(pfUSD, stockID, model.TxBuy, buyFeb, 1, 100, 0),
	}}
	svc := newDashboardTestServiceWithTxs(t, pf, fx, "EUR", assets, txs)

	got, err := svc.GetDashboardPerformance(context.Background(), uuid.New(), "month")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// January: unconvertible → zeros everywhere. February converts at 0.9:
	// value 1100*0.9 = 990 (the stored realized stays out of V), invested
	// only the February deposit 100*0.9 = 90 (the January one was skipped
	// for the missing FX), and the return stays 0 because every day has an
	// unconvertible (hence zero) V(d−1).
	assertPerfBuckets(t, got.Buckets, []model.PerformanceBucket{
		{Period: "2025-01", Return: decimal.Zero, TWR: decimal.Zero, Invested: decimal.Zero, Value: decimal.Zero},
		{Period: "2025-02", Return: decimal.Zero, TWR: decimal.Zero, Invested: decimal.NewFromInt(90), Value: decimal.NewFromInt(990)},
	})
}

func TestGetDashboardPerformance_EmptyVaultHasNoBuckets(t *testing.T) {
	svc := newDashboardTestService(t, &fakePortfolioRepo{}, &fakeFXRepo{}, "EUR", nil)

	got, err := svc.GetDashboardPerformance(context.Background(), uuid.New(), "month")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Currency != "EUR" || got.Granularity != "month" {
		t.Fatalf("currency/granularity = (%q, %q), want (EUR, month)", got.Currency, got.Granularity)
	}
	if got.Buckets == nil || len(got.Buckets) != 0 {
		t.Fatalf("buckets = %+v, want an empty non-nil slice", got.Buckets)
	}
}

func TestGetDashboardPerformance_InvalidGranularity(t *testing.T) {
	svc := newDashboardTestService(t, &fakePortfolioRepo{}, &fakeFXRepo{}, "EUR", nil)

	for _, g := range []string{"", "week", "MONTH"} {
		if _, err := svc.GetDashboardPerformance(context.Background(), uuid.New(), g); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("granularity %q: err = %v, want ErrInvalidInput", g, err)
		}
	}
}

// newPortfolioPerfTestService wires a single-portfolio performance-buckets
// service: the portfolio row carries owner so the authz path can be checked,
// the series map is keyed by its id and the ledger holds its transactions.
func newPortfolioPerfTestService(t *testing.T, p *fakePortfolioRepo, fx *fakeFXRepo, assets map[uuid.UUID][]model.AssetPositionSeries, txs *fakeTransactionRepo) *Service {
	t.Helper()
	repos := &repository.Repository{
		Asset:       &fakeAssetRepo{},
		User:        &fakeUserRepo{user: &model.User{ID: uuid.New(), Email: "u@example.com"}},
		Portfolio:   p,
		Exposure:    &fakeExposureRepo{},
		FX:          fx,
		Series:      &fakeSeriesRepo{assets: assets},
		Transaction: txs,
	}
	return New(repos, nil, nil, nil, time.Minute, time.Hour, cache.New(nil), 0, 0, nil)
}

// perfPortfolio is the owner-scoped portfolio fixture the buckets methods
// look up by id before anything else.
func perfPortfolio(owner uuid.UUID, id uuid.UUID, currency string) *model.Portfolio {
	return &model.Portfolio{ID: id, UserID: owner, Currency: currency}
}

func TestGetPortfolioPerformanceBuckets_MonthlyBucketsWithoutFlows(t *testing.T) {
	owner := uuid.New()
	pfUSD := uuid.New()
	stockID := uuid.New().String()
	buy := time.Date(2025, 1, 5, 0, 0, 0, 0, time.UTC)
	d1 := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2025, 2, 28, 0, 0, 0, 0, time.UTC)
	pf := &fakePortfolioRepo{
		portfolio: perfPortfolio(owner, pfUSD, "USD"),
		holdings:  []*model.Holding{seriesHolding(pfUSD, stockID)},
	}
	assets := map[uuid.UUID][]model.AssetPositionSeries{
		pfUSD: {
			{AssetID: stockID, Series: []model.PositionPoint{
				{Date: d1, MarketValue: decimal.NewFromInt(1000), CostBasis: decimal.NewFromInt(1000)},
				{Date: d2, MarketValue: decimal.NewFromInt(1000), CostBasis: decimal.NewFromInt(1000)},
			}},
		},
	}
	txs := &fakeTransactionRepo{txs: []model.TransactionWithAsset{
		perfTx(pfUSD, stockID, model.TxBuy, buy, 10, 100, 0),
	}}
	svc := newPortfolioPerfTestService(t, pf, &fakeFXRepo{}, assets, txs)

	got, err := svc.GetPortfolioPerformanceBuckets(context.Background(), pfUSD, owner, "month")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Currency != "USD" || got.Granularity != "month" {
		t.Fatalf("currency/granularity = (%q, %q), want (USD, month)", got.Currency, got.Granularity)
	}
	// Same vault-wide model at portfolio scale: the first deposit day has
	// V(d−1) = 0 and is skipped, February holds flat, so both buckets
	// report 0% while invested and value are populated.
	assertPerfBuckets(t, got.Buckets, []model.PerformanceBucket{
		{Period: "2025-01", Return: decimal.Zero, TWR: decimal.Zero, Invested: decimal.NewFromInt(1000), Value: decimal.NewFromInt(1000)},
		{Period: "2025-02", Return: decimal.Zero, TWR: decimal.Zero, Invested: decimal.NewFromInt(1000), Value: decimal.NewFromInt(1000)},
	})
}

func TestGetPortfolioPerformanceBuckets_CanonicalMidPeriodSale(t *testing.T) {
	owner := uuid.New()
	pfUSD := uuid.New()
	stockID := uuid.New().String()
	buy := time.Date(2025, 1, 5, 0, 0, 0, 0, time.UTC)
	sell := time.Date(2025, 2, 14, 0, 0, 0, 0, time.UTC)
	d1 := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2025, 2, 14, 0, 0, 0, 0, time.UTC)
	d3 := time.Date(2025, 2, 28, 0, 0, 0, 0, time.UTC)
	pf := &fakePortfolioRepo{
		portfolio: perfPortfolio(owner, pfUSD, "USD"),
		holdings:  []*model.Holding{seriesHolding(pfUSD, stockID)},
	}
	assets := map[uuid.UUID][]model.AssetPositionSeries{
		pfUSD: {
			{AssetID: stockID, Series: []model.PositionPoint{
				{Date: d1, MarketValue: decimal.NewFromInt(1000), CostBasis: decimal.NewFromInt(300)},
				{Date: d2, MarketValue: decimal.NewFromInt(600), CostBasis: decimal.NewFromInt(150), Realized: decimal.NewFromInt(450)},
				{Date: d3, MarketValue: decimal.NewFromInt(750), CostBasis: decimal.NewFromInt(150), Realized: decimal.NewFromInt(450)},
			}},
		},
	}
	txs := &fakeTransactionRepo{txs: []model.TransactionWithAsset{
		perfTx(pfUSD, stockID, model.TxBuy, buy, 100, 10, 0),
		perfTx(pfUSD, stockID, model.TxSell, sell, 50, 12, 0),
	}}
	svc := newPortfolioPerfTestService(t, pf, &fakeFXRepo{}, assets, txs)

	got, err := svc.GetPortfolioPerformanceBuckets(context.Background(), pfUSD, owner, "month")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The withdrawal cannot distort the return: the sale day measures
	// (600 − 1000 + 600)/1000 = +20% on the full pre-sale value and the
	// rest of February (750 − 600)/600 = +25% on the remainder, linked to
	// 1.2 · 1.25 − 1 = +50%. invested nets the withdrawal and V ignores the
	// realized the series carries.
	assertPerfBuckets(t, got.Buckets, []model.PerformanceBucket{
		{Period: "2025-01", Return: decimal.Zero, TWR: decimal.Zero, Invested: decimal.NewFromInt(1000), Value: decimal.NewFromInt(1000)},
		{Period: "2025-02", Return: decimal.NewFromInt(50), TWR: decimal.NewFromInt(50), Invested: decimal.NewFromInt(400), Value: decimal.NewFromInt(750)},
	})
}

func TestGetPortfolioPerformanceBuckets_MidPeriodBuyIsTimeWeighted(t *testing.T) {
	owner := uuid.New()
	pfUSD := uuid.New()
	stockID := uuid.New().String()
	buy1 := time.Date(2025, 1, 5, 0, 0, 0, 0, time.UTC)
	buy2 := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)
	d1 := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)
	d3 := time.Date(2025, 2, 9, 0, 0, 0, 0, time.UTC)
	pf := &fakePortfolioRepo{
		portfolio: perfPortfolio(owner, pfUSD, "USD"),
		holdings:  []*model.Holding{seriesHolding(pfUSD, stockID)},
	}
	assets := map[uuid.UUID][]model.AssetPositionSeries{
		pfUSD: {
			{AssetID: stockID, Series: []model.PositionPoint{
				{Date: d1, MarketValue: decimal.NewFromInt(1000), CostBasis: decimal.NewFromInt(1000)},
				{Date: d2, MarketValue: decimal.NewFromInt(990), CostBasis: decimal.NewFromInt(1090)},
				{Date: d3, MarketValue: decimal.NewFromInt(880), CostBasis: decimal.NewFromInt(1090)},
			}},
		},
	}
	txs := &fakeTransactionRepo{txs: []model.TransactionWithAsset{
		perfTx(pfUSD, stockID, model.TxBuy, buy1, 100, 10, 0),
		perfTx(pfUSD, stockID, model.TxBuy, buy2, 10, 9, 0),
	}}
	svc := newPortfolioPerfTestService(t, pf, &fakeFXRepo{}, assets, txs)

	got, err := svc.GetPortfolioPerformanceBuckets(context.Background(), pfUSD, owner, "month")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The +90 deposit is isolated out of the return: the buy day measures
	// (990 − 1000 − 90)/1000 = −10% and the rest of the month −11⅑%, the
	// true TWR of the 10 → 9 → 8 path: 0.9 · 8/9 − 1 = −20%.
	assertPerfBuckets(t, got.Buckets, []model.PerformanceBucket{
		{Period: "2025-01", Return: decimal.Zero, TWR: decimal.Zero, Invested: decimal.NewFromInt(1000), Value: decimal.NewFromInt(1000)},
		{Period: "2025-02", Return: decimal.NewFromInt(-20), TWR: decimal.NewFromInt(-20), Invested: decimal.NewFromInt(1090), Value: decimal.NewFromInt(880)},
	})
}

func TestGetPortfolioPerformanceBuckets_UnpricedBondCarriedAtCost(t *testing.T) {
	owner := uuid.New()
	pfUSD := uuid.New()
	stockID := uuid.New().String()
	bondID := uuid.New().String()
	buyStock := time.Date(2025, 1, 5, 0, 0, 0, 0, time.UTC)
	buyBond := time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC)
	sellBond := time.Date(2025, 2, 10, 0, 0, 0, 0, time.UTC)
	d1 := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2025, 2, 5, 0, 0, 0, 0, time.UTC)
	d3 := time.Date(2025, 2, 10, 0, 0, 0, 0, time.UTC)
	d4 := time.Date(2025, 2, 28, 0, 0, 0, 0, time.UTC)
	stock := seriesHolding(pfUSD, stockID)
	bond := seriesHolding(pfUSD, bondID)
	bond.HasPrice = false
	pf := &fakePortfolioRepo{
		portfolio: perfPortfolio(owner, pfUSD, "USD"),
		holdings:  []*model.Holding{stock, bond},
	}
	// The unpriced bond is carried at its 900 cost basis while held and at
	// the 450 remaining cost after the half-sale on February the 10th: the
	// 50 gain (proceeds − sold cost) surfaces exactly on the sale day.
	assets := map[uuid.UUID][]model.AssetPositionSeries{
		pfUSD: {
			{AssetID: stockID, Series: []model.PositionPoint{
				{Date: d1, MarketValue: decimal.NewFromInt(1000), CostBasis: decimal.NewFromInt(1000)},
				{Date: d2, MarketValue: decimal.NewFromInt(1000), CostBasis: decimal.NewFromInt(1000)},
				{Date: d3, MarketValue: decimal.NewFromInt(1000), CostBasis: decimal.NewFromInt(1000)},
				{Date: d4, MarketValue: decimal.NewFromInt(1000), CostBasis: decimal.NewFromInt(1000)},
			}},
			{AssetID: bondID, Series: []model.PositionPoint{
				{Date: d1, CostBasis: decimal.NewFromInt(900)},
				{Date: d2, CostBasis: decimal.NewFromInt(900)},
				{Date: d3, CostBasis: decimal.NewFromInt(450), Realized: decimal.NewFromInt(50)},
				{Date: d4, CostBasis: decimal.NewFromInt(450), Realized: decimal.NewFromInt(50)},
			}},
		},
	}
	txs := &fakeTransactionRepo{txs: []model.TransactionWithAsset{
		perfTx(pfUSD, stockID, model.TxBuy, buyStock, 100, 10, 0),
		perfTx(pfUSD, bondID, model.TxBuy, buyBond, 10, 90, 0),
		perfTx(pfUSD, bondID, model.TxSell, sellBond, 5, 100, 0),
	}}
	svc := newPortfolioPerfTestService(t, pf, &fakeFXRepo{}, assets, txs)

	got, err := svc.GetPortfolioPerformanceBuckets(context.Background(), pfUSD, owner, "month")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertPerfBuckets(t, got.Buckets, []model.PerformanceBucket{
		{Period: "2025-01", Return: decimal.Zero, TWR: decimal.Zero, Invested: decimal.NewFromInt(1900), Value: decimal.NewFromInt(1900)},
		{Period: "2025-02", Return: decimal.RequireFromString("2.6316"), TWR: decimal.RequireFromString("2.6316"), Invested: decimal.NewFromInt(1400), Value: decimal.NewFromInt(1450)},
	})
}

func TestGetPortfolioPerformanceBuckets_YearlyBuckets(t *testing.T) {
	owner := uuid.New()
	pfUSD := uuid.New()
	stockID := uuid.New().String()
	buy := time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC)
	d1 := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2025, 6, 10, 0, 0, 0, 0, time.UTC)
	pf := &fakePortfolioRepo{
		portfolio: perfPortfolio(owner, pfUSD, "USD"),
		holdings:  []*model.Holding{seriesHolding(pfUSD, stockID)},
	}
	assets := map[uuid.UUID][]model.AssetPositionSeries{
		pfUSD: {
			{AssetID: stockID, Series: []model.PositionPoint{
				{Date: d1, MarketValue: decimal.NewFromInt(1200), CostBasis: decimal.NewFromInt(1000), Realized: decimal.NewFromInt(60)},
				{Date: d2, MarketValue: decimal.NewFromInt(1400), CostBasis: decimal.NewFromInt(1000), Realized: decimal.NewFromInt(60)},
			}},
		},
	}
	txs := &fakeTransactionRepo{txs: []model.TransactionWithAsset{
		perfTx(pfUSD, stockID, model.TxBuy, buy, 100, 10, 0),
	}}
	svc := newPortfolioPerfTestService(t, pf, &fakeFXRepo{}, assets, txs)

	got, err := svc.GetPortfolioPerformanceBuckets(context.Background(), pfUSD, owner, "year")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Granularity != "year" {
		t.Fatalf("granularity = %q, want year", got.Granularity)
	}
	assertPerfBuckets(t, got.Buckets, []model.PerformanceBucket{
		{Period: "2024", Return: decimal.Zero, TWR: decimal.Zero, Invested: decimal.NewFromInt(1000), Value: decimal.NewFromInt(1200)},
		{Period: "2025", Return: decimal.RequireFromString("16.6667"), TWR: decimal.RequireFromString("16.6667"), Invested: decimal.NewFromInt(1000), Value: decimal.NewFromInt(1400)},
	})
}

func TestGetPortfolioPerformanceBuckets_ForeignAssetFlowsUsePortfolioCurrency(t *testing.T) {
	owner := uuid.New()
	pfEUR := uuid.New()
	usdAssetID := uuid.New().String()
	buyJan := time.Date(2025, 1, 10, 0, 0, 0, 0, time.UTC)
	d1 := time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2025, 2, 28, 0, 0, 0, 0, time.UTC)
	// A USD asset inside a EUR portfolio: the materialized series is already
	// in EUR (1000 USD at 0.9), while the ledger is in USD and must be
	// converted to EUR at the transaction-date FX (+1000 USD → +900).
	h := seriesHolding(pfEUR, usdAssetID)
	h.Currency = "USD"
	pf := &fakePortfolioRepo{
		portfolio: perfPortfolio(owner, pfEUR, "EUR"),
		holdings:  []*model.Holding{h},
	}
	fx := &fakeFXRepo{
		rates: map[string]decimal.Decimal{"EUR": decimal.RequireFromString("0.9")},
	}
	assets := map[uuid.UUID][]model.AssetPositionSeries{
		pfEUR: {
			{AssetID: usdAssetID, Series: []model.PositionPoint{
				{Date: d1, MarketValue: decimal.NewFromInt(900), CostBasis: decimal.NewFromInt(900)},
				{Date: d2, MarketValue: decimal.NewFromInt(990), CostBasis: decimal.NewFromInt(900)},
			}},
		},
	}
	txs := &fakeTransactionRepo{txs: []model.TransactionWithAsset{
		perfTx(pfEUR, usdAssetID, model.TxBuy, buyJan, 10, 100, 0),
	}}
	svc := newPortfolioPerfTestService(t, pf, fx, assets, txs)

	got, err := svc.GetPortfolioPerformanceBuckets(context.Background(), pfEUR, owner, "month")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Currency != "EUR" {
		t.Fatalf("currency = %q, want EUR (the portfolio's, not the asset's)", got.Currency)
	}
	// January is the first bucket (return 0, invested 1000*0.9 = 900 = V).
	// February is flat at EUR 990 after the +90 rise already measured in
	// the series... except the rise happens on the last point: the only
	// priced days are Jan 20 (900) and Feb 28 (990), so February's single
	// day measures (990 − 900)/900 = +10%.
	assertPerfBuckets(t, got.Buckets, []model.PerformanceBucket{
		{Period: "2025-01", Return: decimal.Zero, TWR: decimal.Zero, Invested: decimal.NewFromInt(900), Value: decimal.NewFromInt(900)},
		{Period: "2025-02", Return: decimal.NewFromInt(10), TWR: decimal.NewFromInt(10), Invested: decimal.NewFromInt(900), Value: decimal.NewFromInt(990)},
	})
}

func TestGetPortfolioPerformanceBuckets_EmptyPortfolioHasNoBuckets(t *testing.T) {
	owner := uuid.New()
	pfEUR := uuid.New()
	pf := &fakePortfolioRepo{portfolio: perfPortfolio(owner, pfEUR, "EUR")}
	svc := newPortfolioPerfTestService(t, pf, &fakeFXRepo{}, nil, &fakeTransactionRepo{})

	got, err := svc.GetPortfolioPerformanceBuckets(context.Background(), pfEUR, owner, "month")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Currency != "EUR" || got.Granularity != "month" {
		t.Fatalf("currency/granularity = (%q, %q), want (EUR, month)", got.Currency, got.Granularity)
	}
	if got.Buckets == nil || len(got.Buckets) != 0 {
		t.Fatalf("buckets = %+v, want an empty non-nil slice", got.Buckets)
	}
}

func TestGetPortfolioPerformanceBuckets_InvalidGranularity(t *testing.T) {
	owner := uuid.New()
	pfEUR := uuid.New()
	pf := &fakePortfolioRepo{portfolio: perfPortfolio(owner, pfEUR, "EUR")}
	svc := newPortfolioPerfTestService(t, pf, &fakeFXRepo{}, nil, &fakeTransactionRepo{})

	for _, g := range []string{"", "week", "MONTH"} {
		if _, err := svc.GetPortfolioPerformanceBuckets(context.Background(), pfEUR, owner, g); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("granularity %q: err = %v, want ErrInvalidInput", g, err)
		}
	}
}

func TestGetPortfolioPerformanceBuckets_ForeignOwnerRejected(t *testing.T) {
	pfEUR := uuid.New()
	pf := &fakePortfolioRepo{portfolio: perfPortfolio(uuid.New(), pfEUR, "EUR")}
	svc := newPortfolioPerfTestService(t, pf, &fakeFXRepo{}, nil, &fakeTransactionRepo{})

	if _, err := svc.GetPortfolioPerformanceBuckets(context.Background(), pfEUR, uuid.New(), "month"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("err = %v, want ErrForbidden", err)
	}
}

// perfTx builds a transaction ledger row for the dashboard performance
// tests: quantity, price and fees are integer amounts in the asset currency.
func perfTx(pf uuid.UUID, assetID string, typ model.TransactionType, date time.Time, qty, price, fees int64) model.TransactionWithAsset {
	return model.TransactionWithAsset{
		PortfolioID: pf,
		AssetID:     mustUUID(assetID),
		Type:        typ,
		Quantity:    decimal.NewFromInt(qty),
		Price:       decimal.NewFromInt(price),
		Fees:        decimal.NewFromInt(fees),
		Date:        date,
	}
}

func assertPerfBuckets(t *testing.T, got, want []model.PerformanceBucket) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("buckets = %+v, want %d buckets", got, len(want))
	}
	for i, w := range want {
		if got[i].Period != w.Period {
			t.Fatalf("buckets[%d].period = %q, want %q", i, got[i].Period, w.Period)
		}
		if !equalDecimal(got[i].Return, w.Return) {
			t.Fatalf("buckets[%d].return = %v, want %v", i, got[i].Return, w.Return)
		}
		if !equalDecimal(got[i].TWR, w.TWR) {
			t.Fatalf("buckets[%d].twr = %v, want %v", i, got[i].TWR, w.TWR)
		}
		if !equalDecimal(got[i].Invested, w.Invested) {
			t.Fatalf("buckets[%d].invested = %v, want %v", i, got[i].Invested, w.Invested)
		}
		if !equalDecimal(got[i].Value, w.Value) {
			t.Fatalf("buckets[%d].value = %v, want %v", i, got[i].Value, w.Value)
		}
	}
}
func TestGetDashboardAllocation_UsesUserBaseCurrency(t *testing.T) {
	stockID := uuid.New()
	pf := &fakePortfolioRepo{
		portfolios: []*model.Portfolio{{ID: uuid.New(), Currency: "USD"}},
		holdings: []*model.Holding{
			holding(stockID.String(), "USD", "IT", "Financials", model.AssetTypeStock, decimal.NewFromInt(1), decimal.NewFromInt(100)),
		},
	}
	fx := &fakeFXRepo{rates: map[string]decimal.Decimal{"EUR": decimal.RequireFromString("0.9")}}
	svc := newDashboardTestService(t, pf, fx, "EUR", nil)

	got, err := svc.GetDashboardAllocation(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Currency != "EUR" {
		t.Fatalf("currency = %q, want EUR (the user's base, not USD)", got.Currency)
	}
	eu := regionByName(t, got.Regions, "Europe Developed")
	if !equalDecimal(eu.Value, decimal.NewFromInt(90)) {
		t.Fatalf("Europe Developed value = %v, want 90 (100 USD at USD->EUR 0.9)", eu.Value)
	}
	if !equalDecimal(got.Covered, decimal.NewFromInt(90)) {
		t.Fatalf("covered = %v, want 90", got.Covered)
	}
}

// newTransactionPageTestService wires the minimal repos the paginated
// transactions read touches: portfolio ownership lookup and the ledger.
func newTransactionPageTestService(t *testing.T, p *fakePortfolioRepo, txs *fakeTransactionRepo) *Service {
	t.Helper()
	repos := &repository.Repository{
		Asset:       &fakeAssetRepo{},
		Portfolio:   p,
		Transaction: txs,
		Exposure:    &fakeExposureRepo{},
		FX:          &fakeFXRepo{},
		Lookup:      &fakeLookupRepo{},
	}
	return New(repos, nil, nil, nil, time.Minute, time.Hour, cache.New(nil), 0, 0, nil)
}

// pagedTxLedger builds n transactions in the order the SQL would return them
// (newest first), so the fake can simply slice them.
func pagedTxLedger(portfolioID uuid.UUID, n int) []model.TransactionWithAsset {
	txs := make([]model.TransactionWithAsset, 0, n)
	base := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	for i := 0; i < n; i++ {
		txs = append(txs, model.TransactionWithAsset{
			ID:          uuid.New(),
			PortfolioID: portfolioID,
			AssetID:     uuid.New(),
			AssetTicker: "ACME",
			Type:        model.TxBuy,
			Quantity:    decimal.NewFromInt(1),
			Price:       decimal.NewFromInt(int64(100 + i)),
			Date:        base.AddDate(0, 0, -i),
			CreatedAt:   base.AddDate(0, 0, -i),
		})
	}
	return txs
}

func TestListTransactionsPaged_FirstPageAndTotal(t *testing.T) {
	owner := uuid.New()
	pid := uuid.New()
	pf := &fakePortfolioRepo{portfolio: perfPortfolio(owner, pid, "EUR")}
	ledger := pagedTxLedger(pid, 5)
	svc := newTransactionPageTestService(t, pf, &fakeTransactionRepo{txs: ledger})

	page, err := svc.ListTransactionsPaged(context.Background(), pid, owner, 2, 0, model.TransactionFilter{})
	if err != nil {
		t.Fatalf("ListTransactionsPaged: %v", err)
	}
	if page.Total != 5 {
		t.Fatalf("total = %d, want 5", page.Total)
	}
	if page.Limit != 2 || page.Offset != 0 {
		t.Fatalf("limit/offset = %d/%d, want 2/0", page.Limit, page.Offset)
	}
	if len(page.Transactions) != 2 {
		t.Fatalf("page size = %d, want 2", len(page.Transactions))
	}
	for i, tx := range page.Transactions {
		if tx.ID != ledger[i].ID {
			t.Fatalf("tx %d = %s, want %s", i, tx.ID, ledger[i].ID)
		}
	}
}

func TestListTransactionsPaged_OffsetBeyondEndYieldsEmptyPageWithTotal(t *testing.T) {
	owner := uuid.New()
	pid := uuid.New()
	pf := &fakePortfolioRepo{portfolio: perfPortfolio(owner, pid, "EUR")}
	svc := newTransactionPageTestService(t, pf, &fakeTransactionRepo{txs: pagedTxLedger(pid, 3)})

	page, err := svc.ListTransactionsPaged(context.Background(), pid, owner, 10, 50, model.TransactionFilter{})
	if err != nil {
		t.Fatalf("ListTransactionsPaged: %v", err)
	}
	if page.Transactions == nil {
		t.Fatal("transactions must be an empty slice, never null")
	}
	if len(page.Transactions) != 0 {
		t.Fatalf("page size = %d, want 0", len(page.Transactions))
	}
	if page.Total != 3 {
		t.Fatalf("total = %d, want 3", page.Total)
	}
}

func TestListTransactionsPaged_LimitDefaultsAndClamps(t *testing.T) {
	owner := uuid.New()
	pid := uuid.New()
	pf := &fakePortfolioRepo{portfolio: perfPortfolio(owner, pid, "EUR")}
	svc := newTransactionPageTestService(t, pf, &fakeTransactionRepo{txs: pagedTxLedger(pid, 3)})

	page, err := svc.ListTransactionsPaged(context.Background(), pid, owner, 0, 0, model.TransactionFilter{})
	if err != nil {
		t.Fatalf("ListTransactionsPaged (default): %v", err)
	}
	if page.Limit != 20 {
		t.Fatalf("default limit = %d, want 20", page.Limit)
	}

	page, err = svc.ListTransactionsPaged(context.Background(), pid, owner, 500, 0, model.TransactionFilter{})
	if err != nil {
		t.Fatalf("ListTransactionsPaged (clamp): %v", err)
	}
	if page.Limit != 100 {
		t.Fatalf("clamped limit = %d, want 100", page.Limit)
	}
	if len(page.Transactions) != 3 {
		t.Fatalf("page size = %d, want 3", len(page.Transactions))
	}
}

func TestListTransactionsPaged_RejectsNegativePagination(t *testing.T) {
	owner := uuid.New()
	pid := uuid.New()
	pf := &fakePortfolioRepo{portfolio: perfPortfolio(owner, pid, "EUR")}
	svc := newTransactionPageTestService(t, pf, &fakeTransactionRepo{txs: pagedTxLedger(pid, 1)})

	if _, err := svc.ListTransactionsPaged(context.Background(), pid, owner, -1, 0, model.TransactionFilter{}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("negative limit err = %v, want ErrInvalidInput", err)
	}
	if _, err := svc.ListTransactionsPaged(context.Background(), pid, owner, 10, -5, model.TransactionFilter{}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("negative offset err = %v, want ErrInvalidInput", err)
	}
}

func TestListTransactionsPaged_NonOwnerForbidden(t *testing.T) {
	owner := uuid.New()
	stranger := uuid.New()
	pid := uuid.New()
	pf := &fakePortfolioRepo{portfolio: perfPortfolio(owner, pid, "EUR")}
	svc := newTransactionPageTestService(t, pf, &fakeTransactionRepo{txs: pagedTxLedger(pid, 3)})

	if _, err := svc.ListTransactionsPaged(context.Background(), pid, stranger, 10, 0, model.TransactionFilter{}); !errors.Is(err, ErrForbidden) {
		t.Fatalf("non-owner err = %v, want ErrForbidden", err)
	}
}

// filterTxLedger builds a seven-row mixed ledger in the order the filtered
// SQL would return it (newest first): types and assets alternate, dates step
// back one day per row, and row 3 carries a 15:30 UTC time-of-day so the
// date-only inclusive bounds are exercised against a real timestamp.
func filterTxLedger(portfolioID, assetA, assetB uuid.UUID) []model.TransactionWithAsset {
	base := time.Date(2024, 12, 7, 0, 0, 0, 0, time.UTC)
	rows := []struct {
		typ   model.TransactionType
		asset uuid.UUID
		days  int
		hours int
	}{
		{model.TxBuy, assetA, 0, 0},
		{model.TxSell, assetB, 1, 0},
		{model.TxDividend, assetA, 2, 0},
		{model.TxBuy, assetB, 3, 15},
		{model.TxFee, assetA, 4, 0},
		{model.TxBuy, assetA, 5, 0},
		{model.TxSplit, assetB, 6, 0},
	}
	txs := make([]model.TransactionWithAsset, 0, len(rows))
	for i, row := range rows {
		date := base.AddDate(0, 0, -row.days).Add(time.Duration(row.hours) * time.Hour)
		txs = append(txs, model.TransactionWithAsset{
			ID:          uuid.New(),
			PortfolioID: portfolioID,
			AssetID:     row.asset,
			AssetTicker: "ACME",
			Type:        row.typ,
			Quantity:    decimal.NewFromInt(1),
			Price:       decimal.NewFromInt(int64(100 + i)),
			Date:        date,
			CreatedAt:   date,
		})
	}
	return txs
}

func newFilterTestService(t *testing.T, owner, pid uuid.UUID, ledger []model.TransactionWithAsset) (*Service, *fakeTransactionRepo) {
	t.Helper()
	pf := &fakePortfolioRepo{portfolio: perfPortfolio(owner, pid, "EUR")}
	rx := &fakeTransactionRepo{txs: ledger}
	return newTransactionPageTestService(t, pf, rx), rx
}

func mustFilterDate(t *testing.T, raw string) *time.Time {
	t.Helper()
	d, err := model.ParseTransactionDate(raw)
	if err != nil {
		t.Fatalf("mustFilterDate(%q): %v", raw, err)
	}
	return &d
}

func txIDs(txs []model.TransactionWithAsset) []uuid.UUID {
	out := make([]uuid.UUID, 0, len(txs))
	for _, tx := range txs {
		out = append(out, tx.ID)
	}
	return out
}

func TestListTransactionsPaged_FilterByType(t *testing.T) {
	owner, pid, assetA, assetB := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	ledger := filterTxLedger(pid, assetA, assetB)
	svc, rx := newFilterTestService(t, owner, pid, ledger)

	page, err := svc.ListTransactionsPaged(context.Background(), pid, owner, 10, 0, model.TransactionFilter{Type: "sell"})
	if err != nil {
		t.Fatalf("ListTransactionsPaged: %v", err)
	}
	if page.Total != 1 {
		t.Fatalf("total = %d, want 1 (filtered)", page.Total)
	}
	if len(page.Transactions) != 1 || page.Transactions[0].ID != ledger[1].ID {
		t.Fatalf("page = %v, want only the sell row", txIDs(page.Transactions))
	}
	if rx.lastFilter.Type != "sell" {
		t.Fatalf("repo received filter %+v, want Type=sell", rx.lastFilter)
	}
}

func TestListTransactionsPaged_FilterByAssetID(t *testing.T) {
	owner, pid, assetA, assetB := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	ledger := filterTxLedger(pid, assetA, assetB)
	svc, _ := newFilterTestService(t, owner, pid, ledger)

	page, err := svc.ListTransactionsPaged(context.Background(), pid, owner, 10, 0, model.TransactionFilter{AssetID: &assetA})
	if err != nil {
		t.Fatalf("ListTransactionsPaged: %v", err)
	}
	if page.Total != 4 {
		t.Fatalf("total = %d, want 4 assetA rows", page.Total)
	}
	want := txIDs([]model.TransactionWithAsset{ledger[0], ledger[2], ledger[4], ledger[5]})
	got := txIDs(page.Transactions)
	if len(got) != len(want) {
		t.Fatalf("page size = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("row %d = %s, want %s", i, got[i], want[i])
		}
	}
}

func TestListTransactionsPaged_FilterDatesAreInclusiveCalendarDays(t *testing.T) {
	owner, pid, assetA, assetB := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	ledger := filterTxLedger(pid, assetA, assetB)
	svc, _ := newFilterTestService(t, owner, pid, ledger)
	row := func(i int) uuid.UUID { return ledger[i].ID }

	// from = the 12-04 boundary must include row 3 (12-04 at 15:30 UTC).
	page, err := svc.ListTransactionsPaged(context.Background(), pid, owner, 10, 0, model.TransactionFilter{From: mustFilterDate(t, "2024-12-04")})
	if err != nil {
		t.Fatalf("from: %v", err)
	}
	if page.Total != 4 || page.Transactions[3].ID != row(3) {
		t.Fatalf("from bound: total=%d page=%v, want 4 rows ending at the boundary row", page.Total, txIDs(page.Transactions))
	}

	// to = the same day must include that same intra-day row and its older siblings.
	page, err = svc.ListTransactionsPaged(context.Background(), pid, owner, 10, 0, model.TransactionFilter{To: mustFilterDate(t, "2024-12-04")})
	if err != nil {
		t.Fatalf("to: %v", err)
	}
	if page.Total != 4 {
		t.Fatalf("to bound: total = %d, want 4", page.Total)
	}

	page, err = svc.ListTransactionsPaged(context.Background(), pid, owner, 10, 0, model.TransactionFilter{
		From: mustFilterDate(t, "2024-12-05"),
		To:   mustFilterDate(t, "2024-12-06"),
	})
	if err != nil {
		t.Fatalf("window: %v", err)
	}
	want := txIDs([]model.TransactionWithAsset{ledger[1], ledger[2]})
	got := txIDs(page.Transactions)
	if page.Total != 2 || len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("window: total=%d page=%v, want the 12-06 and 12-05 rows", page.Total, got)
	}
}

func TestListTransactionsPaged_CombinedFiltersPaginateTheFilteredSet(t *testing.T) {
	owner, pid, assetA, assetB := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	ledger := filterTxLedger(pid, assetA, assetB)
	svc, _ := newFilterTestService(t, owner, pid, ledger)

	// buy + assetA leaves exactly the newest (12-07) and 12-02 rows; the
	// 12-03..12-06 window keeps only the newest of them.
	filter := model.TransactionFilter{Type: "buy", AssetID: &assetA}
	page, err := svc.ListTransactionsPaged(context.Background(), pid, owner, 1, 1, filter)
	if err != nil {
		t.Fatalf("combined: %v", err)
	}
	if page.Total != 2 {
		t.Fatalf("total = %d, want 2 (filtered count, not ledger size)", page.Total)
	}
	if len(page.Transactions) != 1 || page.Transactions[0].ID != ledger[5].ID {
		t.Fatalf("second page = %v, want only the 12-02 buy row", txIDs(page.Transactions))
	}

	narrowed := model.TransactionFilter{
		Type:    "buy",
		AssetID: &assetA,
		From:    mustFilterDate(t, "2024-12-03"),
		To:      mustFilterDate(t, "2024-12-07"),
	}
	page, err = svc.ListTransactionsPaged(context.Background(), pid, owner, 10, 0, narrowed)
	if err != nil {
		t.Fatalf("narrowed: %v", err)
	}
	if page.Total != 1 || page.Transactions[0].ID != ledger[0].ID {
		t.Fatalf("narrowed: total=%d page=%v, want only the 12-07 buy row", page.Total, txIDs(page.Transactions))
	}

	// An offset past the end of the filtered set yields an empty page with
	// the filtered total.
	page, err = svc.ListTransactionsPaged(context.Background(), pid, owner, 10, 5, filter)
	if err != nil {
		t.Fatalf("past end: %v", err)
	}
	if page.Transactions == nil || len(page.Transactions) != 0 || page.Total != 2 {
		t.Fatalf("past end: total=%d len=%d, want empty page with filtered total 2", page.Total, len(page.Transactions))
	}
}

func TestListTransactionsPaged_RejectsUnknownFilterType(t *testing.T) {
	owner, pid, assetA, assetB := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	svc, rx := newFilterTestService(t, owner, pid, filterTxLedger(pid, assetA, assetB))

	if _, err := svc.ListTransactionsPaged(context.Background(), pid, owner, 10, 0, model.TransactionFilter{Type: "withdraw"}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("bad type err = %v, want ErrInvalidInput", err)
	}
	if rx.lastFilter != (model.TransactionFilter{}) {
		t.Fatalf("repo must not be queried after a rejected filter, saw %+v", rx.lastFilter)
	}
}

func TestParseTransactionFilter(t *testing.T) {
	asset := uuid.New()
	t.Run("all filters", func(t *testing.T) {
		f, err := model.ParseTransactionFilter(url.Values{
			"type":     {"dividend"},
			"asset_id": {asset.String()},
			"from":     {"2024-12-01"},
			"to":       {"2024-12-31"},
		})
		if err != nil {
			t.Fatalf("ParseTransactionFilter: %v", err)
		}
		if f.Type != "dividend" || f.AssetID == nil || *f.AssetID != asset {
			t.Fatalf("filter = %+v, want dividend + asset %s", f, asset)
		}
		if f.From == nil || !f.From.Equal(time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC)) {
			t.Fatalf("from = %v, want 2024-12-01 UTC midnight", f.From)
		}
		if f.To == nil || !f.To.Equal(time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)) {
			t.Fatalf("to = %v, want 2024-12-31 UTC midnight", f.To)
		}
	})
	t.Run("empty query yields zero filter", func(t *testing.T) {
		f, err := model.ParseTransactionFilter(url.Values{})
		if err != nil {
			t.Fatalf("ParseTransactionFilter: %v", err)
		}
		if f != (model.TransactionFilter{}) {
			t.Fatalf("filter = %+v, want zero value", f)
		}
	})
	cases := []struct {
		name  string
		query url.Values
		want  string
	}{
		{"bad type", url.Values{"type": {"withdraw"}}, "invalid type"},
		{"bad asset id", url.Values{"asset_id": {"not-a-uuid"}}, "invalid asset_id"},
		{"impossible date", url.Values{"from": {"2024-13-01"}}, "invalid from: date must be YYYY-MM-DD"},
		{"unpadded date", url.Values{"from": {"2024-1-5"}}, "invalid from: date must be YYYY-MM-DD"},
		{"datetime not accepted", url.Values{"to": {"2024-12-01T00:00"}}, "invalid to: date must be YYYY-MM-DD"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := model.ParseTransactionFilter(tc.query); err == nil {
				t.Fatalf("err = nil, want %q", tc.want)
			} else if err.Error() != tc.want {
				t.Fatalf("err = %q, want %q", err, tc.want)
			}
		})
	}
}
