package service

import (
	"context"
	"testing"

	"github.com/alv67/peculium/internal/model"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// A stock's geography is never editable: it is always derived 100% from the
// asset's country field. Stored country/region rows on a stock (legacy
// manual edits from the geo modal) are ignored by the geographic allocation
// and its drill-down.

func TestStockGeography_AllocationIgnoresStoredRows(t *testing.T) {
	stockID := uuid.New()
	pf := &fakePortfolioRepo{
		portfolio: &model.Portfolio{Currency: "EUR"},
		holdings: []*model.Holding{
			holding(stockID.String(), "EUR", "US", "Technology", model.AssetTypeStock, decimal.NewFromInt(1), decimal.NewFromInt(100)),
		},
	}
	ex := &fakeExposureRepo{
		countries: map[string][]model.ExposureRow{
			stockID.String(): {{Name: "FR", Weight: decimal.NewFromInt(100)}},
		},
		regions: map[string][]model.ExposureRow{
			stockID.String(): {{Name: "Europe Developed", Weight: decimal.NewFromInt(100)}},
		},
	}
	svc := newTestService(t, pf, ex, &fakeFXRepo{})
	portfolioID := uuid.New()

	got, err := svc.GetPortfolioGeographyAllocation(context.Background(), portfolioID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Countries) != 1 || got.Countries[0].Country != "US" || !equalDecimal(got.Countries[0].Value, decimal.NewFromInt(100)) {
		t.Fatalf("countries = %+v, want only US 100", got.Countries)
	}
	if !equalDecimal(regionByName(t, got.Regions, "North America").Value, decimal.NewFromInt(100)) {
		t.Fatalf("North America = %v, want 100", regionByName(t, got.Regions, "North America").Value)
	}
	if !regionByName(t, got.Regions, "Europe Developed").Value.IsZero() {
		t.Fatalf("Europe Developed = %v, want zero", regionByName(t, got.Regions, "Europe Developed").Value)
	}
	// The filtering copies the map: the repo's stored rows are untouched.
	if len(ex.countries[stockID.String()]) != 1 || ex.countries[stockID.String()][0].Name != "FR" {
		t.Fatalf("stored FR row mutated: %+v", ex.countries[stockID.String()])
	}
}

func TestStockGeography_DrillIgnoresStoredRows(t *testing.T) {
	stockID := uuid.New()
	pf := &fakePortfolioRepo{
		portfolio: &model.Portfolio{Currency: "EUR"},
		holdings: []*model.Holding{
			holding(stockID.String(), "EUR", "US", "Technology", model.AssetTypeStock, decimal.NewFromInt(1), decimal.NewFromInt(100)),
		},
	}
	ex := &fakeExposureRepo{
		countries: map[string][]model.ExposureRow{
			stockID.String(): {{Name: "FR", Weight: decimal.NewFromInt(100)}},
		},
		regions: map[string][]model.ExposureRow{
			stockID.String(): {{Name: "Europe Developed", Weight: decimal.NewFromInt(100)}},
		},
	}
	svc := newTestService(t, pf, ex, &fakeFXRepo{})
	portfolioID := uuid.New()

	fr, err := svc.GetPortfolioAllocationDrill(context.Background(), portfolioID, "country", "FR")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fr.Assets == nil || len(fr.Assets) != 0 || !fr.Total.IsZero() {
		t.Fatalf("FR drill = %+v, want non-nil empty assets and zero total", fr)
	}
	us, err := svc.GetPortfolioAllocationDrill(context.Background(), portfolioID, "country", "US")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(us.Assets) != 1 || us.Assets[0].AssetID != stockID.String() || !equalDecimal(us.Assets[0].Weight, decimal.NewFromInt(100)) {
		t.Fatalf("US drill = %+v, want only the stock at weight 100", us.Assets)
	}
	assertDrillContribution(t, us.Assets[0], "US stock")

	eu, err := svc.GetPortfolioAllocationDrill(context.Background(), portfolioID, "region", "Europe Developed")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if eu.Assets == nil || len(eu.Assets) != 0 || !eu.Total.IsZero() {
		t.Fatalf("Europe Developed drill = %+v, want non-nil empty assets and zero total", eu)
	}
	na, err := svc.GetPortfolioAllocationDrill(context.Background(), portfolioID, "region", "North America")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(na.Assets) != 1 || na.Assets[0].AssetID != stockID.String() || !equalDecimal(na.Assets[0].Weight, decimal.NewFromInt(100)) {
		t.Fatalf("North America drill = %+v, want only the stock at weight 100", na.Assets)
	}
	assertDrillContribution(t, na.Assets[0], "North America stock")
}

func TestStockGeography_DashboardIgnoresStoredRows(t *testing.T) {
	stockID := uuid.New()
	pf := &fakePortfolioRepo{
		portfolios: []*model.Portfolio{{Currency: "USD"}},
		holdings: []*model.Holding{
			holding(stockID.String(), "USD", "US", "Technology", model.AssetTypeStock, decimal.NewFromInt(1), decimal.NewFromInt(100)),
		},
	}
	ex := &fakeExposureRepo{
		countries: map[string][]model.ExposureRow{
			stockID.String(): {{Name: "FR", Weight: decimal.NewFromInt(100)}},
		},
		regions: map[string][]model.ExposureRow{
			stockID.String(): {{Name: "Europe Developed", Weight: decimal.NewFromInt(100)}},
		},
	}
	svc := newTestService(t, pf, ex, &fakeFXRepo{})

	got, err := svc.GetDashboardAllocation(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Countries) != 1 || got.Countries[0].Country != "US" || !equalDecimal(got.Countries[0].Value, decimal.NewFromInt(100)) {
		t.Fatalf("countries = %+v, want only US 100", got.Countries)
	}
	if !equalDecimal(regionByName(t, got.Regions, "North America").Value, decimal.NewFromInt(100)) {
		t.Fatalf("North America = %v, want 100", regionByName(t, got.Regions, "North America").Value)
	}
	if !regionByName(t, got.Regions, "Europe Developed").Value.IsZero() {
		t.Fatalf("Europe Developed = %v, want zero", regionByName(t, got.Regions, "Europe Developed").Value)
	}
}

// ETFs keep their stored provider/manual breakdowns: in a map shared with an
// ignored stock row, only the stock entry is dropped.
func TestStockGeography_EtfKeepsStoredRows(t *testing.T) {
	stockID := uuid.New()
	etfID := uuid.New()
	pf := &fakePortfolioRepo{
		portfolio: &model.Portfolio{Currency: "EUR"},
		holdings: []*model.Holding{
			holding(stockID.String(), "EUR", "US", "Technology", model.AssetTypeStock, decimal.NewFromInt(1), decimal.NewFromInt(100)),
			holding(etfID.String(), "EUR", "", "", model.AssetTypeETF, decimal.NewFromInt(1), decimal.NewFromInt(100)),
		},
	}
	ex := &fakeExposureRepo{
		countries: map[string][]model.ExposureRow{
			stockID.String(): {{Name: "FR", Weight: decimal.NewFromInt(100)}},
			etfID.String():   {{Name: "US", Weight: decimal.NewFromInt(50)}, {Name: "JP", Weight: decimal.NewFromInt(50)}},
		},
		regions: map[string][]model.ExposureRow{
			stockID.String(): {{Name: "Europe Developed", Weight: decimal.NewFromInt(100)}},
			etfID.String():   {{Name: "North America", Weight: decimal.NewFromInt(50)}, {Name: "Japan", Weight: decimal.NewFromInt(50)}},
		},
	}
	svc := newTestService(t, pf, ex, &fakeFXRepo{})

	got, err := svc.GetPortfolioGeographyAllocation(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The stock contributes its own US 100; the ETF splits 50/50 over
	// US/JP from its stored rows. FR never appears.
	if len(got.Countries) != 2 || got.Countries[0].Country != "US" || got.Countries[1].Country != "JP" {
		t.Fatalf("countries = %+v, want US then JP", got.Countries)
	}
	if !equalDecimal(got.Countries[0].Value, decimal.NewFromInt(150)) {
		t.Fatalf("US = %v, want 150", got.Countries[0].Value)
	}
	if !equalDecimal(got.Countries[1].Value, decimal.NewFromInt(50)) {
		t.Fatalf("JP = %v, want 50", got.Countries[1].Value)
	}
	if !equalDecimal(regionByName(t, got.Regions, "North America").Value, decimal.NewFromInt(150)) {
		t.Fatalf("North America = %v, want 150", regionByName(t, got.Regions, "North America").Value)
	}
	if !equalDecimal(regionByName(t, got.Regions, "Japan").Value, decimal.NewFromInt(50)) {
		t.Fatalf("Japan = %v, want 50", regionByName(t, got.Regions, "Japan").Value)
	}
	if !regionByName(t, got.Regions, "Europe Developed").Value.IsZero() {
		t.Fatalf("Europe Developed = %v, want zero", regionByName(t, got.Regions, "Europe Developed").Value)
	}
}

// Sectors are unaffected: a stock still allocates from its stored sector
// rows when it has them, and from the normalized sector default otherwise.
func TestStockGeography_StockSectorRowsStillUsed(t *testing.T) {
	stockID := uuid.New()
	plainID := uuid.New()
	pf := &fakePortfolioRepo{
		portfolio: &model.Portfolio{Currency: "EUR"},
		holdings: []*model.Holding{
			holding(stockID.String(), "EUR", "US", "Technology", model.AssetTypeStock, decimal.NewFromInt(1), decimal.NewFromInt(100)),
			holding(plainID.String(), "EUR", "US", "Technology", model.AssetTypeStock, decimal.NewFromInt(1), decimal.NewFromInt(100)),
		},
	}
	ex := &fakeExposureRepo{
		sectors: map[string][]model.ExposureRow{
			stockID.String(): {{Name: "Financials", Weight: decimal.NewFromInt(100)}},
		},
	}
	svc := newTestService(t, pf, ex, &fakeFXRepo{})

	got, err := svc.GetPortfolioSectorAllocation(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !equalDecimal(sectorByName(t, got.Sectors, "Financials").Value, decimal.NewFromInt(100)) {
		t.Fatalf("Financials = %v, want 100 (stored sector rows still used)", sectorByName(t, got.Sectors, "Financials").Value)
	}
	if !equalDecimal(sectorByName(t, got.Sectors, "Information Technology").Value, decimal.NewFromInt(100)) {
		t.Fatalf("Information Technology = %v, want 100 (sector default)", sectorByName(t, got.Sectors, "Information Technology").Value)
	}
}
