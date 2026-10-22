package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/alv67/peculium/internal/cache"
	"github.com/alv67/peculium/internal/model"
	"github.com/alv67/peculium/internal/repository"
)

// Option A1 (asset-level quote divisor): a 10 000 nominal bond quoted at
// 100.25 per 100 is worth 10 025, while an ordinary asset and an explicit
// divisor 1 keep the plain quantity × close maths.

func qdBondHolding(pfID uuid.UUID, assetID, currency string) *model.Holding {
	return &model.Holding{
		PortfolioID:   pfID.String(),
		AssetID:       assetID,
		Ticker:        "BND",
		Name:          "Bond quoted per 100",
		Currency:      currency,
		Type:          model.AssetTypeBond,
		AssetClass:    "bond",
		IssuerCountry: "IT",
		Qty:           dec("10000"),
		LastClose:     dec("100.25"),
		HasPrice:      true,
		Cost:          dec("10025"),
		CostCCY:       dec("10025"),
		QuoteDivisor:  dec("100"),
	}
}

const qdBondValue = "10025" // 10000 × 100.25 / 100

func TestGetPortfolioSummary_QuoteDivisor(t *testing.T) {
	pfID := uuid.New()
	bondID := uuid.New().String()
	equityID := uuid.New().String()
	pf := &fakePortfolioRepo{
		portfolio: &model.Portfolio{ID: pfID, Currency: "EUR"},
		holdings: []*model.Holding{
			qdBondHolding(pfID, bondID, "EUR"),
			holding(equityID, "EUR", "US", "Technology", model.AssetTypeStock, dec("10"), dec("50")),
		},
	}
	svc := newTestService(t, pf, &fakeExposureRepo{}, &fakeFXRepo{})

	got, err := svc.GetPortfolioSummary(context.Background(), pfID)
	if err != nil {
		t.Fatalf("summary: %v", err)
	}
	if !equalDecimal(got.TotalValue, dec("10525")) {
		t.Fatalf("total_value = %v, want 10525 (bond 10025 + equity 500)", got.TotalValue)
	}
	for _, h := range got.Holdings {
		want := qdBondValue
		if h.AssetID == equityID {
			want = "500"
		}
		if !equalDecimal(h.Value, dec(want)) {
			t.Fatalf("holding %s value = %v, want %s", h.Ticker, h.Value, want)
		}
	}
	// Only the bond carries a cost in this fixture (the equity helper leaves
	// it zero), so the active invested is the divided bond cost alone.
	if !equalDecimal(got.Active.Invested, dec(qdBondValue)) {
		t.Fatalf("active.invested = %v, want 10025", got.Active.Invested)
	}
}

func TestGetPortfolioAllocation_QuoteDivisor(t *testing.T) {
	pfID := uuid.New()
	bondID := uuid.New().String()
	pf := &fakePortfolioRepo{
		portfolio: &model.Portfolio{ID: pfID, Currency: "EUR"},
		holdings:  []*model.Holding{qdBondHolding(pfID, bondID, "EUR")},
	}
	svc := newTestService(t, pf, &fakeExposureRepo{}, &fakeFXRepo{})

	t.Run("asset", func(t *testing.T) {
		allocs, err := svc.GetPortfolioAllocation(context.Background(), pfID)
		if err != nil {
			t.Fatal(err)
		}
		if len(allocs) != 1 || !equalDecimal(allocs[0].Value, dec(qdBondValue)) {
			t.Fatalf("allocations = %+v, want a single 10025", allocs)
		}
	})
	t.Run("class", func(t *testing.T) {
		classes, err := svc.GetPortfolioClassAllocation(context.Background(), pfID)
		if err != nil {
			t.Fatal(err)
		}
		if len(classes.Classes) != 1 || !equalDecimal(classes.Classes[0].Value, dec(qdBondValue)) {
			t.Fatalf("classes = %+v, want bond 10025", classes.Classes)
		}
	})
	t.Run("geography", func(t *testing.T) {
		ga, err := svc.GetPortfolioGeographyAllocation(context.Background(), pfID)
		if err != nil {
			t.Fatal(err)
		}
		if len(ga.Countries) != 1 || ga.Countries[0].Country != "IT" || !equalDecimal(ga.Countries[0].Value, dec(qdBondValue)) {
			t.Fatalf("countries = %+v, want IT 10025", ga.Countries)
		}
	})
	t.Run("sector excludes the bond at the divided value", func(t *testing.T) {
		sec, err := svc.GetPortfolioSectorAllocation(context.Background(), pfID)
		if err != nil {
			t.Fatal(err)
		}
		if !equalDecimal(sec.Excluded, dec(qdBondValue)) {
			t.Fatalf("excluded = %v, want 10025", sec.Excluded)
		}
	})
	t.Run("drill", func(t *testing.T) {
		drill, err := svc.GetPortfolioAllocationDrill(context.Background(), pfID, "class", "bond")
		if err != nil {
			t.Fatal(err)
		}
		if len(drill.Assets) != 1 || !equalDecimal(drill.Assets[0].Contribution, dec(qdBondValue)) {
			t.Fatalf("drill = %+v, want a single 10025 contribution", drill.Assets)
		}
	})
}

func TestGetDashboard_QuoteDivisor(t *testing.T) {
	pfID := uuid.New()
	bondID := uuid.New().String()
	pf := &fakePortfolioRepo{
		portfolios: []*model.Portfolio{{ID: pfID, Currency: "USD"}},
		holdings:   []*model.Holding{qdBondHolding(pfID, bondID, "USD")},
	}
	svc := newDashboardTestService(t, pf, &fakeFXRepo{}, "USD", nil)

	dash, err := svc.GetDashboard(context.Background(), uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if !equalDecimal(dash.Summary.Active.Value, dec(qdBondValue)) {
		t.Fatalf("summary active.value = %v, want 10025", dash.Summary.Active.Value)
	}
	if len(dash.InvestedAssets) != 1 || !equalDecimal(dash.InvestedAssets[0].Value, dec(qdBondValue)) {
		t.Fatalf("invested assets = %+v, want one 10025", dash.InvestedAssets)
	}

	allocation, err := svc.GetDashboardAllocation(context.Background(), uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if len(allocation.Classes) != 1 || !equalDecimal(allocation.Classes[0].Value, dec(qdBondValue)) {
		t.Fatalf("dashboard classes = %+v, want bond 10025", allocation.Classes)
	}
}

// TestGetPortfolioPerformanceBuckets_QuoteDivisor checks the TWR cash-flow
// path: the buy flow is the divided 10 025, so invested matches it.
func TestGetPortfolioPerformanceBuckets_QuoteDivisor(t *testing.T) {
	owner := uuid.New()
	pfID := uuid.New()
	bondID := uuid.New().String()
	buy := time.Date(2025, 1, 5, 0, 0, 0, 0, time.UTC)
	d1 := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)

	pf := &fakePortfolioRepo{
		portfolio: perfPortfolio(owner, pfID, "USD"),
		holdings:  []*model.Holding{seriesHolding(pfID, bondID)},
	}
	assets := map[uuid.UUID][]model.AssetPositionSeries{
		pfID: {
			{AssetID: bondID, Series: []model.PositionPoint{
				{Date: d1, MarketValue: dec(qdBondValue), CostBasis: dec(qdBondValue)},
			}},
		},
	}
	txs := &fakeTransactionRepo{txs: []model.TransactionWithAsset{
		{PortfolioID: pfID, AssetID: mustUUID(bondID), Type: model.TxBuy, Date: buy,
			Quantity: dec("10000"), Price: dec("100.25"), QuoteDivisor: dec("100")},
	}}
	svc := newPortfolioPerfTestService(t, pf, &fakeFXRepo{}, assets, txs)

	got, err := svc.GetPortfolioPerformanceBuckets(context.Background(), pfID, owner, "month")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Buckets) != 1 {
		t.Fatalf("buckets = %+v, want one", got.Buckets)
	}
	if !equalDecimal(got.Buckets[0].Invested, dec(qdBondValue)) {
		t.Fatalf("invested = %v, want 10025 (the flow divided by the quote divisor)", got.Buckets[0].Invested)
	}
	if !equalDecimal(got.Buckets[0].Value, dec(qdBondValue)) {
		t.Fatalf("value = %v, want 10025", got.Buckets[0].Value)
	}
}

// TestGetPortfolioSummary_QuoteDivisorOneUnchanged is the byte-for-byte
// regression guard: an explicit 1 divisor (as scanned from the
// NUMERIC(12,4) column default) must produce the very same JSON as today's
// unset behaviour.
func TestGetPortfolioSummary_QuoteDivisorOneUnchanged(t *testing.T) {
	render := func(divisor decimal.Decimal) string {
		pfID := uuid.New()
		h := qdBondHolding(pfID, uuid.New().String(), "EUR")
		// A divisor-1 asset is an ordinary instrument: drop the bond-only
		// figures so both runs differ only in the divisor field.
		h.QuoteDivisor = divisor
		h.Qty = dec("10")
		h.LastClose = dec("50")
		h.Cost = dec("400")
		h.CostCCY = dec("400")
		pf := &fakePortfolioRepo{
			portfolio: &model.Portfolio{ID: pfID, Currency: "EUR"},
			holdings:  []*model.Holding{h},
		}
		svc := newTestService(t, pf, &fakeExposureRepo{}, &fakeFXRepo{})
		got, err := svc.GetPortfolioSummary(context.Background(), pfID)
		if err != nil {
			t.Fatal(err)
		}
		// The portfolio and asset ids differ per run: pin them before
		// comparing.
		got.PortfolioID = ""
		for i := range got.Holdings {
			got.Holdings[i].AssetID = ""
		}
		b, err := json.Marshal(got)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	unset := render(decimal.Zero)
	one := render(dec("1"))
	if unset != one {
		t.Fatalf("divisor 1 changed the summary JSON:\nunset: %s\n1:     %s", unset, one)
	}
}

func TestCreateAsset_QuoteDivisor(t *testing.T) {
	newSvc := func() *Service {
		repos := &repository.Repository{
			Asset: &fakeAssetRepo{}, Portfolio: &fakePortfolioRepo{}, Exposure: &fakeExposureRepo{},
			FX: &fakeFXRepo{}, Job: &fakeJobRepo{},
			User: &fakeUserRepo{user: &model.User{ID: uuid.New()}},
		}
		return New(repos, nil, nil, nil, time.Minute, time.Hour, cache.New(nil), 0, 0, nil, nil)
	}
	// The fake asset repo returns the input pointer, so what the service
	// normalized is exactly what the real repository would have persisted.
	t.Run("omitted defaults to 1", func(t *testing.T) {
		got, err := newSvc().CreateAsset(context.Background(), &model.Asset{Ticker: "B", Type: model.AssetTypeBond, PriceSource: "none"})
		if err != nil {
			t.Fatal(err)
		}
		if !equalDecimal(got.QuoteDivisor, dec("1")) {
			t.Fatalf("quote_divisor = %v, want the default 1", got.QuoteDivisor)
		}
	})
	t.Run("negative rejected", func(t *testing.T) {
		_, err := newSvc().CreateAsset(context.Background(), &model.Asset{Ticker: "B", PriceSource: "none", QuoteDivisor: dec("-5")})
		if !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("err = %v, want ErrInvalidInput", err)
		}
	})
	t.Run("valid kept", func(t *testing.T) {
		got, err := newSvc().CreateAsset(context.Background(), &model.Asset{Ticker: "B", PriceSource: "none", QuoteDivisor: dec("100")})
		if err != nil {
			t.Fatal(err)
		}
		if !equalDecimal(got.QuoteDivisor, dec("100")) {
			t.Fatalf("quote_divisor = %v, want 100", got.QuoteDivisor)
		}
	})
}

func TestUpdateAsset_QuoteDivisor(t *testing.T) {
	fa := &fakeAssetRepo{asset: &model.Asset{ID: uuid.New(), Ticker: "BND", Type: model.AssetTypeBond, PriceSource: "none", QuoteDivisor: dec("1")}}
	svc := newTestServiceWithAsset(t, fa)

	d := dec("100")
	got, err := svc.UpdateAsset(context.Background(), fa.asset.ID, &model.AssetPatch{QuoteDivisor: &d})
	if err != nil {
		t.Fatal(err)
	}
	if !equalDecimal(got.QuoteDivisor, dec("100")) {
		t.Fatalf("quote_divisor = %v, want 100", got.QuoteDivisor)
	}

	for _, bad := range []string{"0", "-5", "0.0000"} {
		v := dec(bad)
		if _, err := svc.UpdateAsset(context.Background(), fa.asset.ID, &model.AssetPatch{QuoteDivisor: &v}); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("quote_divisor %s: err = %v, want ErrInvalidInput", bad, err)
		}
	}

	// An absent field leaves the stored value untouched.
	if _, err := svc.UpdateAsset(context.Background(), fa.asset.ID, &model.AssetPatch{}); err != nil {
		t.Fatal(err)
	}
	if !equalDecimal(fa.asset.QuoteDivisor, dec("100")) {
		t.Fatalf("unset patch changed the divisor to %v, want 100", fa.asset.QuoteDivisor)
	}
}

// TestBackupQuoteDivisorRoundTrip checks that the divisor travels the export
// → restore path and that a document without the field imports the asset
// without it (the real repository then applies the column default 1).
func TestBackupQuoteDivisorRoundTrip(t *testing.T) {
	s := newBackupStore()
	userID := s.seedUser("a@b.co", "A", "EUR")
	pfID := s.seedPortfolio(userID, "P", "EUR", backupDay(2024, 1, 1))
	assetID := s.seedAsset("BND", "Bond", model.AssetTypeBond, "EUR", "", "none", "bond")
	s.assets[assetID].QuoteDivisor = dec("100")
	s.seedTx(pfID, assetID, model.TxBuy, "10000", "100.25", "0", backupDay(2024, 2, 1), "")

	doc, err := buildUserBackup(context.Background(), s.repos(), userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Portfolios[0].Assets) != 1 {
		t.Fatalf("portfolio assets = %+v", doc.Portfolios[0].Assets)
	}
	exported := doc.Portfolios[0].Assets[0]
	if exported.QuoteDivisor == nil || !equalDecimal(*exported.QuoteDivisor, dec("100")) {
		t.Fatalf("exported divisor = %v, want 100", exported.QuoteDivisor)
	}
	if len(doc.Assets) != 1 || doc.Assets[0].QuoteDivisor == nil {
		t.Fatalf("bundle asset divisor = %v, want 100", doc.Assets[0].QuoteDivisor)
	}

	// Restore into an empty store: the created asset keeps the divisor.
	dst := newBackupStore()
	dstUser := dst.seedUser("b@b.co", "B", "EUR")
	if _, err := applyUserBackup(context.Background(), dst.repos(), dstUser, doc, BackupModeAdd, &model.BackupRestoreSummary{}); err != nil {
		t.Fatal(err)
	}
	restored := dst.assetByTicker("BND")
	if restored == nil {
		t.Fatal("BND not restored")
	}
	if !equalDecimal(restored.QuoteDivisor, dec("100")) {
		t.Fatalf("restored divisor = %v, want 100", restored.QuoteDivisor)
	}

	// A document from before the field existed: nil on import leaves the
	// asset at the repository default instead of failing.
	doc.Portfolios[0].Assets[0].QuoteDivisor = nil
	doc.Assets[0].QuoteDivisor = nil
	dst2 := newBackupStore()
	u2 := dst2.seedUser("c@c.co", "C", "EUR")
	if _, err := applyUserBackup(context.Background(), dst2.repos(), u2, doc, BackupModeAdd, &model.BackupRestoreSummary{}); err != nil {
		t.Fatal(err)
	}
	if got := dst2.assetByTicker("BND"); got == nil || !got.QuoteDivisor.IsZero() {
		t.Fatalf("legacy import must leave the divisor unset for the repo default, got %+v", got)
	}
}

// TestToExportAssetOmitsDefaultDivisor pins the additive-optional rule: only
// a non-1 divisor shows up in the JSON document.
func TestToExportAssetOmitsDefaultDivisor(t *testing.T) {
	b, err := json.Marshal(toExportAsset(&model.Asset{Ticker: "A", QuoteDivisor: dec("1")}))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "quote_divisor") {
		t.Fatalf("the default divisor must be omitted, got %s", b)
	}
	b, _ = json.Marshal(toExportAsset(&model.Asset{Ticker: "A", QuoteDivisor: dec("100")}))
	if !strings.Contains(string(b), `"quote_divisor":"100"`) {
		t.Fatalf("want the divisor as a decimal string, got %s", b)
	}
}
