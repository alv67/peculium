package position

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/alv67/peculium/internal/model"
)

func posTx(typ model.TransactionType, qty, price, fees string) model.TransactionWithAsset {
	return model.TransactionWithAsset{
		PortfolioID: uuid.New(),
		AssetID:     uuid.New(),
		Type:        typ,
		Quantity:    decimal.RequireFromString(qty),
		Price:       decimal.RequireFromString(price),
		Fees:        decimal.RequireFromString(fees),
		Date:        time.Now(),
	}
}

func assertDec(t *testing.T, name string, got, want decimal.Decimal) {
	t.Helper()
	if !got.Equal(want) {
		t.Fatalf("%s = %v, want %v", name, got, want)
	}
}

func TestApplySellAccumulatesClosedLotMetrics(t *testing.T) {
	s := &State{}
	Apply(s, posTx(model.TxBuy, "10", "10", "0"))
	Apply(s, posTx(model.TxSell, "4", "12", "2"))

	// 4 of 10 lots closed at an AVCO cost of 10 each.
	assertDec(t, "ClosedCost", s.ClosedCost, decimal.NewFromInt(40))
	assertDec(t, "ClosedCostCCY", s.ClosedCostCCY, decimal.NewFromInt(40))
	// Proceeds are net of fees in portfolio currency, gross in asset currency.
	assertDec(t, "Proceeds", s.Proceeds, decimal.NewFromInt(46))
	assertDec(t, "ProceedsCCY", s.ProceedsCCY, decimal.NewFromInt(48))
	assertDec(t, "Dividends", s.Dividends, decimal.Zero)
	assertDec(t, "DividendsCCY", s.DividendsCCY, decimal.Zero)

	// Open lot and existing realized/cost computations stay unchanged.
	assertDec(t, "Qty", s.Qty, decimal.NewFromInt(6))
	assertDec(t, "Cost", s.Cost, decimal.NewFromInt(60))
	assertDec(t, "Avg", s.Avg, decimal.NewFromInt(10))
	assertDec(t, "Realized", s.Realized, decimal.NewFromInt(6))
	assertDec(t, "RealizedCCY", s.RealizedCCY, decimal.NewFromInt(8))
}

func TestApplyFullyClosedPosition(t *testing.T) {
	s := &State{}
	Apply(s, posTx(model.TxBuy, "10", "10", "5"))
	Apply(s, posTx(model.TxSell, "10", "12", "3"))

	assertDec(t, "Qty", s.Qty, decimal.Zero)
	assertDec(t, "Cost", s.Cost, decimal.Zero)
	// Buy fees stay in the AVCO cost basis: the whole 105 is attributed to closed lots.
	assertDec(t, "ClosedCost", s.ClosedCost, decimal.NewFromInt(105))
	assertDec(t, "Proceeds", s.Proceeds, decimal.NewFromInt(117))
	assertDec(t, "Realized", s.Realized, decimal.NewFromInt(12))
	assertDec(t, "Dividends", s.Dividends, decimal.Zero)
}

func TestApplyDividendsAccumulateSeparately(t *testing.T) {
	s := &State{}
	Apply(s, posTx(model.TxBuy, "10", "10", "0"))
	Apply(s, posTx(model.TxDividend, "10", "1.5", "0"))
	// Lump-sum dividend: no quantity, the price carries the amount.
	Apply(s, posTx(model.TxDividend, "0", "7", "0"))
	Apply(s, posTx(model.TxSell, "10", "12", "0"))

	assertDec(t, "Dividends", s.Dividends, decimal.RequireFromString("22"))
	assertDec(t, "DividendsCCY", s.DividendsCCY, decimal.RequireFromString("22"))
	// Realized still mixes dividends and capital; the closed group keeps them apart.
	assertDec(t, "Realized", s.Realized, decimal.RequireFromString("42"))
	assertDec(t, "ClosedCost", s.ClosedCost, decimal.NewFromInt(100))
	assertDec(t, "Proceeds", s.Proceeds, decimal.NewFromInt(120))
	assertDec(t, "Qty", s.Qty, decimal.Zero)
}

func TestWalkAccumulatesClosedMetricsPerPortfolioAsset(t *testing.T) {
	pf1 := uuid.New()
	pf2 := uuid.New()
	assetA := uuid.New()
	assetB := uuid.New()
	d := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)

	mk := func(pf, asset uuid.UUID, typ model.TransactionType, qty, price string) model.TransactionWithAsset {
		return model.TransactionWithAsset{
			PortfolioID: pf,
			AssetID:     asset,
			Type:        typ,
			Quantity:    decimal.RequireFromString(qty),
			Price:       decimal.RequireFromString(price),
			Date:        d,
			CreatedAt:   d,
		}
	}
	txs := []model.TransactionWithAsset{
		mk(pf1, assetA, model.TxBuy, "10", "10"),
		mk(pf1, assetA, model.TxSell, "10", "12"),
		mk(pf1, assetA, model.TxDividend, "0", "3"),
		mk(pf1, assetB, model.TxBuy, "5", "20"),
		mk(pf2, assetA, model.TxBuy, "4", "10"),
	}
	states := Walk(txs, nil)

	s1a := states[pf1.String()+"|"+assetA.String()]
	assertDec(t, "pf1/assetA ClosedCost", s1a.ClosedCost, decimal.NewFromInt(100))
	assertDec(t, "pf1/assetA Proceeds", s1a.Proceeds, decimal.NewFromInt(120))
	assertDec(t, "pf1/assetA Dividends", s1a.Dividends, decimal.NewFromInt(3))
	assertDec(t, "pf1/assetA Qty", s1a.Qty, decimal.Zero)

	s1b := states[pf1.String()+"|"+assetB.String()]
	assertDec(t, "pf1/assetB ClosedCost", s1b.ClosedCost, decimal.Zero)
	assertDec(t, "pf1/assetB Dividends", s1b.Dividends, decimal.Zero)

	s2a := states[pf2.String()+"|"+assetA.String()]
	assertDec(t, "pf2/assetA Proceeds", s2a.Proceeds, decimal.Zero)
	assertDec(t, "pf2/assetA Cost", s2a.Cost, decimal.NewFromInt(40))
}
