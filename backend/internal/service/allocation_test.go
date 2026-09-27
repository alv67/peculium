package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/alv67/peculium/internal/geo"
	"github.com/alv67/peculium/internal/model"
)

func TestGetPortfolioAllocation_MultiCurrencyWeights(t *testing.T) {
	aaplID := uuid.New()
	msftID := uuid.New()
	pf := &fakePortfolioRepo{
		portfolio: &model.Portfolio{Currency: "EUR"},
		holdings: []*model.Holding{
			holding(aaplID.String(), "USD", "", "", model.AssetTypeStock, decimal.NewFromInt(10), decimal.NewFromInt(100)),
			holding(msftID.String(), "USD", "", "", model.AssetTypeStock, decimal.NewFromInt(10), decimal.NewFromInt(200)),
		},
	}
	fx := &fakeFXRepo{rates: map[string]decimal.Decimal{"USD": decimal.NewFromInt(1), "EUR": decimal.NewFromInt(1)}}
	svc := newTestService(t, pf, &fakeExposureRepo{}, fx)

	got, err := svc.GetPortfolioAllocation(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("allocations len = %d, want 2", len(got))
	}
	if got[0].AssetID != aaplID.String() || !equalDecimal(got[0].Value, decimal.NewFromInt(1000)) {
		t.Fatalf("got[0] = %+v, want asset %s value 1000", got[0], aaplID)
	}
	if got[1].AssetID != msftID.String() || !equalDecimal(got[1].Value, decimal.NewFromInt(2000)) {
		t.Fatalf("got[1] = %+v, want asset %s value 2000", got[1], msftID)
	}
	if got[0].FXMissing || got[1].FXMissing {
		t.Fatalf("FXMissing flags = %v/%v, want false/false", got[0].FXMissing, got[1].FXMissing)
	}
	assertDecimalInDelta(t, got[0].AllocPct, decimal.RequireFromString("33.33"), "0.01", "AAPL alloc pct")
	assertDecimalInDelta(t, got[1].AllocPct, decimal.RequireFromString("66.67"), "0.01", "MSFT alloc pct")
	sum := decimal.Zero
	for _, a := range got {
		sum = sum.Add(a.AllocPct)
	}
	assertDecimalInDelta(t, sum, decimal.NewFromInt(100), "0.01", "alloc pct sum")
}

func TestGetPortfolioAllocation_MissingFXSetsFlag(t *testing.T) {
	usdID := uuid.New()
	jpyID := uuid.New()
	pf := &fakePortfolioRepo{
		portfolio: &model.Portfolio{Currency: "EUR"},
		holdings: []*model.Holding{
			holding(usdID.String(), "USD", "", "", model.AssetTypeStock, decimal.NewFromInt(10), decimal.NewFromInt(100)),
			holding(jpyID.String(), "JPY", "", "", model.AssetTypeStock, decimal.NewFromInt(10), decimal.NewFromInt(100)),
		},
	}
	fx := &fakeFXRepo{rates: map[string]decimal.Decimal{"USD": decimal.NewFromInt(1), "EUR": decimal.NewFromInt(1)}}
	svc := newTestService(t, pf, &fakeExposureRepo{}, fx)

	got, err := svc.GetPortfolioAllocation(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("allocations len = %d, want 2", len(got))
	}
	if got[0].AssetID != usdID.String() || got[1].AssetID != jpyID.String() {
		t.Fatalf("allocation order = [%s %s], want [%s %s]", got[0].AssetID, got[1].AssetID, usdID, jpyID)
	}
	if got[0].FXMissing {
		t.Fatalf("USD FXMissing = true, want false")
	}
	assertDecimalInDelta(t, got[0].AllocPct, decimal.NewFromInt(100), "0.01", "USD alloc pct")
	if !got[1].FXMissing {
		t.Fatalf("JPY FXMissing = false, want true")
	}
	if !got[1].AllocPct.IsZero() {
		t.Fatalf("JPY alloc pct = %v, want zero", got[1].AllocPct)
	}
}

func TestGetPortfolioAllocation_SkipsZeroQtyAndNoPrice(t *testing.T) {
	validID := uuid.New()
	zeroID := uuid.New()
	noPriceID := uuid.New()
	valid := holding(validID.String(), "USD", "", "", model.AssetTypeStock, decimal.NewFromInt(5), decimal.NewFromInt(100))
	zero := holding(zeroID.String(), "USD", "", "", model.AssetTypeStock, decimal.Zero, decimal.NewFromInt(100))
	noPrice := holding(noPriceID.String(), "USD", "", "", model.AssetTypeStock, decimal.NewFromInt(5), decimal.NewFromInt(100))
	noPrice.HasPrice = false
	pf := &fakePortfolioRepo{
		portfolio: &model.Portfolio{Currency: "USD"},
		holdings:  []*model.Holding{valid, zero, noPrice},
	}
	svc := newTestService(t, pf, &fakeExposureRepo{}, &fakeFXRepo{})

	got, err := svc.GetPortfolioAllocation(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("allocations len = %d, want 1", len(got))
	}
	if got[0].AssetID != validID.String() {
		t.Fatalf("asset = %s, want %s (the only priced, positive-qty holding)", got[0].AssetID, validID)
	}
	if !equalDecimal(got[0].Value, decimal.NewFromInt(500)) {
		t.Fatalf("value = %v, want 500", got[0].Value)
	}
	if !equalDecimal(got[0].AllocPct, decimal.NewFromInt(100)) {
		t.Fatalf("alloc pct = %v, want 100", got[0].AllocPct)
	}
}

func TestGetPortfolioClassAllocation_GroupsAndSorts(t *testing.T) {
	eqID := uuid.New()
	bdID := uuid.New()
	ocID := uuid.New()
	equity := holding(eqID.String(), "EUR", "", "", model.AssetTypeStock, decimal.NewFromInt(10), decimal.NewFromInt(100))
	bond := holding(bdID.String(), "EUR", "", "", model.AssetTypeBond, decimal.NewFromInt(5), decimal.NewFromInt(100))
	bond.AssetClass = "bond"
	unclassified := holding(ocID.String(), "EUR", "", "", model.AssetTypeStock, decimal.NewFromInt(1), decimal.NewFromInt(100))
	unclassified.AssetClass = ""
	pf := &fakePortfolioRepo{
		portfolio: &model.Portfolio{Currency: "EUR"},
		holdings:  []*model.Holding{equity, bond, unclassified},
	}
	fx := &fakeFXRepo{rates: map[string]decimal.Decimal{"EUR": decimal.NewFromInt(1)}}
	svc := newTestService(t, pf, &fakeExposureRepo{}, fx)

	got, err := svc.GetPortfolioClassAllocation(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Currency != "EUR" {
		t.Fatalf("currency = %q, want EUR", got.Currency)
	}
	if len(got.Classes) != 3 {
		t.Fatalf("classes len = %d, want 3: %+v", len(got.Classes), got.Classes)
	}
	if got.Classes[0].Class != "equity" || !equalDecimal(got.Classes[0].Value, decimal.NewFromInt(1000)) {
		t.Fatalf("classes[0] = %+v, want equity 1000", got.Classes[0])
	}
	assertDecimalInDelta(t, got.Classes[0].Weight, decimal.RequireFromString("62.5"), "0.01", "equity weight")
	if got.Classes[1].Class != "bond" || !equalDecimal(got.Classes[1].Value, decimal.NewFromInt(500)) {
		t.Fatalf("classes[1] = %+v, want bond 500", got.Classes[1])
	}
	assertDecimalInDelta(t, got.Classes[1].Weight, decimal.RequireFromString("31.25"), "0.01", "bond weight")
	if got.Classes[2].Class != "other" || !equalDecimal(got.Classes[2].Value, decimal.NewFromInt(100)) {
		t.Fatalf("classes[2] = %+v, want other 100", got.Classes[2])
	}
	assertDecimalInDelta(t, got.Classes[2].Weight, decimal.RequireFromString("6.25"), "0.01", "other weight")
}

func TestGetPortfolioClassAllocation_SkipsMissingFX(t *testing.T) {
	eqID := uuid.New()
	bdID := uuid.New()
	equity := holding(eqID.String(), "EUR", "", "", model.AssetTypeStock, decimal.NewFromInt(10), decimal.NewFromInt(100))
	bond := holding(bdID.String(), "JPY", "", "", model.AssetTypeBond, decimal.NewFromInt(5), decimal.NewFromInt(100))
	bond.AssetClass = "bond"
	pf := &fakePortfolioRepo{
		portfolio: &model.Portfolio{Currency: "EUR"},
		holdings:  []*model.Holding{equity, bond},
	}
	fx := &fakeFXRepo{rates: map[string]decimal.Decimal{"EUR": decimal.NewFromInt(1)}}
	svc := newTestService(t, pf, &fakeExposureRepo{}, fx)

	got, err := svc.GetPortfolioClassAllocation(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Classes) != 1 {
		t.Fatalf("classes = %+v, want only the convertible equity bucket (the JPY holding has no FX rate)", got.Classes)
	}
	if got.Classes[0].Class != "equity" {
		t.Fatalf("class = %q, want equity", got.Classes[0].Class)
	}
	if !equalDecimal(got.Classes[0].Value, decimal.NewFromInt(1000)) {
		t.Fatalf("equity value = %v, want 1000", got.Classes[0].Value)
	}
	assertDecimalInDelta(t, got.Classes[0].Weight, decimal.NewFromInt(100), "0.01", "equity weight")
}

func TestGetPortfolioSectorAllocation_ETF(t *testing.T) {
	assetID := uuid.New()
	pf := &fakePortfolioRepo{
		portfolio: &model.Portfolio{Currency: "USD"},
		holdings: []*model.Holding{
			holding(assetID.String(), "USD", "", "", model.AssetTypeETF, decimal.NewFromInt(10), decimal.NewFromInt(100)),
		},
	}
	ex := &fakeExposureRepo{
		sectors: map[string][]model.ExposureRow{
			assetID.String(): {
				{Name: "Information Technology", Weight: decimal.NewFromInt(70)},
				{Name: "Financials", Weight: decimal.NewFromInt(30)},
			},
		},
	}
	svc := newTestService(t, pf, ex, &fakeFXRepo{})

	got, err := svc.GetPortfolioSectorAllocation(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Currency != "USD" {
		t.Fatalf("currency = %q, want USD", got.Currency)
	}
	if len(got.Sectors) != len(geo.GICSSectors) {
		t.Fatalf("sectors len = %d, want %d (fully-mapped ETF has no Other bucket)", len(got.Sectors), len(geo.GICSSectors))
	}
	for i, s := range got.Sectors {
		if s.Sector != geo.GICSSectors[i] {
			t.Fatalf("sector[%d] = %q, want %q (canonical order)", i, s.Sector, geo.GICSSectors[i])
		}
	}
	it := sectorByName(t, got.Sectors, "Information Technology")
	if !equalDecimal(it.Value, decimal.NewFromInt(700)) {
		t.Fatalf("IT value = %v, want 700", it.Value)
	}
	assertDecimalInDelta(t, it.Weight, decimal.NewFromInt(70), "0.01", "IT weight")
	fin := sectorByName(t, got.Sectors, "Financials")
	if !equalDecimal(fin.Value, decimal.NewFromInt(300)) {
		t.Fatalf("Financials value = %v, want 300", fin.Value)
	}
	assertDecimalInDelta(t, fin.Weight, decimal.NewFromInt(30), "0.01", "Financials weight")
	if !equalDecimal(got.Covered, decimal.NewFromInt(1000)) {
		t.Fatalf("covered = %v, want 1000", got.Covered)
	}
	if !got.Excluded.IsZero() {
		t.Fatalf("excluded = %v, want zero", got.Excluded)
	}
}
