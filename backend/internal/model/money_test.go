package model

import (
	"testing"

	"github.com/shopspring/decimal"
)

func dec(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func TestTradeAmount(t *testing.T) {
	qty, price := dec("10000"), dec("100.25")
	got := TradeAmount(qty, price, dec("100"))
	if !got.Equal(dec("10025")) {
		t.Fatalf("10000×100.25/100 = %v, want 10025", got)
	}
	// Divisor 1 and the unset zero value must keep the product byte-for-byte,
	// exactly like the pre-divisor code did.
	for _, d := range []decimal.Decimal{dec("1"), decimal.Zero, dec("0.0000")} {
		if s := TradeAmount(qty, price, d).String(); s != "1002500" {
			t.Fatalf("divisor %v: got %s, want the plain product 1002500", d, s)
		}
	}
}

func TestHoldingValue(t *testing.T) {
	h := &Holding{Qty: dec("10000"), LastClose: dec("100.25"), QuoteDivisor: dec("100")}
	if v := h.Value(); !v.Equal(dec("10025")) {
		t.Fatalf("bond value = %v, want 10025", v)
	}
	plain := &Holding{Qty: dec("10"), LastClose: dec("50")}
	if v := plain.Value(); v.String() != "500" {
		t.Fatalf("unset-divisor value = %v, want the plain qty×close 500", v)
	}
}
