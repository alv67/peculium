package series

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/alv67/peculium/internal/model"
)

func fxDay(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func testDateRates() *dateRates {
	return &dateRates{
		base:   "USD",
		latest: map[string]decimal.Decimal{"EUR": decimal.RequireFromString("1.05")},
		history: map[string][]model.FXRatePoint{
			"EUR": {
				{Date: fxDay(2024, 1, 15), Rate: decimal.RequireFromString("0.90")},
				{Date: fxDay(2024, 3, 15), Rate: decimal.RequireFromString("1.10")},
			},
		},
	}
}

func TestDateRatesRateExactDate(t *testing.T) {
	got, ok := testDateRates().rate("EUR", fxDay(2024, 1, 15))
	if !ok {
		t.Fatal("rate missing for exact history date")
	}
	if !got.Equal(decimal.RequireFromString("0.90")) {
		t.Fatalf("rate = %v, want 0.90", got)
	}
}

func TestDateRatesRateClosestEarlier(t *testing.T) {
	got, ok := testDateRates().rate("EUR", fxDay(2024, 3, 14))
	if !ok {
		t.Fatal("rate missing for closest-earlier lookup")
	}
	if !got.Equal(decimal.RequireFromString("0.90")) {
		t.Fatalf("rate = %v, want 0.90 (closest earlier than 2024-03-15)", got)
	}
}

func TestDateRatesRateLatestAfterLastHistoryPoint(t *testing.T) {
	got, ok := testDateRates().rate("EUR", fxDay(2024, 6, 1))
	if !ok {
		t.Fatal("rate missing after last history point")
	}
	if !got.Equal(decimal.RequireFromString("1.10")) {
		t.Fatalf("rate = %v, want 1.10 (last history point)", got)
	}
}

func TestDateRatesRateFallbackSnapshotWhenHistoryEmpty(t *testing.T) {
	dr := &dateRates{
		base:   "USD",
		latest: map[string]decimal.Decimal{"EUR": decimal.RequireFromString("1.05")},
	}
	got, ok := dr.rate("EUR", fxDay(2024, 6, 1))
	if !ok {
		t.Fatal("rate missing when history is empty")
	}
	if !got.Equal(decimal.RequireFromString("1.05")) {
		t.Fatalf("rate = %v, want snapshot 1.05", got)
	}
}

func TestDateRatesRateFallbackSnapshotBeforeFirstHistoryPoint(t *testing.T) {
	got, ok := testDateRates().rate("EUR", fxDay(2024, 1, 1))
	if !ok {
		t.Fatal("rate missing when target predates the first history point")
	}
	if !got.Equal(decimal.RequireFromString("1.05")) {
		t.Fatalf("rate = %v, want snapshot 1.05", got)
	}
}

func TestDateRatesRateBaseIsOne(t *testing.T) {
	got, ok := testDateRates().rate("USD", fxDay(2024, 6, 1))
	if !ok {
		t.Fatal("base rate missing")
	}
	if !got.Equal(decimal.NewFromInt(1)) {
		t.Fatalf("base rate = %v, want 1", got)
	}
}

func TestDateRatesRateMissingQuote(t *testing.T) {
	if _, ok := testDateRates().rate("JPY", fxDay(2024, 6, 1)); ok {
		t.Fatal("rate resolved for quote with neither history nor snapshot")
	}
}

func TestDateRatesFactorSameCurrency(t *testing.T) {
	got, ok := testDateRates().Factor("EUR", "EUR", fxDay(2024, 6, 1))
	if !ok {
		t.Fatal("same-currency factor missing")
	}
	if !got.Equal(decimal.NewFromInt(1)) {
		t.Fatalf("factor = %v, want 1", got)
	}
}

func TestDateRatesFactorBoundaryMarketValue(t *testing.T) {
	dr := testDateRates()
	qty := decimal.NewFromInt(10)
	price := decimal.NewFromInt(100)

	fBefore, ok := dr.Factor("EUR", "USD", fxDay(2024, 3, 14))
	if !ok {
		t.Fatal("factor before boundary missing")
	}
	if !fBefore.Equal(decimal.NewFromInt(1).Div(decimal.RequireFromString("0.90"))) {
		t.Fatalf("factor before boundary = %v, want 1/0.90", fBefore)
	}
	mvBefore := qty.Mul(price).Mul(fBefore)

	fOn, ok := dr.Factor("EUR", "USD", fxDay(2024, 3, 15))
	if !ok {
		t.Fatal("factor on boundary missing")
	}
	if !fOn.Equal(decimal.NewFromInt(1).Div(decimal.RequireFromString("1.10"))) {
		t.Fatalf("factor on boundary = %v, want 1/1.10", fOn)
	}
	mvOn := qty.Mul(price).Mul(fOn)

	// 1.10 USD per EUR on/after the boundary vs 0.90 before: the USD market
	// value of the EUR holding must drop at the boundary.
	if !mvBefore.Round(2).Equal(decimal.RequireFromString("1111.11")) {
		t.Fatalf("market value before boundary = %v, want 1111.11", mvBefore.Round(2))
	}
	if !mvOn.Round(2).Equal(decimal.RequireFromString("909.09")) {
		t.Fatalf("market value on boundary = %v, want 909.09", mvOn.Round(2))
	}
	if !mvOn.LessThan(mvBefore) {
		t.Fatalf("market value on boundary %v not below %v", mvOn, mvBefore)
	}
}

func TestDateRatesFactorMissingQuote(t *testing.T) {
	if _, ok := testDateRates().Factor("JPY", "USD", fxDay(2024, 6, 1)); ok {
		t.Fatal("factor resolved with a missing quote")
	}
}
