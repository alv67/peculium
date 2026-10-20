package service

import (
	"context"
	"errors"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/alv67/peculium/internal/cache"
	"github.com/alv67/peculium/internal/model"
	"github.com/alv67/peculium/internal/repository"
)

// manualTestPriceRepo implements the manual-price subset of
// repository.PriceRepository over an in-memory store, upserting on
// (asset_id, date) like the real ON CONFLICT clause and filtering
// source='manual' like the real queries.
type manualTestPriceRepo struct {
	repository.PriceRepository
	rows        []*model.Price
	createCalls int
}

func (f *manualTestPriceRepo) Create(ctx context.Context, price *model.Price) (*model.Price, error) {
	f.createCalls++
	for _, p := range f.rows {
		if p.AssetID == price.AssetID && p.Date.Equal(price.Date) {
			p.Open, p.High, p.Low, p.Close, p.Volume, p.Source = price.Open, price.High, price.Low, price.Close, price.Volume, price.Source
			return p, nil
		}
	}
	stored := *price
	f.rows = append(f.rows, &stored)
	return &stored, nil
}

func (f *manualTestPriceRepo) FindManualByAssets(ctx context.Context, assetIDs []uuid.UUID) ([]*model.Price, error) {
	var out []*model.Price
	for _, p := range f.rows {
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
	sort.SliceStable(out, func(i, j int) bool { return out[i].Date.Before(out[j].Date) })
	return out, nil
}

func (f *manualTestPriceRepo) DeleteManual(ctx context.Context, assetID uuid.UUID, date time.Time) (int64, error) {
	var kept []*model.Price
	deleted := int64(0)
	for _, p := range f.rows {
		if p.AssetID == assetID && p.Date.Equal(date) && p.Source == "manual" {
			deleted++
			continue
		}
		kept = append(kept, p)
	}
	f.rows = kept
	return deleted, nil
}

func newManualPriceTestService(t *testing.T, asset *model.Asset) (*Service, *manualTestPriceRepo) {
	t.Helper()
	prices := &manualTestPriceRepo{}
	repos := &repository.Repository{Asset: &fakeAssetRepo{asset: asset}, Price: prices}
	return New(repos, nil, nil, nil, time.Minute, time.Hour, cache.New(nil), 0, 0, nil, nil), prices
}

func TestPutManualPrice_InsertAndOverwrite(t *testing.T) {
	asset := &model.Asset{ID: uuid.New(), Ticker: "BND", PriceSource: "manual"}
	svc, prices := newManualPriceTestService(t, asset)
	ctx := context.Background()

	saved, err := svc.PutManualPrice(ctx, asset.ID, &model.ManualPriceInput{
		Date: "2026-01-15", Close: decimal.RequireFromString("103.2"),
	})
	if err != nil {
		t.Fatalf("put: unexpected error: %v", err)
	}
	if saved.Date != "2026-01-15" || saved.Close.String() != "103.2" || saved.Volume != 0 {
		t.Fatalf("saved = %+v, want date 2026-01-15 close 103.2 volume 0", saved)
	}
	if len(prices.rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(prices.rows))
	}
	row := prices.rows[0]
	if row.Source != "manual" || !row.Close.Equal(decimal.RequireFromString("103.2")) {
		t.Fatalf("stored row = %+v, want source manual and close 103.2", row)
	}
	if !row.Open.Equal(row.Close) || !row.High.Equal(row.Close) || !row.Low.Equal(row.Close) {
		t.Fatalf("stored OHLC = %s/%s/%s, want all equal to close %s", row.Open, row.High, row.Low, row.Close)
	}

	overwritten, err := svc.PutManualPrice(ctx, asset.ID, &model.ManualPriceInput{
		Date: "2026-01-15", Close: decimal.RequireFromString("99.5"), Volume: 10,
	})
	if err != nil {
		t.Fatalf("overwrite: unexpected error: %v", err)
	}
	if overwritten.Close.String() != "99.5" || overwritten.Volume != 10 {
		t.Fatalf("overwrite = %+v, want close 99.5 volume 10", overwritten)
	}
	if len(prices.rows) != 1 || prices.createCalls != 2 {
		t.Fatalf("rows = %d createCalls = %d, want 1 and 2 (same date upserts)", len(prices.rows), prices.createCalls)
	}
}

func TestPutManualPrice_Validation(t *testing.T) {
	tomorrow := time.Now().UTC().AddDate(0, 0, 1).Format("2006-01-02")
	valid := decimal.RequireFromString("1.5")
	cases := []struct {
		name string
		in   model.ManualPriceInput
	}{
		{"malformed date", model.ManualPriceInput{Date: "15/01/2026", Close: valid}},
		{"empty date", model.ManualPriceInput{Date: "", Close: valid}},
		{"future date", model.ManualPriceInput{Date: tomorrow, Close: valid}},
		{"zero close", model.ManualPriceInput{Date: "2026-01-15", Close: decimal.Zero}},
		{"negative close", model.ManualPriceInput{Date: "2026-01-15", Close: decimal.NewFromInt(-1)}},
		{"absurd close", model.ManualPriceInput{Date: "2026-01-15", Close: decimal.New(1, 13)}},
		{"negative volume", model.ManualPriceInput{Date: "2026-01-15", Close: valid, Volume: -1}},
	}
	for _, tc := range cases {
		asset := &model.Asset{ID: uuid.New(), Ticker: "BND", PriceSource: "manual"}
		svc, prices := newManualPriceTestService(t, asset)

		_, err := svc.PutManualPrice(context.Background(), asset.ID, &tc.in)
		if !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("%s: err = %v, want ErrInvalidInput", tc.name, err)
		}
		if prices.createCalls != 0 || len(prices.rows) != 0 {
			t.Fatalf("%s: rejected input stored %d rows", tc.name, len(prices.rows))
		}
	}
}

func TestPutManualPrice_YahooAssetRejectedOtherSourcesAccepted(t *testing.T) {
	for _, tc := range []struct {
		source string
		wantOK bool
	}{
		{"yahoo", false},
		{"", false}, // legacy default means Yahoo
		{"manual", true},
		{"none", true},
	} {
		asset := &model.Asset{ID: uuid.New(), Ticker: "X", PriceSource: tc.source}
		svc, _ := newManualPriceTestService(t, asset)

		_, err := svc.PutManualPrice(context.Background(), asset.ID, &model.ManualPriceInput{
			Date: "2026-01-15", Close: decimal.RequireFromString("1.5"),
		})
		if tc.wantOK && err != nil {
			t.Fatalf("price_source=%q: unexpected error: %v", tc.source, err)
		}
		if !tc.wantOK && !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("price_source=%q: err = %v, want ErrInvalidInput", tc.source, err)
		}
	}
}

func TestManualPrices_AssetNotFound(t *testing.T) {
	svc, _ := newManualPriceTestService(t, nil)

	_, err := svc.ManualPrices(context.Background(), uuid.New())
	if err != ErrAssetNotFound {
		t.Fatalf("err = %v, want ErrAssetNotFound", err)
	}
}

func TestManualPrices_NewestFirstManualOnly(t *testing.T) {
	asset := &model.Asset{ID: uuid.New(), Ticker: "BND", PriceSource: "none"}
	svc, prices := newManualPriceTestService(t, asset)
	ctx := context.Background()

	// A Yahoo bar must never surface through the manual API.
	prices.rows = append(prices.rows, &model.Price{
		AssetID: asset.ID, Date: time.Date(2026, 1, 16, 0, 0, 0, 0, time.UTC),
		Close: decimal.RequireFromString("200"), Source: "yahoo",
	})
	for _, day := range []string{"2026-01-10", "2026-01-12"} {
		if _, err := svc.PutManualPrice(ctx, asset.ID, &model.ManualPriceInput{
			Date: day, Close: decimal.RequireFromString("1.5"),
		}); err != nil {
			t.Fatalf("put %s: %v", day, err)
		}
	}

	got, err := svc.ManualPrices(ctx, asset.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 || got[0].Date != "2026-01-12" || got[1].Date != "2026-01-10" {
		t.Fatalf("rows = %+v, want manual bars newest-first", got)
	}

	// Unknown asset and missing bars: ErrAssetNotFound / empty slice.
	missing := &model.Asset{ID: uuid.New(), PriceSource: "manual"}
	svc2, _ := newManualPriceTestService(t, missing)
	empty, err := svc2.ManualPrices(ctx, missing.ID)
	if err != nil || len(empty) != 0 {
		t.Fatalf("empty asset: rows = %+v err = %v, want empty and nil", empty, err)
	}
}

func TestDeleteManualPrice(t *testing.T) {
	asset := &model.Asset{ID: uuid.New(), Ticker: "BND", PriceSource: "manual"}
	svc, prices := newManualPriceTestService(t, asset)
	ctx := context.Background()

	if _, err := svc.PutManualPrice(ctx, asset.ID, &model.ManualPriceInput{
		Date: "2026-01-15", Close: decimal.RequireFromString("1.5"),
	}); err != nil {
		t.Fatalf("put: %v", err)
	}
	prices.rows = append(prices.rows, &model.Price{
		AssetID: asset.ID, Date: time.Date(2026, 1, 16, 0, 0, 0, 0, time.UTC),
		Close: decimal.RequireFromString("200"), Source: "yahoo",
	})

	if err := svc.DeleteManualPrice(ctx, asset.ID, "2026-01-15"); err != nil {
		t.Fatalf("delete existing: unexpected error: %v", err)
	}
	if len(prices.rows) != 1 || prices.rows[0].Source != "yahoo" {
		t.Fatalf("rows after delete = %+v, want only the yahoo bar left", prices.rows)
	}
	if err := svc.DeleteManualPrice(ctx, asset.ID, "2026-01-15"); err != ErrNotFound {
		t.Fatalf("delete missing: err = %v, want ErrNotFound", err)
	}
	// A Yahoo bar is unreachable through the manual delete, same date or not.
	if err := svc.DeleteManualPrice(ctx, asset.ID, "2026-01-16"); err != ErrNotFound {
		t.Fatalf("delete yahoo bar: err = %v, want ErrNotFound", err)
	}
	if len(prices.rows) != 1 {
		t.Fatalf("yahoo bar was deleted through the manual API: %+v", prices.rows)
	}
	if err := svc.DeleteManualPrice(ctx, asset.ID, "garbage"); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("malformed date: err = %v, want ErrInvalidInput", err)
	}

	svc2, _ := newManualPriceTestService(t, nil)
	if err := svc2.DeleteManualPrice(ctx, uuid.New(), "2026-01-15"); err != ErrAssetNotFound {
		t.Fatalf("unknown asset: err = %v, want ErrAssetNotFound", err)
	}
}

func TestManualPriceBar_UntouchedBySync(t *testing.T) {
	// A manual asset carrying a user-entered bar must stay completely out of
	// the sync paths: no Yahoo call, no write to the bar.
	asset := &model.Asset{ID: uuid.New(), Ticker: "BND", PriceSource: "manual"}
	svc, prices := newManualPriceTestService(t, asset)
	if _, err := svc.PutManualPrice(context.Background(), asset.ID, &model.ManualPriceInput{
		Date: "2026-01-15", Close: decimal.RequireFromString("1.5"), Volume: 7,
	}); err != nil {
		t.Fatalf("put: %v", err)
	}

	yf := &fakeYahooFetcher{}
	sync := newSyncTestService(t, &fakeAssetRepo{assets: []*model.Asset{asset}}, &fakeTransactionRepo{}, yf)
	sync.repos.Price = prices
	if err := sync.syncAssets(context.Background(), []uuid.UUID{asset.ID}); err != nil {
		t.Fatalf("syncAssets: %v", err)
	}

	if yf.historyCalls != 0 || yf.splitCalls != 0 {
		t.Fatalf("fetcher calls = history %d splits %d, want 0 and 0 for a manual asset", yf.historyCalls, yf.splitCalls)
	}
	if prices.createCalls != 1 {
		t.Fatalf("price writes = %d, want only the manual put", prices.createCalls)
	}
	bar := prices.rows[0]
	if bar.Source != "manual" || !bar.Close.Equal(decimal.RequireFromString("1.5")) || bar.Volume != 7 {
		t.Fatalf("manual bar changed after sync: %+v", bar)
	}
}
