package service

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/alv67/peculium/internal/model"
	"github.com/alv67/peculium/internal/repository"
)

// backupStore is a small in-memory stand-in for the Postgres schema shared
// by the repository stubs below. It mirrors the semantics the backup paths
// depend on: assets key by unique (exact-case) ticker, prices upsert on
// (asset_id, date), deleting a portfolio cascades to its transactions,
// FindByUser includes portfolios shared through portfolio_shares, and the
// exposure helpers keep the repository's filtering and name-ordering rules.

type backupStore struct {
	users      map[uuid.UUID]*model.User
	portfolios map[uuid.UUID]*model.Portfolio
	shares     map[uuid.UUID][]uuid.UUID
	txs        []model.TransactionWithAsset
	assets     map[uuid.UUID]*model.Asset
	regions    map[string][]model.ExposureRow
	sectors    map[string][]model.ExposureRow
	countries  map[string][]model.ExposureRow
	provenance map[string]map[string]model.ExposureProvenance
	prices     []*model.Price
	currencies map[string]*model.Currency
}

func newBackupStore() *backupStore {
	return &backupStore{
		users:      map[uuid.UUID]*model.User{},
		portfolios: map[uuid.UUID]*model.Portfolio{},
		shares:     map[uuid.UUID][]uuid.UUID{},
		assets:     map[uuid.UUID]*model.Asset{},
		regions:    map[string][]model.ExposureRow{},
		sectors:    map[string][]model.ExposureRow{},
		countries:  map[string][]model.ExposureRow{},
		provenance: map[string]map[string]model.ExposureProvenance{},
		currencies: map[string]*model.Currency{},
	}
}

func (s *backupStore) repos() *repository.Repository {
	return &repository.Repository{
		User:        (*backupUserRepo)(s),
		Portfolio:   (*backupPortfolioRepo)(s),
		Transaction: (*backupTransactionRepo)(s),
		Asset:       (*backupAssetRepo)(s),
		Exposure:    (*backupExposureRepo)(s),
		Price:       (*backupPriceRepo)(s),
		Currency:    (*backupCurrencyRepo)(s),
	}
}

func (s *backupStore) assetByTicker(ticker string) *model.Asset {
	for _, a := range s.assets {
		if a.Ticker == ticker {
			return a
		}
	}
	return nil
}

func (s *backupStore) portfolioForUser(userID uuid.UUID) []*model.Portfolio {
	var out []*model.Portfolio
	for _, p := range s.portfolios {
		if p.UserID == userID {
			out = append(out, p)
			continue
		}
		for _, uid := range s.shares[p.ID] {
			if uid == userID {
				out = append(out, p)
				break
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}

func (s *backupStore) txsOf(portfolioID uuid.UUID) []model.TransactionWithAsset {
	var out []model.TransactionWithAsset
	for _, tx := range s.txs {
		if tx.PortfolioID == portfolioID {
			out = append(out, tx)
		}
	}
	return out
}

func backupDay(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func (s *backupStore) seedUser(email, name, baseCurrency string) uuid.UUID {
	id := uuid.New()
	s.users[id] = &model.User{ID: id, Email: email, Name: name, BaseCurrency: baseCurrency}
	return id
}

func (s *backupStore) seedPortfolio(userID uuid.UUID, name, currency string, createdAt time.Time) uuid.UUID {
	id := uuid.New()
	s.portfolios[id] = &model.Portfolio{ID: id, UserID: userID, Name: name, Currency: currency, CreatedAt: createdAt, UpdatedAt: createdAt}
	return id
}

func (s *backupStore) seedAsset(ticker, name string, typ model.AssetType, currency, isin, priceSource, assetClass string) uuid.UUID {
	id := uuid.New()
	s.assets[id] = &model.Asset{ID: id, Ticker: ticker, Name: name, Type: typ, Currency: currency, ISIN: isin, PriceSource: priceSource, AssetClass: assetClass}
	return id
}

func (s *backupStore) seedTx(portfolioID, assetID uuid.UUID, typ model.TransactionType, qty, price, fees string, date time.Time, notes string) {
	a := s.assets[assetID]
	s.txs = append(s.txs, model.TransactionWithAsset{
		ID: uuid.New(), PortfolioID: portfolioID, AssetID: assetID,
		AssetTicker: a.Ticker, AssetName: a.Name,
		Type: typ, Quantity: dec(qty), Price: dec(price), Fees: dec(fees), Date: date, Notes: notes,
		CreatedAt: date,
	})
}

func (s *backupStore) seedPrice(assetID uuid.UUID, date time.Time, close string, source string) {
	s.prices = append(s.prices, &model.Price{
		ID: uuid.New(), AssetID: assetID, Date: date,
		Open: dec(close), High: dec(close), Low: dec(close), Close: dec(close),
		Source: source,
	})
}

func (s *backupStore) seedExposure(assetID uuid.UUID, dimension string, rows map[string]string, provSource string) {
	key := assetID.String()
	exposed := make([]model.ExposureRow, 0, len(rows))
	for name, w := range rows {
		exposed = append(exposed, model.ExposureRow{Name: name, Weight: dec(w)})
	}
	switch dimension {
	case model.ExposureDimensionCountries:
		s.countries[key] = exposed
	case model.ExposureDimensionRegions:
		s.regions[key] = exposed
	case model.ExposureDimensionSectors:
		s.sectors[key] = exposed
	}
	if provSource != "" {
		if s.provenance[key] == nil {
			s.provenance[key] = map[string]model.ExposureProvenance{}
		}
		s.provenance[key][dimension] = model.ExposureProvenance{Source: provSource, UpdatedAt: backupDay(2026, 1, 1)}
	}
}

func (s *backupStore) seedCurrency(code, name, symbol string) {
	s.currencies[code] = &model.Currency{Code: code, Name: name, Symbol: symbol, Enabled: true}
}

func dec(s string) decimal.Decimal {
	d, err := decimal.NewFromString(s)
	if err != nil {
		panic(err)
	}
	return d
}

// backupUserRepo implements repository.UserRepository over backupStore.

type backupUserRepo backupStore

func (r *backupUserRepo) Create(ctx context.Context, email, name, password string) (*model.User, error) {
	return nil, nil
}
func (r *backupUserRepo) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	for _, u := range r.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, nil
}
func (r *backupUserRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	return r.users[id], nil
}
func (r *backupUserRepo) Update(ctx context.Context, user *model.User) error {
	r.users[user.ID] = user
	return nil
}
func (r *backupUserRepo) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	return nil
}

// backupPortfolioRepo implements repository.PortfolioRepository over backupStore.

type backupPortfolioRepo backupStore

func (r *backupPortfolioRepo) Create(ctx context.Context, p *model.Portfolio) (*model.Portfolio, error) {
	stored := *p
	stored.ID = uuid.New()
	stored.CreatedAt = time.Now().UTC()
	stored.UpdatedAt = stored.CreatedAt
	r.portfolios[stored.ID] = &stored
	return &stored, nil
}
func (r *backupPortfolioRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.Portfolio, error) {
	return r.portfolios[id], nil
}
func (r *backupPortfolioRepo) FindByUser(ctx context.Context, userID uuid.UUID) ([]*model.Portfolio, error) {
	return (*backupStore)(r).portfolioForUser(userID), nil
}
func (r *backupPortfolioRepo) Update(ctx context.Context, p *model.Portfolio) error { return nil }
func (r *backupPortfolioRepo) Delete(ctx context.Context, id uuid.UUID) error {
	delete(r.portfolios, id)
	delete(r.shares, id)
	// mirror ON DELETE CASCADE on transactions
	kept := r.txs[:0]
	for _, tx := range r.txs {
		if tx.PortfolioID != id {
			kept = append(kept, tx)
		}
	}
	r.txs = kept
	return nil
}
func (r *backupPortfolioRepo) HeldAssets(ctx context.Context, portfolioID uuid.UUID) ([]*model.Asset, error) {
	return nil, nil
}
func (r *backupPortfolioRepo) HoldingsDetailed(ctx context.Context, portfolioIDs []uuid.UUID) ([]*model.Holding, error) {
	return nil, nil
}
func (r *backupPortfolioRepo) FindAll(ctx context.Context) ([]uuid.UUID, error) { return nil, nil }

// backupTransactionRepo implements repository.TransactionRepository over backupStore.

type backupTransactionRepo backupStore

func (r *backupTransactionRepo) Create(ctx context.Context, tx *model.Transaction) (*model.Transaction, error) {
	stored := *tx
	stored.ID = uuid.New()
	stored.CreatedAt = time.Now().UTC()
	a := r.assets[tx.AssetID]
	ticker, name := "", ""
	if a != nil {
		ticker, name = a.Ticker, a.Name
	}
	r.txs = append(r.txs, model.TransactionWithAsset{
		ID: stored.ID, PortfolioID: stored.PortfolioID, AssetID: stored.AssetID,
		AssetTicker: ticker, AssetName: name,
		Type: stored.Type, Quantity: stored.Quantity, Price: stored.Price, Fees: stored.Fees,
		Date: stored.Date, Notes: stored.Notes, CreatedAt: stored.CreatedAt,
	})
	return &stored, nil
}
func (r *backupTransactionRepo) FindByPortfolio(ctx context.Context, portfolioID uuid.UUID) ([]model.TransactionWithAsset, error) {
	return (*backupStore)(r).txsOf(portfolioID), nil
}
func (r *backupTransactionRepo) FindByPortfolioPage(ctx context.Context, portfolioID uuid.UUID, filter model.TransactionFilter, limit, offset int) ([]model.TransactionWithAsset, error) {
	return nil, nil
}
func (r *backupTransactionRepo) CountByPortfolio(ctx context.Context, portfolioID uuid.UUID, filter model.TransactionFilter) (int64, error) {
	return 0, nil
}
func (r *backupTransactionRepo) FindByPortfoliosAsc(ctx context.Context, portfolioIDs []uuid.UUID) ([]model.TransactionWithAsset, error) {
	var out []model.TransactionWithAsset
	for _, tx := range r.txs {
		for _, pid := range portfolioIDs {
			if tx.PortfolioID == pid {
				out = append(out, tx)
				break
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Date.Before(out[j].Date) })
	return out, nil
}
func (r *backupTransactionRepo) MinDateByAsset(ctx context.Context, assetIDs []uuid.UUID) (map[uuid.UUID]time.Time, error) {
	return nil, nil
}
func (r *backupTransactionRepo) MinDateByCurrency(ctx context.Context) (map[string]time.Time, error) {
	return nil, nil
}
func (r *backupTransactionRepo) CountByAsset(ctx context.Context, assetID uuid.UUID) (int64, error) {
	return 0, nil
}
func (r *backupTransactionRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.Transaction, error) {
	return nil, nil
}
func (r *backupTransactionRepo) Update(ctx context.Context, tx *model.Transaction) error { return nil }
func (r *backupTransactionRepo) Delete(ctx context.Context, id uuid.UUID) error          { return nil }

// backupAssetRepo implements repository.AssetRepository over backupStore.

type backupAssetRepo backupStore

func (r *backupAssetRepo) Create(ctx context.Context, asset *model.Asset) (*model.Asset, error) {
	stored := *asset
	stored.ID = uuid.New()
	stored.CreatedAt = time.Now().UTC()
	r.assets[stored.ID] = &stored
	return &stored, nil
}
func (r *backupAssetRepo) Update(ctx context.Context, asset *model.Asset) (*model.Asset, error) {
	r.assets[asset.ID] = asset
	return asset, nil
}
func (r *backupAssetRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.Asset, error) {
	return r.assets[id], nil
}
func (r *backupAssetRepo) FindByTicker(ctx context.Context, ticker string) (*model.Asset, error) {
	return (*backupStore)(r).assetByTicker(ticker), nil
}
func (r *backupAssetRepo) FindByIDs(ctx context.Context, ids []uuid.UUID) ([]*model.Asset, error) {
	out := make([]*model.Asset, 0)
	for _, id := range ids {
		if a := r.assets[id]; a != nil {
			out = append(out, a)
		}
	}
	return out, nil
}
func (r *backupAssetRepo) Search(ctx context.Context, query string) ([]*model.Asset, error) {
	return nil, nil
}
func (r *backupAssetRepo) List(ctx context.Context) ([]*model.Asset, error) { return nil, nil }
func (r *backupAssetRepo) ListYahoo(ctx context.Context) ([]*model.Asset, error) {
	return nil, nil
}
func (r *backupAssetRepo) AllStocks(ctx context.Context) ([]*model.Asset, error) { return nil, nil }
func (r *backupAssetRepo) MarkPricesFetched(ctx context.Context, ids []uuid.UUID, at time.Time) error {
	return nil
}
func (r *backupAssetRepo) MarkHistoryBackfilled(ctx context.Context, id uuid.UUID) error { return nil }
func (r *backupAssetRepo) Delete(ctx context.Context, id uuid.UUID) error {
	delete(r.assets, id)
	return nil
}
func (r *backupAssetRepo) Currencies(ctx context.Context) ([]string, error) { return nil, nil }

// backupExposureRepo implements repository.ExposureRepository over backupStore.

type backupExposureRepo backupStore

func exposureRowsByKey(store map[string][]model.ExposureRow, assetIDs []uuid.UUID) map[string][]model.ExposureRow {
	out := map[string][]model.ExposureRow{}
	for _, id := range assetIDs {
		if rows := store[id.String()]; len(rows) > 0 {
			out[id.String()] = sortExposureRows(rows)
		}
	}
	return out
}

func sortExposureRows(rows []model.ExposureRow) []model.ExposureRow {
	out := make([]model.ExposureRow, 0, len(rows))
	for _, row := range rows {
		if row.Name == "" || !row.Weight.IsPositive() {
			continue
		}
		out = append(out, row)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (r *backupExposureRepo) FindRegions(ctx context.Context, assetID uuid.UUID) ([]model.ExposureRow, error) {
	return sortExposureRows(r.regions[assetID.String()]), nil
}
func (r *backupExposureRepo) FindSectors(ctx context.Context, assetID uuid.UUID) ([]model.ExposureRow, error) {
	return sortExposureRows(r.sectors[assetID.String()]), nil
}
func (r *backupExposureRepo) FindCountries(ctx context.Context, assetID uuid.UUID) ([]model.ExposureRow, error) {
	return sortExposureRows(r.countries[assetID.String()]), nil
}
func (r *backupExposureRepo) ReplaceRegions(ctx context.Context, assetID uuid.UUID, rows []model.ExposureRow) error {
	r.regions[assetID.String()] = sortExposureRows(rows)
	return nil
}
func (r *backupExposureRepo) ReplaceSectors(ctx context.Context, assetID uuid.UUID, rows []model.ExposureRow) error {
	r.sectors[assetID.String()] = sortExposureRows(rows)
	return nil
}
func (r *backupExposureRepo) ReplaceCountries(ctx context.Context, assetID uuid.UUID, rows []model.ExposureRow) error {
	r.countries[assetID.String()] = sortExposureRows(rows)
	return nil
}
func (r *backupExposureRepo) FindRegionsByAssets(ctx context.Context, assetIDs []uuid.UUID) (map[string][]model.ExposureRow, error) {
	return exposureRowsByKey(r.regions, assetIDs), nil
}
func (r *backupExposureRepo) FindSectorsByAssets(ctx context.Context, assetIDs []uuid.UUID) (map[string][]model.ExposureRow, error) {
	return exposureRowsByKey(r.sectors, assetIDs), nil
}
func (r *backupExposureRepo) FindCountriesByAssets(ctx context.Context, assetIDs []uuid.UUID) (map[string][]model.ExposureRow, error) {
	return exposureRowsByKey(r.countries, assetIDs), nil
}
func (r *backupExposureRepo) FindProvenance(ctx context.Context, assetID uuid.UUID) (map[string]model.ExposureProvenance, error) {
	return r.provenance[assetID.String()], nil
}
func (r *backupExposureRepo) FindProvenanceByAssets(ctx context.Context, assetIDs []uuid.UUID) (map[string]map[string]model.ExposureProvenance, error) {
	out := map[string]map[string]model.ExposureProvenance{}
	for _, id := range assetIDs {
		if prov := r.provenance[id.String()]; len(prov) > 0 {
			out[id.String()] = prov
		}
	}
	return out, nil
}
func (r *backupExposureRepo) SetProvenance(ctx context.Context, assetID uuid.UUID, dimension, source string) error {
	key := assetID.String()
	if r.provenance[key] == nil {
		r.provenance[key] = map[string]model.ExposureProvenance{}
	}
	r.provenance[key][dimension] = model.ExposureProvenance{Source: source, UpdatedAt: time.Now().UTC()}
	return nil
}

// backupPriceRepo implements repository.PriceRepository over backupStore,
// upserting on (asset_id, date) like the real ON CONFLICT clause.

type backupPriceRepo backupStore

func (r *backupPriceRepo) Create(ctx context.Context, price *model.Price) (*model.Price, error) {
	for _, p := range r.prices {
		if p.AssetID == price.AssetID && p.Date.Equal(price.Date) {
			p.Open, p.High, p.Low, p.Close, p.Volume, p.Source = price.Open, price.High, price.Low, price.Close, price.Volume, price.Source
			return p, nil
		}
	}
	stored := *price
	stored.ID = uuid.New()
	stored.CreatedAt = time.Now().UTC()
	r.prices = append(r.prices, &stored)
	return &stored, nil
}
func (r *backupPriceRepo) FindByAsset(ctx context.Context, assetID uuid.UUID) ([]*model.Price, error) {
	var out []*model.Price
	for _, p := range r.prices {
		if p.AssetID == assetID {
			out = append(out, p)
		}
	}
	return out, nil
}
func (r *backupPriceRepo) FindLatest(ctx context.Context, assetID uuid.UUID) (*model.Price, error) {
	return nil, nil
}
func (r *backupPriceRepo) FindLatestForAssets(ctx context.Context, assetIDs []uuid.UUID) (map[uuid.UUID]*model.Price, error) {
	return nil, nil
}
func (r *backupPriceRepo) FindManualByAssets(ctx context.Context, assetIDs []uuid.UUID) ([]*model.Price, error) {
	var out []*model.Price
	for _, p := range r.prices {
		if p.Source != "manual" {
			continue
		}
		for _, id := range assetIDs {
			if p.AssetID == id {
				out = append(out, p)
				break
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].AssetID.String() != out[j].AssetID.String() {
			return out[i].AssetID.String() < out[j].AssetID.String()
		}
		return out[i].Date.Before(out[j].Date)
	})
	return out, nil
}
func (r *backupPriceRepo) FindForPortfolio(ctx context.Context, portfolioID uuid.UUID) ([]model.Price, error) {
	return nil, nil
}
func (r *backupPriceRepo) MinMaxDate(ctx context.Context, assetID uuid.UUID) (*time.Time, *time.Time, error) {
	return nil, nil, nil
}
func (r *backupPriceRepo) ReferenceCloses(ctx context.Context, assetID uuid.UUID, dates []time.Time) (map[time.Time]decimal.Decimal, error) {
	return nil, nil
}

// backupCurrencyRepo implements repository.CurrencyRepository over backupStore.

type backupCurrencyRepo backupStore

func (r *backupCurrencyRepo) ListEnabled(ctx context.Context) ([]model.Currency, error) {
	return nil, nil
}
func (r *backupCurrencyRepo) ListAll(ctx context.Context) ([]model.Currency, error) { return nil, nil }
func (r *backupCurrencyRepo) Get(ctx context.Context, code string) (*model.Currency, error) {
	if c := r.currencies[code]; c != nil {
		return c, nil
	}
	return nil, nil
}
func (r *backupCurrencyRepo) Create(ctx context.Context, c *model.Currency) error {
	stored := *c
	r.currencies[stored.Code] = &stored
	return nil
}
func (r *backupCurrencyRepo) Delete(ctx context.Context, code string) error { return nil }
func (r *backupCurrencyRepo) EnabledByCodes(ctx context.Context, codes []string) ([]string, error) {
	return nil, nil
}
func (r *backupCurrencyRepo) CountInUse(ctx context.Context, code string) (int, error) {
	return 0, nil
}

// --- fixtures ---

// backupFixture seeds an original account with two owned portfolios, a
// shared one that must stay out of the bundle, manual and Yahoo prices,
// exposure with provenance, and a currency in use that is not in the
// global whitelist yet.
type backupFixture struct {
	store     *backupStore
	userID    uuid.UUID
	vwceID    uuid.UUID
	aaplID    uuid.UUID
	cashID    uuid.UUID
	coreID    uuid.UUID
	retireID  uuid.UUID
	sharedID  uuid.UUID
	otherUser uuid.UUID
}

func newBackupFixture(t *testing.T) *backupFixture {
	t.Helper()
	s := newBackupStore()
	f := &backupFixture{store: s}
	f.userID = s.seedUser("alice@example.com", "Alice", "EUR")
	f.otherUser = s.seedUser("carol@example.com", "Carol", "USD")

	f.vwceID = s.seedAsset("VWCE", "Vanguard Global Aggregate", model.AssetTypeETF, "EUR", "IE00B4L5Y983", "manual", "equity")
	f.aaplID = s.seedAsset("AAPL", "Apple Inc.", model.AssetTypeStock, "USD", "", "yahoo", "equity")
	f.cashID = s.seedAsset("CHF-CASH", "Swiss cash", model.AssetTypeCash, "CHF", "", "none", "currency")
	s.seedAsset("ORPHAN", "Never transacted", model.AssetTypeStock, "USD", "", "yahoo", "equity")

	f.coreID = s.seedPortfolio(f.userID, "Core", "EUR", backupDay(2024, 1, 10))
	f.retireID = s.seedPortfolio(f.userID, "Retirement", "USD", backupDay(2024, 3, 2))
	f.sharedID = s.seedPortfolio(f.otherUser, "Shared with Alice", "EUR", backupDay(2024, 4, 1))
	s.shares[f.sharedID] = []uuid.UUID{f.userID}

	s.seedTx(f.coreID, f.aaplID, model.TxBuy, "10", "182.5", "1.5", backupDay(2024, 2, 1), "")
	s.seedTx(f.coreID, f.vwceID, model.TxBuy, "100", "98.25", "0", backupDay(2024, 2, 15), "first batch")
	s.seedTx(f.coreID, f.vwceID, model.TxDividend, "0", "12.34", "0", backupDay(2024, 6, 1), "")
	s.seedTx(f.retireID, f.cashID, model.TxBuy, "1000", "1", "0", backupDay(2024, 4, 15), "")
	s.seedTx(f.sharedID, f.aaplID, model.TxBuy, "1", "190", "0", backupDay(2024, 5, 5), "")

	s.seedPrice(f.vwceID, backupDay(2024, 2, 15), "98.25", "manual")
	s.seedPrice(f.vwceID, backupDay(2024, 7, 1), "104.5", "manual")
	s.seedPrice(f.vwceID, backupDay(2024, 7, 2), "105.0", "yahoo")
	s.seedPrice(f.aaplID, backupDay(2024, 2, 1), "182.5", "yahoo")

	s.seedExposure(f.vwceID, model.ExposureDimensionCountries, map[string]string{"United States": "60", "Japan": "5", "Germany": "10"}, "justetf")
	s.seedExposure(f.vwceID, model.ExposureDimensionRegions, map[string]string{"North America": "60", "Europe": "35", "Asia": "5"}, "justetf")
	s.seedExposure(f.aaplID, model.ExposureDimensionCountries, map[string]string{"United States": "100"}, "yahoo")

	s.seedCurrency("EUR", "Euro", "€")
	s.seedCurrency("USD", "US Dollar", "$")
	return f
}

// wireBackup exports the fixture's bundle and pushes it through the JSON
// encoding, the same path the download/upload pair runs on.
func wireBackup(t *testing.T, f *backupFixture) *model.UserBackup {
	t.Helper()
	doc, err := buildUserBackup(context.Background(), f.store.repos(), f.userID)
	if err != nil {
		t.Fatalf("buildUserBackup: %v", err)
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var wire model.UserBackup
	if err := json.Unmarshal(raw, &wire); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return &wire
}

// --- tests ---

func TestBuildUserBackup_BundleShape(t *testing.T) {
	f := newBackupFixture(t)
	doc, err := buildUserBackup(context.Background(), f.store.repos(), f.userID)
	if err != nil {
		t.Fatalf("buildUserBackup: %v", err)
	}

	if doc.Version != model.UserBackupVersion {
		t.Errorf("version = %d, want %d", doc.Version, model.UserBackupVersion)
	}
	if doc.User.Email != "alice@example.com" || doc.User.Name != "Alice" || doc.User.BaseCurrency != "EUR" {
		t.Errorf("user block = %+v", doc.User)
	}
	if doc.Portfolios == nil || doc.Assets == nil || doc.Currencies == nil {
		t.Errorf("bundle slices must be non-nil so the JSON shows [] not null")
	}

	// The shared portfolio of another user stays out; owned ones come in
	// creation order.
	if len(doc.Portfolios) != 2 {
		t.Fatalf("portfolios = %d, want 2 (%+v)", len(doc.Portfolios), doc.Portfolios)
	}
	if doc.Portfolios[0].Portfolio.Name != "Core" || doc.Portfolios[1].Portfolio.Name != "Retirement" {
		t.Errorf("portfolio order = %q, %q, want Core, Retirement", doc.Portfolios[0].Portfolio.Name, doc.Portfolios[1].Portfolio.Name)
	}
	core := doc.Portfolios[0]
	if core.Version != 1 || core.Portfolio.Currency != "EUR" || core.Portfolio.Description != "" {
		t.Errorf("core export = %+v", core.Portfolio)
	}
	if len(core.Transactions) != 3 || core.Transactions[0].Type != model.TxBuy || core.Transactions[0].AssetTicker != "AAPL" {
		t.Errorf("core transactions = %+v", core.Transactions)
	}
	if core.Transactions[1].Notes != "first batch" || !core.Transactions[1].Fees.Equal(decimal.Zero) {
		t.Errorf("core transaction notes/fees = %+v", core.Transactions[1])
	}
	// Per-portfolio assets are only the referenced ones, ticker-sorted.
	if len(core.Assets) != 2 || core.Assets[0].Ticker != "AAPL" || core.Assets[1].Ticker != "VWCE" {
		t.Errorf("core assets = %+v", core.Assets)
	}
	if core.Assets[1].ISIN != "IE00B4L5Y983" || core.Assets[1].PriceSource != "manual" || core.Assets[1].AssetClass != "equity" {
		t.Errorf("core asset metadata = %+v", core.Assets[1])
	}
	if len(doc.Portfolios[1].Transactions) != 1 || doc.Portfolios[1].Transactions[0].AssetTicker != "CHF-CASH" {
		t.Errorf("retirement transactions = %+v, want the single CHF cash buy", doc.Portfolios[1].Transactions)
	}
	if len(doc.Portfolios[1].Assets) != 1 || doc.Portfolios[1].Assets[0].Ticker != "CHF-CASH" {
		t.Errorf("retirement assets = %+v", doc.Portfolios[1].Assets)
	}

	// Top-level assets: referenced ones only (no ORPHAN, no shared-only),
	// ticker-sorted.
	if len(doc.Assets) != 3 || doc.Assets[0].Ticker != "AAPL" || doc.Assets[1].Ticker != "CHF-CASH" || doc.Assets[2].Ticker != "VWCE" {
		t.Fatalf("bundle assets = %+v", doc.Assets)
	}

	vwce := doc.Assets[2]
	if vwce.Exposure == nil {
		t.Fatal("VWCE exposure missing")
	}
	if len(vwce.Exposure.Countries) != 3 || vwce.Exposure.Countries[0].Name != "Germany" {
		t.Errorf("VWCE countries = %+v", vwce.Exposure.Countries)
	}
	if vwce.Exposure.CountriesSource != "justetf" || vwce.Exposure.RegionsSource != "justetf" || vwce.Exposure.SectorsSource != "" {
		t.Errorf("VWCE exposure sources = %+v", vwce.Exposure)
	}
	if vwce.Exposure.Sectors != nil && len(vwce.Exposure.Sectors) != 0 {
		t.Errorf("VWCE sectors = %+v, want none stored", vwce.Exposure.Sectors)
	}
	if vwce.Exposure.Provenance != nil {
		t.Errorf("Provenance is an API output field and must stay out of the bundle, got %+v", vwce.Exposure.Provenance)
	}

	// Manual prices only: the Yahoo row of the same asset is refetchable.
	if len(vwce.ManualPrices) != 2 {
		t.Fatalf("VWCE manual prices = %+v, want 2 rows", vwce.ManualPrices)
	}
	if vwce.ManualPrices[0].Date.Day() != 15 || !vwce.ManualPrices[0].Close.Equal(dec("98.25")) || vwce.ManualPrices[0].Source != "manual" {
		t.Errorf("VWCE price row = %+v", vwce.ManualPrices[0])
	}
	// AAPL carries only a Yahoo price and a stored countries dimension; the
	// cash asset carries neither exposure nor prices.
	aapl := doc.Assets[0]
	if aapl.ManualPrices != nil {
		t.Errorf("AAPL must carry no manual prices (Yahoo-only asset): %+v", aapl.ManualPrices)
	}
	if aapl.Exposure == nil || len(aapl.Exposure.Countries) != 1 || aapl.Exposure.Countries[0].Name != "United States" || aapl.Exposure.CountriesSource != "yahoo" {
		t.Errorf("AAPL exposure = %+v, want stored countries with yahoo source", aapl.Exposure)
	}
	cash := doc.Assets[1]
	if cash.Exposure != nil || cash.ManualPrices != nil {
		t.Errorf("CHF-CASH should carry no exposure or manual prices: %+v", cash)
	}

	// Currencies in use, sorted; CHF is not in the global whitelist so it
	// carries the bare code.
	want := []model.ExportCurrency{{Code: "CHF"}, {Code: "EUR", Name: "Euro", Symbol: "€"}, {Code: "USD", Name: "US Dollar", Symbol: "$"}}
	if !reflect.DeepEqual(doc.Currencies, want) {
		t.Errorf("currencies = %+v, want %+v", doc.Currencies, want)
	}
}

func TestBuildUserBackup_EmptyUser(t *testing.T) {
	s := newBackupStore()
	userID := s.seedUser("empty@example.com", "Empty", "EUR")
	s.seedCurrency("EUR", "Euro", "€")
	doc, err := buildUserBackup(context.Background(), s.repos(), userID)
	if err != nil {
		t.Fatalf("buildUserBackup: %v", err)
	}
	if len(doc.Portfolios) != 0 || len(doc.Assets) != 0 {
		t.Errorf("empty user bundle = %+v", doc)
	}
	if len(doc.Currencies) != 1 || doc.Currencies[0].Code != "EUR" || doc.Currencies[0].Name != "Euro" {
		t.Errorf("currencies = %+v, want the base currency only", doc.Currencies)
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"portfolios":[]`) || !strings.Contains(string(raw), `"assets":[]`) {
		t.Errorf("empty bundle must serialize empty arrays, got %s", raw)
	}
}

func TestUserBackupRestore_RoundTrip(t *testing.T) {
	f := newBackupFixture(t)
	wire := wireBackup(t, f)

	target := newBackupStore()
	userID := target.seedUser("bob@example.com", "Bob", "USD")
	target.seedCurrency("EUR", "Euro", "€")
	target.seedCurrency("USD", "US Dollar", "$")

	summary := &model.BackupRestoreSummary{Mode: BackupModeAdd}
	created, err := applyUserBackup(context.Background(), target.repos(), userID, wire, BackupModeAdd, summary)
	if err != nil {
		t.Fatalf("applyUserBackup: %v", err)
	}
	if len(created) != 2 {
		t.Errorf("created portfolios = %d, want 2", len(created))
	}
	want := model.BackupRestoreSummary{Mode: BackupModeAdd, PortfoliosCreated: 2, TransactionsCreated: 4, AssetsCreated: 3, AssetsReused: 0}
	if *summary != want {
		t.Errorf("summary = %+v, want %+v", *summary, want)
	}

	// Base currency and the missing whitelist entry come back.
	if got := target.users[userID].BaseCurrency; got != "EUR" {
		t.Errorf("base currency = %q, want EUR", got)
	}
	chf := target.currencies["CHF"]
	if chf == nil || !chf.Enabled || chf.Name != "CHF" {
		t.Errorf("CHF whitelist entry = %+v, want created enabled with code as name", chf)
	}
	// Existing entries are never touched.
	if eur := target.currencies["EUR"]; eur == nil || eur.Name != "Euro" {
		t.Errorf("EUR entry changed: %+v", eur)
	}

	// Portfolios, transactions and asset resolution land correctly.
	bob := target.portfolioForUser(userID)
	var core, retire *model.Portfolio
	for _, p := range bob {
		switch p.Name {
		case "Core":
			core = p
		case "Retirement":
			retire = p
		}
	}
	if core == nil || retire == nil || len(bob) != 2 {
		t.Fatalf("restored portfolios = %+v", bob)
	}
	if core.Currency != "EUR" || retire.Currency != "USD" {
		t.Errorf("restored portfolio currencies = %q / %q", core.Currency, retire.Currency)
	}
	coreTxs := target.txsOf(core.ID)
	if len(coreTxs) != 3 {
		t.Fatalf("core transactions = %d, want 3", len(coreTxs))
	}
	vwce := target.assetByTicker("VWCE")
	aapl := target.assetByTicker("AAPL")
	if vwce == nil || aapl == nil || target.assetByTicker("ORPHAN") != nil {
		t.Fatalf("restored assets: vwce=%v aapl=%v", vwce, aapl)
	}
	if vwce.Name != "Vanguard Global Aggregate" || vwce.Type != model.AssetTypeETF || vwce.PriceSource != "manual" || vwce.ISIN != "IE00B4L5Y983" || vwce.AssetClass != "equity" {
		t.Errorf("restored VWCE identity = %+v", vwce)
	}
	for _, tx := range coreTxs {
		if tx.Type == model.TxDividend && tx.AssetID != vwce.ID {
			t.Errorf("dividend attached to wrong asset")
		}
		if tx.Type == model.TxBuy && tx.Notes == "first batch" && tx.AssetID != vwce.ID {
			t.Errorf("buy with notes attached to wrong asset")
		}
		if tx.AssetTicker == "" {
			t.Errorf("transaction %v joined no asset", tx)
		}
	}

	// Exposure and manual prices are restored onto the new assets; Yahoo
	// prices never enter the bundle.
	vwceKey := vwce.ID.String()
	if len(target.countries[vwceKey]) != 3 || len(target.regions[vwceKey]) != 3 {
		t.Errorf("restored exposure = countries %+v regions %+v", target.countries[vwceKey], target.regions[vwceKey])
	}
	if got := target.provenance[vwceKey][model.ExposureDimensionCountries].Source; got != "justetf" {
		t.Errorf("restored countries provenance = %q, want justetf", got)
	}
	manual := 0
	for _, p := range target.prices {
		if p.AssetID == vwce.ID && p.Source == "manual" {
			manual++
		}
	}
	if manual != 2 {
		t.Errorf("restored manual prices = %d, want 2", manual)
	}

	// A second export of the restored store, pushed through JSON and
	// normalized, is deep-equal to the bundle that produced it: the round
	// trip is lossless for everything the bundle carries.
	docB, err := buildUserBackup(context.Background(), target.repos(), userID)
	if err != nil {
		t.Fatalf("re-export: %v", err)
	}
	rawB, _ := json.Marshal(docB)
	var wire2 model.UserBackup
	if err := json.Unmarshal(rawB, &wire2); err != nil {
		t.Fatal(err)
	}
	normalizeBackupDoc(wire)
	normalizeBackupDoc(&wire2)
	// Restore fills a missing whitelist name with the code, so the first
	// export (CHF unknown to the whitelist) gains it on the round trip.
	for i := range wire.Currencies {
		if wire.Currencies[i].Name == "" {
			wire.Currencies[i].Name = wire.Currencies[i].Code
		}
	}
	if !reflect.DeepEqual(wire, &wire2) {
		t.Errorf("round trip not stable:\nfirst:  %+v\nsecond: %+v", wire, wire2)
	}
}

func normalizeBackupDoc(doc *model.UserBackup) {
	doc.ExportedAt = time.Time{}
	doc.User.Email = ""
	doc.User.Name = ""
	for i := range doc.Portfolios {
		doc.Portfolios[i].ExportedAt = time.Time{}
	}
	// Restored portfolios share a single creation instant (the restore
	// transaction's now()), so tie-break ordering is not a round-trip
	// invariant; content is. Compare in name order.
	sort.Slice(doc.Portfolios, func(i, j int) bool { return doc.Portfolios[i].Portfolio.Name < doc.Portfolios[j].Portfolio.Name })
}

func TestUserBackupRestore_AddKeepsExistingAndReusesAssets(t *testing.T) {
	// A hand-crafted bundle referencing two tickers that already exist on
	// the target plus one portfolio, so reuse vs create stays readable.
	doc := &model.UserBackup{
		Version: model.UserBackupVersion,
		User:    model.BackupUser{BaseCurrency: "EUR"},
		Portfolios: []model.PortfolioExport{{
			Version:   1,
			Portfolio: model.ExportPortfolio{Name: "Restored", Currency: "EUR"},
			Transactions: []model.ExportTransaction{
				{Date: backupDay(2024, 2, 15), Type: model.TxBuy, AssetTicker: "VWCE", Quantity: dec("100"), Price: dec("98.25")},
				{Date: backupDay(2024, 2, 1), Type: model.TxBuy, AssetTicker: "AAPL", Quantity: dec("10"), Price: dec("182.5")},
			},
		}},
		Assets: []model.BackupAsset{{
			ExportAsset: model.ExportAsset{Ticker: "VWCE", Name: "Vanguard Global Aggregate", Type: model.AssetTypeETF, Currency: "EUR", ISIN: "IE00B4L5Y983", PriceSource: "manual", AssetClass: "equity"},
			Exposure: &model.AssetExposure{
				Countries:       []model.ExposureRow{{Name: "United States", Weight: dec("60")}, {Name: "Germany", Weight: dec("10")}, {Name: "Japan", Weight: dec("5")}},
				CountriesSource: "justetf",
				Regions:         []model.ExposureRow{{Name: "North America", Weight: dec("60")}, {Name: "Europe", Weight: dec("35")}, {Name: "Asia", Weight: dec("5")}},
				RegionsSource:   "justetf",
			},
			ManualPrices: []model.ExportPrice{
				{Date: backupDay(2024, 2, 15), Open: dec("98.25"), High: dec("98.25"), Low: dec("98.25"), Close: dec("98.25"), Source: "manual"},
				{Date: backupDay(2024, 7, 1), Open: dec("104.5"), High: dec("104.5"), Low: dec("104.5"), Close: dec("104.5"), Source: "manual"},
			},
		}},
	}

	// The target already knows both tickers under its own (shared,
	// unmodifiable) identity data, plus a legacy portfolio and its own
	// exposure/price history for VWCE.
	s := newBackupStore()
	userID := s.seedUser("bob@example.com", "Bob", "USD")
	vwce := s.seedAsset("VWCE", "Existing VWCE", model.AssetTypeETF, "CHF", "IE00DIFFERENT", "none", "mixed")
	aapl := s.seedAsset("AAPL", "Apple Inc.", model.AssetTypeStock, "USD", "", "yahoo", "equity")
	existingPF := s.seedPortfolio(userID, "Legacy", "EUR", backupDay(2023, 1, 1))
	s.seedTx(existingPF, aapl, model.TxBuy, "1", "100", "0", backupDay(2023, 2, 1), "")
	vwceKey := vwce.String()
	s.seedExposure(vwce, model.ExposureDimensionCountries, map[string]string{"Switzerland": "100"}, "manual")
	s.seedExposure(vwce, model.ExposureDimensionSectors, map[string]string{"Technology": "100"}, "manual")
	s.seedPrice(vwce, backupDay(2024, 2, 15), "1.00", "manual")
	s.seedPrice(vwce, backupDay(2024, 9, 9), "9.00", "manual")

	summary := &model.BackupRestoreSummary{Mode: BackupModeAdd}
	if _, err := applyUserBackup(context.Background(), s.repos(), userID, doc, BackupModeAdd, summary); err != nil {
		t.Fatalf("applyUserBackup: %v", err)
	}
	if summary.PortfoliosCreated != 1 || summary.TransactionsCreated != 2 {
		t.Errorf("summary = %+v, want 1 portfolio and 2 transactions created", summary)
	}
	if summary.AssetsReused != 2 || summary.AssetsCreated != 0 {
		t.Errorf("asset counts = created %d reused %d, want 0 created 2 reused", summary.AssetsCreated, summary.AssetsReused)
	}

	// Non-destructive: the legacy portfolio and its transactions survive.
	if _, ok := s.portfolios[existingPF]; !ok {
		t.Fatal("add mode must not delete existing portfolios")
	}
	if len(s.txsOf(existingPF)) != 1 {
		t.Errorf("legacy transactions lost")
	}
	// Reused assets keep their shared identity untouched…
	storedVWCE := s.assetByTicker("VWCE")
	if storedVWCE.Name != "Existing VWCE" || storedVWCE.Currency != "CHF" || storedVWCE.PriceSource != "none" || storedVWCE.ISIN != "IE00DIFFERENT" {
		t.Errorf("reused asset identity overwritten: %+v", storedVWCE)
	}
	// …and the imported transactions attach to the existing asset ids.
	restoredPF := s.portfolioForUser(userID)
	var imported *model.Portfolio
	for _, p := range restoredPF {
		if p.Name == "Restored" {
			imported = p
		}
	}
	if imported == nil {
		t.Fatal("Restored portfolio missing")
	}
	for _, tx := range s.txsOf(imported.ID) {
		wantID := vwce
		if tx.AssetTicker == "AAPL" {
			wantID = aapl
		}
		if tx.AssetID != wantID {
			t.Errorf("transaction %s attached to %s, want the pre-existing asset %s", tx.AssetTicker, tx.AssetID, wantID)
		}
	}
	// …while the bundle's exposure and manual prices are restored onto it.
	restored := s.countries[vwceKey]
	if len(restored) != 3 || restored[0].Name != "Germany" {
		t.Errorf("countries not replaced: %+v", restored)
	}
	if got := s.provenanceOf(vwceKey, model.ExposureDimensionCountries); got != "justetf" {
		t.Errorf("countries provenance = %q, want the bundle's justetf", got)
	}
	if len(s.regions[vwceKey]) != 3 {
		t.Errorf("regions not restored: %+v", s.regions[vwceKey])
	}
	// The bundle carries no sectors for VWCE: stored sectors survive.
	sectors := s.sectors[vwceKey]
	if len(sectors) != 1 || sectors[0].Name != "Technology" {
		t.Errorf("untouched dimension overwritten: %+v", sectors)
	}
	if got := s.provenanceOf(vwceKey, model.ExposureDimensionSectors); got != "manual" {
		t.Errorf("sectors provenance = %q, want the stored manual", got)
	}
	var d15 *model.Price
	var d99 *model.Price
	for _, p := range s.prices {
		if p.AssetID == vwce && p.Date.Equal(backupDay(2024, 2, 15)) {
			d15 = p
		}
		if p.AssetID == vwce && p.Date.Equal(backupDay(2024, 9, 9)) {
			d99 = p
		}
	}
	if d15 == nil || !d15.Close.Equal(dec("98.25")) || d15.Source != "manual" {
		t.Errorf("manual price not upserted on (asset, date): %+v", d15)
	}
	if d99 == nil || !d99.Close.Equal(dec("9.00")) {
		t.Errorf("price outside the bundle changed: %+v", d99)
	}
	if len(s.prices) != 3 {
		t.Errorf("prices total = %d, want 3: D15 upserted in place, Jul-01 added, Sep-09 outside the bundle", len(s.prices))
	}
}

func (s *backupStore) provenanceOf(assetKey, dimension string) string {
	return s.provenance[assetKey][dimension].Source
}

func TestUserBackupRestore_ReplaceDeletesOwnedOnly(t *testing.T) {
	f := newBackupFixture(t)
	wire := wireBackup(t, f)

	target := newBackupStore()
	userID := target.seedUser("bob@example.com", "Bob", "USD")
	otherID := target.seedUser("carol@example.com", "Carol", "USD")
	mineID := target.seedPortfolio(userID, "Mine", "EUR", backupDay(2023, 5, 1))
	theirsID := target.seedPortfolio(otherID, "Theirs", "EUR", backupDay(2023, 5, 2))
	sharedID := target.seedPortfolio(otherID, "Shared with Bob", "EUR", backupDay(2023, 5, 3))
	target.shares[sharedID] = []uuid.UUID{userID}

	assetID := target.seedAsset("OLD", "Old asset", model.AssetTypeStock, "USD", "", "yahoo", "equity")
	target.seedTx(mineID, assetID, model.TxBuy, "1", "1", "0", backupDay(2023, 6, 1), "")
	target.seedTx(theirsID, assetID, model.TxBuy, "1", "1", "0", backupDay(2023, 6, 1), "")
	target.seedTx(sharedID, assetID, model.TxBuy, "1", "1", "0", backupDay(2023, 6, 1), "")

	summary := &model.BackupRestoreSummary{Mode: BackupModeReplace}
	if _, err := applyUserBackup(context.Background(), target.repos(), userID, wire, BackupModeReplace, summary); err != nil {
		t.Fatalf("applyUserBackup: %v", err)
	}

	if _, ok := target.portfolios[mineID]; ok {
		t.Error("replace must delete the user's own portfolio")
	}
	if len(target.txsOf(mineID)) != 0 {
		t.Error("replace must cascade the deleted portfolio's transactions")
	}
	for _, keep := range []uuid.UUID{theirsID, sharedID} {
		if _, ok := target.portfolios[keep]; !ok {
			t.Errorf("portfolio %s belonging to another user was deleted", keep)
		}
	}
	// Bundle portfolios are created as the caller's own.
	owned := 0
	for _, p := range target.portfolios {
		if p.UserID == userID {
			owned++
		}
	}
	if owned != 2 {
		t.Errorf("owned portfolios = %d, want 2", owned)
	}
	if summary.PortfoliosCreated != 2 || summary.TransactionsCreated != 4 {
		t.Errorf("summary = %+v, want 2 portfolios and 4 transactions created", summary)
	}
}

func TestUserBackupRestore_EmptyBundle(t *testing.T) {
	s := newBackupStore()
	userID := s.seedUser("bob@example.com", "Bob", "USD")
	doc := &model.UserBackup{Version: model.UserBackupVersion}
	summary := &model.BackupRestoreSummary{Mode: BackupModeAdd}
	created, err := applyUserBackup(context.Background(), s.repos(), userID, doc, BackupModeAdd, summary)
	if err != nil {
		t.Fatalf("applyUserBackup: %v", err)
	}
	if len(created) != 0 {
		t.Errorf("created = %v, want none", created)
	}
	if *summary != (model.BackupRestoreSummary{Mode: BackupModeAdd}) {
		t.Errorf("summary = %+v, want all zero", *summary)
	}
	if s.users[userID].BaseCurrency != "USD" {
		t.Errorf("empty bundle must not touch the user's base currency")
	}
}

func TestUserBackupRestore_ImportDefaultsForUnknownTickers(t *testing.T) {
	s := newBackupStore()
	userID := s.seedUser("bob@example.com", "Bob", "USD")
	doc := &model.UserBackup{
		Version: model.UserBackupVersion,
		Portfolios: []model.PortfolioExport{{
			Version:   1,
			Portfolio: model.ExportPortfolio{Name: "P", Currency: "USD"},
			Assets: []model.ExportAsset{{
				Ticker: "ZZZ", ISIN: "", PriceSource: "coingecko", // unknown source
			}},
			Transactions: []model.ExportTransaction{{
				Date: backupDay(2024, 1, 1), Type: model.TxBuy, AssetTicker: "ZZZ", Quantity: dec("1"), Price: dec("2"),
			}},
		}},
	}
	summary := &model.BackupRestoreSummary{Mode: BackupModeAdd}
	if _, err := applyUserBackup(context.Background(), s.repos(), userID, doc, BackupModeAdd, summary); err != nil {
		t.Fatalf("applyUserBackup: %v", err)
	}
	a := s.assetByTicker("ZZZ")
	if a == nil {
		t.Fatal("ZZZ not created")
	}
	if a.Name != "ZZZ" || a.Type != model.AssetTypeStock || a.Currency != "USD" || a.PriceSource != "yahoo" || a.AssetClass != "equity" {
		t.Errorf("created asset defaults = %+v", a)
	}
	if summary.AssetsCreated != 1 || summary.AssetsReused != 0 {
		t.Errorf("counts = %+v", summary)
	}
}

func TestUserBackupRestore_Validation(t *testing.T) {
	ctx := context.Background()
	s := newBackupStore()
	userID := s.seedUser("bob@example.com", "Bob", "USD")
	rx := s.repos()
	okDoc := func() *model.UserBackup {
		return &model.UserBackup{
			Version: model.UserBackupVersion,
			Portfolios: []model.PortfolioExport{{
				Version:   1,
				Portfolio: model.ExportPortfolio{Name: "P", Currency: "USD"},
			}},
		}
	}

	if _, err := normalizeBackupMode(""); err != nil {
		t.Errorf("empty mode: %v, want default add", err)
	}
	if m, _ := normalizeBackupMode(" REPLACE "); m != BackupModeReplace {
		t.Errorf("mode normalization = %q, want replace", m)
	}
	if _, err := normalizeBackupMode("delete"); err == nil || !isInvalidInput(err) {
		t.Errorf("unknown mode error = %v, want ErrInvalidInput", err)
	}

	if err := validateUserBackup(nil); !isInvalidInput(err) {
		t.Errorf("nil doc = %v, want ErrInvalidInput", err)
	}
	doc := okDoc()
	doc.Version = model.UserBackupVersion + 1
	err := validateUserBackup(doc)
	if !isInvalidInput(err) || !strings.Contains(err.Error(), "unsupported backup version 2") {
		t.Errorf("newer version = %v, want explicit unsupported-version error", err)
	}
	doc = okDoc()
	doc.Version = 0
	if err := validateUserBackup(doc); !isInvalidInput(err) {
		t.Errorf("missing version = %v, want ErrInvalidInput", err)
	}
	doc = okDoc()
	doc.Portfolios[0].Version = 2
	err = validateUserBackup(doc)
	if !isInvalidInput(err) || !strings.Contains(err.Error(), "unsupported portfolio export version 2") {
		t.Errorf("newer nested version = %v, want explicit error", err)
	}
	doc = okDoc()
	doc.Portfolios[0].Portfolio.Name = "   "
	if err := validateUserBackup(doc); !isInvalidInput(err) {
		t.Errorf("unnamed portfolio = %v, want ErrInvalidInput", err)
	}
	doc = okDoc()
	doc.Assets = []model.BackupAsset{{ExportAsset: model.ExportAsset{}}}
	if err := validateUserBackup(doc); !isInvalidInput(err) {
		t.Errorf("asset without ticker = %v, want ErrInvalidInput", err)
	}

	doc = okDoc()
	doc.Portfolios[0].Transactions = []model.ExportTransaction{{Type: model.TxBuy, Quantity: dec("1"), Price: dec("1")}}
	if _, err := applyUserBackup(ctx, rx, userID, doc, BackupModeAdd, &model.BackupRestoreSummary{}); !isInvalidInput(err) {
		t.Errorf("transaction without ticker = %v, want ErrInvalidInput", err)
	}

	doc = okDoc()
	doc.Portfolios[0].Transactions = []model.ExportTransaction{{AssetTicker: "AAPL", Type: "magic", Quantity: dec("1"), Price: dec("1")}}
	if _, err := applyUserBackup(ctx, rx, userID, doc, BackupModeAdd, &model.BackupRestoreSummary{}); !isInvalidInput(err) {
		t.Errorf("invalid transaction type = %v, want ErrInvalidInput", err)
	}
}

func isInvalidInput(err error) bool {
	return errors.Is(err, ErrInvalidInput)
}
