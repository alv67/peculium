package service

import (
	"context"
	"testing"

	"github.com/alv67/peculium/internal/geo"
	"github.com/alv67/peculium/internal/model"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// A saved-but-empty exposure table (every weight zero) must not mask the
// single-name country inference: the caller's default fallback applies exactly
// as if no rows were stored.

func TestGeographyAllocation_ZeroWeightRowsInferStockCountry(t *testing.T) {
	stockID := uuid.New()
	pf := &fakePortfolioRepo{
		portfolio: &model.Portfolio{Currency: "EUR"},
		holdings: []*model.Holding{
			holding(stockID.String(), "EUR", "US", "Technology", model.AssetTypeStock, decimal.NewFromInt(1), decimal.NewFromInt(100)),
		},
	}
	ex := &fakeExposureRepo{
		countries: map[string][]model.ExposureRow{
			stockID.String(): {{Name: "US", Weight: decimal.Zero}, {Name: "DE", Weight: decimal.Zero}},
		},
		regions: map[string][]model.ExposureRow{
			stockID.String(): {{Name: "North America", Weight: decimal.Zero}, {Name: "Europe Developed", Weight: decimal.Zero}},
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
	if len(got.Regions) != len(geo.Regions) {
		t.Fatalf("regions len = %d, want %d (no Other)", len(got.Regions), len(geo.Regions))
	}
	if !equalDecimal(regionByName(t, got.Regions, "North America").Value, decimal.NewFromInt(100)) {
		t.Fatalf("North America = %v, want 100", regionByName(t, got.Regions, "North America").Value)
	}

	// The drill shares canonicalExposureRows: the same US bucket finds the
	// stock at 100% weight.
	drill, err := svc.GetPortfolioAllocationDrill(context.Background(), portfolioID, "country", "US")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(drill.Assets) != 1 || drill.Assets[0].AssetID != stockID.String() || !equalDecimal(drill.Assets[0].Weight, decimal.NewFromInt(100)) {
		t.Fatalf("US drill = %+v, want only the stock at weight 100", drill.Assets)
	}
}

func TestGeographyAllocation_ZeroWeightRowsInferBondIssuer(t *testing.T) {
	bondID := uuid.New()
	pf := &fakePortfolioRepo{
		portfolio: &model.Portfolio{Currency: "EUR"},
		holdings: []*model.Holding{
			issuerHolding(bondID.String(), "EUR", model.AssetTypeBond, "bond", "DE", 5, 100),
		},
	}
	ex := &fakeExposureRepo{
		countries: map[string][]model.ExposureRow{
			bondID.String(): {{Name: "DE", Weight: decimal.Zero}, {Name: "FR", Weight: decimal.Zero}},
		},
		regions: map[string][]model.ExposureRow{
			bondID.String(): {{Name: "Europe Developed", Weight: decimal.Zero}},
		},
	}
	svc := newTestService(t, pf, ex, &fakeFXRepo{})

	got, err := svc.GetPortfolioGeographyAllocation(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Countries) != 1 || got.Countries[0].Country != "DE" || !equalDecimal(got.Countries[0].Value, decimal.NewFromInt(500)) {
		t.Fatalf("countries = %+v, want only DE 500", got.Countries)
	}
	if len(got.Regions) != len(geo.Regions) {
		t.Fatalf("regions len = %d, want %d (no Other)", len(got.Regions), len(geo.Regions))
	}
	if !equalDecimal(regionByName(t, got.Regions, "Europe Developed").Value, decimal.NewFromInt(500)) {
		t.Fatalf("Europe Developed = %v, want 500", regionByName(t, got.Regions, "Europe Developed").Value)
	}
}

// An ETF (equity class) never takes the single-name default: zero-weight
// stored rows keep landing the whole value in "Other", as before.
func TestGeographyAllocation_EtfZeroWeightRowsStayOther(t *testing.T) {
	etfID := uuid.New()
	pf := &fakePortfolioRepo{
		portfolio: &model.Portfolio{Currency: "EUR"},
		holdings: []*model.Holding{
			holding(etfID.String(), "EUR", "US", "", model.AssetTypeETF, decimal.NewFromInt(1), decimal.NewFromInt(100)),
		},
	}
	ex := &fakeExposureRepo{
		countries: map[string][]model.ExposureRow{
			etfID.String(): {{Name: "US", Weight: decimal.Zero}},
		},
		regions: map[string][]model.ExposureRow{
			etfID.String(): {{Name: "North America", Weight: decimal.Zero}},
		},
	}
	svc := newTestService(t, pf, ex, &fakeFXRepo{})

	got, err := svc.GetPortfolioGeographyAllocation(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Countries) != 1 || got.Countries[0].Country != "Other" || !equalDecimal(got.Countries[0].Value, decimal.NewFromInt(100)) {
		t.Fatalf("countries = %+v, want only Other 100", got.Countries)
	}
	if !equalDecimal(regionByName(t, got.Regions, "Other").Value, decimal.NewFromInt(100)) {
		t.Fatalf("Other region = %v, want 100", regionByName(t, got.Regions, "Other").Value)
	}
	if !equalDecimal(regionByName(t, got.Regions, "North America").Value, decimal.Zero) {
		t.Fatalf("North America = %v, want zero", regionByName(t, got.Regions, "North America").Value)
	}
}

// A real stored breakdown is never overwritten by the default.
func TestGeographyAllocation_StockKeepsStoredBreakdown(t *testing.T) {
	stockID := uuid.New()
	pf := &fakePortfolioRepo{
		portfolio: &model.Portfolio{Currency: "EUR"},
		holdings: []*model.Holding{
			holding(stockID.String(), "EUR", "US", "Technology", model.AssetTypeStock, decimal.NewFromInt(1), decimal.NewFromInt(100)),
		},
	}
	ex := &fakeExposureRepo{
		countries: map[string][]model.ExposureRow{
			stockID.String(): {{Name: "US", Weight: decimal.NewFromInt(70)}, {Name: "DE", Weight: decimal.NewFromInt(30)}},
		},
		regions: map[string][]model.ExposureRow{
			stockID.String(): {{Name: "North America", Weight: decimal.NewFromInt(70)}, {Name: "Europe Developed", Weight: decimal.NewFromInt(30)}},
		},
	}
	svc := newTestService(t, pf, ex, &fakeFXRepo{})

	got, err := svc.GetPortfolioGeographyAllocation(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !equalDecimal(countryByName(t, got.Countries, "US").Value, decimal.NewFromInt(70)) {
		t.Fatalf("US = %v, want 70", countryByName(t, got.Countries, "US").Value)
	}
	if !equalDecimal(countryByName(t, got.Countries, "DE").Value, decimal.NewFromInt(30)) {
		t.Fatalf("DE = %v, want 30", countryByName(t, got.Countries, "DE").Value)
	}
	if !equalDecimal(regionByName(t, got.Regions, "North America").Value, decimal.NewFromInt(70)) {
		t.Fatalf("North America = %v, want 70", regionByName(t, got.Regions, "North America").Value)
	}
}
