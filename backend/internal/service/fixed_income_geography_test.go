package service

import (
	"context"
	"testing"

	"github.com/alv67/peculium/internal/geo"
	"github.com/alv67/peculium/internal/model"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// issuerHolding builds a holding whose geography comes from its issuer
// country: a bond, certificate or deposit (EPIC J.4). An empty issuer opts
// the asset out of the geography universe.
func issuerHolding(id, currency string, typ model.AssetType, class, issuer string, qty, lastClose int64) *model.Holding {
	h := holding(id, currency, "", "", typ, decimal.NewFromInt(qty), decimal.NewFromInt(lastClose))
	h.IssuerCountry = issuer
	h.AssetClass = class
	return h
}

func TestGetPortfolioGeographyAllocation_FixedIncomeContribute(t *testing.T) {
	stockID := uuid.New()
	bondID := uuid.New()
	certID := uuid.New()
	cashID := uuid.New()
	cryptoID := uuid.New()
	pf := &fakePortfolioRepo{
		portfolio: &model.Portfolio{Currency: "EUR"},
		holdings: []*model.Holding{
			holding(stockID.String(), "EUR", "US", "Technology", model.AssetTypeStock, decimal.NewFromInt(1), decimal.NewFromInt(100)),
			issuerHolding(bondID.String(), "EUR", model.AssetTypeBond, "bond", "DE", 5, 100),
			issuerHolding(certID.String(), "EUR", model.AssetTypeCertificate, "other", "JP", 2, 100),
			issuerHolding(cashID.String(), "EUR", model.AssetTypeCash, "currency", "CH", 3, 100),
			issuerHolding(cryptoID.String(), "EUR", model.AssetTypeCrypto, "crypto", "", 1, 100),
		},
	}
	svc := newTestService(t, pf, &fakeExposureRepo{}, &fakeFXRepo{})

	got, err := svc.GetPortfolioGeographyAllocation(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The bond (500), certificate (200) and cash (300) join the stock (100)
	// in the geography universe: covered 1100; only the crypto (100) stays
	// excluded.
	if !equalDecimal(got.Covered, decimal.NewFromInt(1100)) {
		t.Fatalf("covered = %v, want 1100", got.Covered)
	}
	if !equalDecimal(got.Excluded, decimal.NewFromInt(100)) {
		t.Fatalf("excluded = %v, want 100", got.Excluded)
	}
	// Without stored rows the issuer country is the 100% fallback, mapped to
	// its macro-region: DE+CH → Europe Developed 800, JP → Japan 200, the
	// stock stays North America 100. Fully mapped, so no Other bucket.
	if len(got.Regions) != len(geo.Regions) {
		t.Fatalf("regions len = %d, want %d", len(got.Regions), len(geo.Regions))
	}
	if !equalDecimal(regionByName(t, got.Regions, "Europe Developed").Value, decimal.NewFromInt(800)) {
		t.Fatalf("Europe Developed = %v, want 800", regionByName(t, got.Regions, "Europe Developed").Value)
	}
	if !equalDecimal(regionByName(t, got.Regions, "Japan").Value, decimal.NewFromInt(200)) {
		t.Fatalf("Japan = %v, want 200", regionByName(t, got.Regions, "Japan").Value)
	}
	if !equalDecimal(regionByName(t, got.Regions, "North America").Value, decimal.NewFromInt(100)) {
		t.Fatalf("North America = %v, want 100", regionByName(t, got.Regions, "North America").Value)
	}
	// Countries sorted by value: DE 500, CH 300, JP 200, US 100.
	want := []struct {
		name  string
		value int64
	}{{"DE", 500}, {"CH", 300}, {"JP", 200}, {"US", 100}}
	if len(got.Countries) != len(want) {
		t.Fatalf("countries = %+v, want %d entries", got.Countries, len(want))
	}
	for i, w := range want {
		if got.Countries[i].Country != w.name || !equalDecimal(got.Countries[i].Value, decimal.NewFromInt(w.value)) {
			t.Fatalf("countries[%d] = %+v, want %s %d", i, got.Countries[i], w.name, w.value)
		}
	}
}

func TestGetPortfolioGeographyAllocation_IssuerCountryOptOut(t *testing.T) {
	legacyID := uuid.New()
	staleID := uuid.New()
	certID := uuid.New()
	legacy := holding(legacyID.String(), "EUR", "DE", "", model.AssetTypeBond, decimal.NewFromInt(1), decimal.NewFromInt(100))
	legacy.AssetClass = "bond"
	stale := issuerHolding(staleID.String(), "EUR", model.AssetTypeBond, "bond", "", 1, 100)
	pf := &fakePortfolioRepo{
		portfolio: &model.Portfolio{Currency: "EUR"},
		holdings: []*model.Holding{
			legacy,
			stale,
			issuerHolding(certID.String(), "EUR", model.AssetTypeCertificate, "other", "US", 1, 100),
		},
	}
	// The stale bond keeps a derived country row the asset write did not
	// (re)seed: clearing the issuer must opt it out even with stored rows.
	ex := &fakeExposureRepo{
		countries: map[string][]model.ExposureRow{
			staleID.String(): {{Name: "FR", Weight: decimal.NewFromInt(100)}},
		},
	}
	svc := newTestService(t, pf, ex, &fakeFXRepo{})

	got, err := svc.GetPortfolioGeographyAllocation(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Only the certificate (100) contributes; the legacy country field and
	// the stale stored rows do not: both bonds (200) land in Excluded.
	if !equalDecimal(got.Covered, decimal.NewFromInt(100)) {
		t.Fatalf("covered = %v, want 100", got.Covered)
	}
	if !equalDecimal(got.Excluded, decimal.NewFromInt(200)) {
		t.Fatalf("excluded = %v, want 200", got.Excluded)
	}
	if len(got.Countries) != 1 || got.Countries[0].Country != "US" {
		t.Fatalf("countries = %+v, want only US", got.Countries)
	}
	if len(got.Regions) != len(geo.Regions) {
		t.Fatalf("regions len = %d, want %d (no Other)", len(got.Regions), len(geo.Regions))
	}
	if !equalDecimal(regionByName(t, got.Regions, "North America").Value, decimal.NewFromInt(100)) {
		t.Fatalf("North America = %v, want 100", regionByName(t, got.Regions, "North America").Value)
	}
}

func TestPortfolioSectorAllocationKeepsFixedIncomeOut(t *testing.T) {
	stockID := uuid.New()
	bondID := uuid.New()
	certID := uuid.New()
	stock := holding(stockID.String(), "EUR", "US", "Technology", model.AssetTypeStock, decimal.NewFromInt(1), decimal.NewFromInt(100))
	bond := holding(bondID.String(), "EUR", "DE", "Technology", model.AssetTypeBond, decimal.NewFromInt(5), decimal.NewFromInt(100))
	bond.IssuerCountry = "DE"
	cert := issuerHolding(certID.String(), "EUR", model.AssetTypeCertificate, "other", "US", 2, 100)
	pf := &fakePortfolioRepo{
		portfolio: &model.Portfolio{Currency: "EUR"},
		holdings:  []*model.Holding{stock, bond, cert},
	}
	svc := newTestService(t, pf, &fakeExposureRepo{}, &fakeFXRepo{})
	portfolioID := uuid.New()

	// Geography now covers the fixed income (800 covered, nothing excluded)…
	got, err := svc.GetPortfolioGeographyAllocation(context.Background(), portfolioID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !equalDecimal(got.Covered, decimal.NewFromInt(800)) || !got.Excluded.IsZero() {
		t.Fatalf("geography coverage = (%v, %v), want (800, 0)", got.Covered, got.Excluded)
	}
	// …while sectors stay equity-only: only the stock (100) is covered, the
	// bond (500) and certificate (200) are excluded even with a sector set.
	sec, err := svc.GetPortfolioSectorAllocation(context.Background(), portfolioID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !equalDecimal(sec.Covered, decimal.NewFromInt(100)) {
		t.Fatalf("sector covered = %v, want 100", sec.Covered)
	}
	if !equalDecimal(sec.Excluded, decimal.NewFromInt(700)) {
		t.Fatalf("sector excluded = %v, want 700", sec.Excluded)
	}
	if !equalDecimal(sectorByName(t, sec.Sectors, "Information Technology").Value, decimal.NewFromInt(100)) {
		t.Fatalf("Information Technology = %v, want 100", sectorByName(t, sec.Sectors, "Information Technology").Value)
	}
	if !sectorByName(t, sec.Sectors, "Financials").Value.IsZero() {
		t.Fatalf("Financials = %v, want zero (certificate excluded)", sectorByName(t, sec.Sectors, "Financials").Value)
	}
}

func TestAllocationDrill_FixedIncomeGeography(t *testing.T) {
	bondID := uuid.New()
	cashID := uuid.New()
	stockID := uuid.New()
	noIssuerID := uuid.New()
	bond := issuerHolding(bondID.String(), "EUR", model.AssetTypeBond, "bond", "DE", 5, 100)
	bond.Sector = "Technology"
	noIssuer := holding(noIssuerID.String(), "EUR", "IT", "", model.AssetTypeBond, decimal.NewFromInt(2), decimal.NewFromInt(100))
	pf := &fakePortfolioRepo{
		portfolio: &model.Portfolio{Currency: "EUR"},
		holdings: []*model.Holding{
			bond,
			issuerHolding(cashID.String(), "EUR", model.AssetTypeCash, "currency", "CH", 3, 100),
			holding(stockID.String(), "EUR", "US", "Technology", model.AssetTypeStock, decimal.NewFromInt(1), decimal.NewFromInt(100)),
			noIssuer,
		},
	}
	svc := newTestService(t, pf, &fakeExposureRepo{}, &fakeFXRepo{})
	portfolioID := uuid.New()

	// The issuer fallback maps into the drill too: DE drills down to the
	// bond at 100%, with the same total as the geography bucket.
	de, err := svc.GetPortfolioAllocationDrill(context.Background(), portfolioID, "country", "DE")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(de.Assets) != 1 || de.Assets[0].AssetID != bondID.String() {
		t.Fatalf("DE drill = %+v, want only the bond", de.Assets)
	}
	if !equalDecimal(de.Assets[0].Weight, decimal.NewFromInt(100)) || !equalDecimal(de.Assets[0].Contribution, decimal.NewFromInt(500)) {
		t.Fatalf("bond entry = %+v, want weight 100 contribution 500", de.Assets[0])
	}
	assertDrillContribution(t, de.Assets[0], "DE bond")
	geoAlloc, err := svc.GetPortfolioGeographyAllocation(context.Background(), portfolioID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !equalDecimal(de.Total, countryByName(t, geoAlloc.Countries, "DE").Value) {
		t.Fatalf("DE drill total = %v, want %v", de.Total, countryByName(t, geoAlloc.Countries, "DE").Value)
	}

	// Region drill: Europe Developed aggregates the bond (500) and the cash
	// (300), sorted by contribution.
	eu, err := svc.GetPortfolioAllocationDrill(context.Background(), portfolioID, "region", "Europe Developed")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(eu.Assets) != 2 || eu.Assets[0].AssetID != bondID.String() || eu.Assets[1].AssetID != cashID.String() {
		t.Fatalf("Europe Developed drill = %+v, want bond then cash", eu.Assets)
	}
	for _, a := range eu.Assets {
		assertDrillContribution(t, a, a.AssetID)
	}
	if !equalDecimal(eu.Total, decimal.NewFromInt(800)) {
		t.Fatalf("Europe Developed total = %v, want 800", eu.Total)
	}

	// Sector drill keeps the fixed income out even with a sector and an
	// issuer country: only the stock remains.
	it, err := svc.GetPortfolioAllocationDrill(context.Background(), portfolioID, "sector", "Information Technology")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(it.Assets) != 1 || it.Assets[0].AssetID != stockID.String() {
		t.Fatalf("IT drill = %+v, want only the stock", it.Assets)
	}

	// The opted-out bond (country IT, no issuer) leaves no drill behind.
	empty, err := svc.GetPortfolioAllocationDrill(context.Background(), portfolioID, "country", "IT")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if empty.Assets == nil || len(empty.Assets) != 0 || !empty.Total.IsZero() {
		t.Fatalf("IT drill = %+v, want non-nil empty assets and zero total", empty)
	}
}

func TestGetDashboardAllocation_FixedIncomeGeography(t *testing.T) {
	bondID := uuid.New()
	certID := uuid.New()
	stockID := uuid.New()
	cryptoID := uuid.New()
	pf := &fakePortfolioRepo{
		portfolios: []*model.Portfolio{{Currency: "USD"}},
		holdings: []*model.Holding{
			holding(stockID.String(), "USD", "US", "Technology", model.AssetTypeStock, decimal.NewFromInt(1), decimal.NewFromInt(100)),
			issuerHolding(bondID.String(), "USD", model.AssetTypeBond, "bond", "DE", 4, 100),
			issuerHolding(certID.String(), "USD", model.AssetTypeCertificate, "other", "JP", 1, 100),
			issuerHolding(cryptoID.String(), "USD", model.AssetTypeCrypto, "crypto", "", 1, 100),
		},
	}
	svc := newTestService(t, pf, &fakeExposureRepo{}, &fakeFXRepo{})
	userID := uuid.New()

	got, err := svc.GetDashboardAllocation(context.Background(), userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The vault-wide coverage counts the fixed income (bond 400 + certificate
	// 100) next to the stock (100); only the crypto (100) stays excluded.
	if !equalDecimal(got.Covered, decimal.NewFromInt(600)) {
		t.Fatalf("covered = %v, want 600", got.Covered)
	}
	if !equalDecimal(got.Excluded, decimal.NewFromInt(100)) {
		t.Fatalf("excluded = %v, want 100", got.Excluded)
	}
	if !equalDecimal(countryByName(t, got.Countries, "DE").Value, decimal.NewFromInt(400)) {
		t.Fatalf("DE = %v, want 400", countryByName(t, got.Countries, "DE").Value)
	}
	if !equalDecimal(regionByName(t, got.Regions, "Europe Developed").Value, decimal.NewFromInt(400)) {
		t.Fatalf("Europe Developed = %v, want 400", regionByName(t, got.Regions, "Europe Developed").Value)
	}
	// The vault-wide region drill reaches the same bucket through the
	// dimension map: the bond alone contributes 400.
	drill, err := svc.GetDashboardAllocationDrill(context.Background(), userID, "region", "Europe Developed")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(drill.Assets) != 1 || drill.Assets[0].AssetID != bondID.String() {
		t.Fatalf("Europe Developed drill = %+v, want only the bond", drill.Assets)
	}
	assertDrillContribution(t, drill.Assets[0], "DE bond")
	if !equalDecimal(drill.Total, regionByName(t, got.Regions, "Europe Developed").Value) {
		t.Fatalf("drill total = %v, want %v", drill.Total, regionByName(t, got.Regions, "Europe Developed").Value)
	}
}

func TestGetPortfolioSummary_MissingCountryFollowsGeography(t *testing.T) {
	pfID := uuid.New()
	issuerOnly := issuerHolding(uuid.New().String(), "USD", model.AssetTypeBond, "bond", "CH", 1, 100)
	legacy := holding(uuid.New().String(), "USD", "DE", "", model.AssetTypeBond, decimal.NewFromInt(1), decimal.NewFromInt(100))
	noCountry := holding(uuid.New().String(), "USD", "", "Technology", model.AssetTypeStock, decimal.NewFromInt(1), decimal.NewFromInt(100))
	us := holding(uuid.New().String(), "USD", "US", "Technology", model.AssetTypeStock, decimal.NewFromInt(1), decimal.NewFromInt(100))
	pf := &fakePortfolioRepo{
		portfolio: &model.Portfolio{ID: pfID, Name: "Main", Currency: "USD"},
		holdings:  []*model.Holding{issuerOnly, legacy, noCountry, us},
	}
	svc := newTestService(t, pf, &fakeExposureRepo{}, &fakeFXRepo{})

	got, err := svc.GetPortfolioSummary(context.Background(), pfID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Fixed income is missing geography when the ISSUER country is empty
	// (the bond with only the legacy country field counts), while an
	// issuer-only bond is not. Stocks keep the domicile rule.
	if got.MissingCountry != 2 {
		t.Fatalf("missing_country = %d, want 2", got.MissingCountry)
	}
}
