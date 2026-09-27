package series

import (
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/alv67/peculium/internal/model"
)

func TestIndicesLTTBProperties(t *testing.T) {
	vals := make([]float64, 137)
	for i := range vals {
		vals[i] = 1000 + 500*math.Sin(float64(i)/5) + 50*math.Sin(float64(i)*1.7)
	}
	for _, max := range []int{2, 3, 5, 10, 50} {
		idx := IndicesLTTB(vals, max)
		if len(idx) > max {
			t.Fatalf("max=%d: len %d exceeds budget", max, len(idx))
		}
		if len(idx) < 2 || idx[0] != 0 || idx[len(idx)-1] != len(vals)-1 {
			t.Fatalf("max=%d: first/last not preserved: %v", max, idx)
		}
		for i := 1; i < len(idx); i++ {
			if idx[i] <= idx[i-1] {
				t.Fatalf("max=%d: indices not strictly increasing: %v", max, idx)
			}
		}
	}
}

func TestIndicesLTTBIdentity(t *testing.T) {
	vals := []float64{1, 5, 2, 8, 3, 9}
	idx := IndicesLTTB(vals, 10)
	if !reflect.DeepEqual(idx, []int{0, 1, 2, 3, 4, 5}) {
		t.Fatalf("expected all indices, got %v", idx)
	}
}

func TestIndicesLTTBClampsMaxPoints(t *testing.T) {
	idx := IndicesLTTB([]float64{1, 2, 3, 4, 5, 6}, 0)
	if !reflect.DeepEqual(idx, []int{0, 5}) {
		t.Fatalf("expected [0 5], got %v", idx)
	}
	idx = IndicesLTTB([]float64{1, 2, 3}, 1)
	if !reflect.DeepEqual(idx, []int{0, 2}) {
		t.Fatalf("expected [0 2], got %v", idx)
	}
}

func TestEventIndicesStep(t *testing.T) {
	vals := []int{1, 1, 2, 2, 3, 4, 4, 5}
	idx := EventIndices(vals, 10, func(v int) decimal.Decimal { return decimal.NewFromInt(int64(v)) })
	if !reflect.DeepEqual(idx, []int{2, 4, 5, 7}) {
		t.Fatalf("expected [2 4 5 7], got %v", idx)
	}
}

func TestEventIndicesSpacing(t *testing.T) {
	vals := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}
	idx := EventIndices(vals, 4, func(v int) decimal.Decimal { return decimal.NewFromInt(int64(v)) })
	if len(idx) != 4 {
		t.Fatalf("expected 4 indices, got %v", idx)
	}
	if idx[0] != 1 || idx[len(idx)-1] != 10 {
		t.Fatalf("expected first and last event preserved, got %v", idx)
	}
}

func TestMergeDedupesAndSorts(t *testing.T) {
	out := Merge([]int{0, 5, 9}, []int{9, 2, 5}, 10)
	if !reflect.DeepEqual(out, []int{0, 2, 5, 9}) {
		t.Fatalf("expected [0 2 5 9], got %v", out)
	}
}

func TestMergeTruncatesKeepingEvents(t *testing.T) {
	out := Merge([]int{0, 3, 6, 9}, []int{1, 2, 7}, 3)
	if !reflect.DeepEqual(out, []int{0, 1, 9}) {
		t.Fatalf("expected [0 1 9], got %v", out)
	}
}

func TestMergeAlwaysKeepsEndpoints(t *testing.T) {
	out := Merge([]int{2, 4, 6}, []int{3, 5}, 2)
	if !reflect.DeepEqual(out, []int{2, 6}) {
		t.Fatalf("expected [2 6], got %v", out)
	}
}

func assertAscendingAndEndpoints[T any](t *testing.T, in, out []T, maxPoints int, dateOf func(T) time.Time) {
	t.Helper()
	if len(out) > maxPoints {
		t.Fatalf("output len %d exceeds maxPoints %d", len(out), maxPoints)
	}
	if len(out) == 0 {
		t.Fatal("empty output")
	}
	if !dateOf(out[0]).Equal(dateOf(in[0])) || !dateOf(out[len(out)-1]).Equal(dateOf(in[len(in)-1])) {
		t.Fatal("first/last points not preserved")
	}
	for i := 1; i < len(out); i++ {
		if !dateOf(out[i]).After(dateOf(out[i-1])) {
			t.Fatalf("dates not strictly increasing at %d", i)
		}
	}
}

func assertIdentity[T any](t *testing.T, in, out []T) {
	t.Helper()
	if !reflect.DeepEqual(out, in) {
		t.Fatal("expected identity when input is under budget")
	}
}

func perfVals(n int) []model.PortfolioPerformance {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	out := make([]model.PortfolioPerformance, n)
	for i := range out {
		out[i] = model.PortfolioPerformance{
			Date:  start.AddDate(0, 0, i),
			Value: decimal.NewFromFloat(1000 + 500*math.Sin(float64(i)/5) + 50*math.Sin(float64(i)*1.7)),
		}
	}
	return out
}

func posVals(n int) []model.PositionPoint {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	out := make([]model.PositionPoint, n)
	qty := decimal.NewFromInt(10)
	cost := decimal.NewFromInt(1000)
	realized := decimal.Zero
	for i := range out {
		if i > 0 && i%9 == 4 {
			qty = qty.Add(decimal.NewFromInt(2))
			cost = cost.Add(decimal.NewFromInt(250))
		}
		if i > 0 && i%13 == 7 {
			realized = realized.Add(decimal.NewFromInt(60))
		}
		out[i] = model.PositionPoint{
			Date:        start.AddDate(0, 0, i),
			Qty:         qty,
			CostBasis:   cost,
			MarketValue: qty.Mul(decimal.NewFromFloat(95 + 10*math.Sin(float64(i)/4))),
			Realized:    realized,
		}
	}
	return out
}

func TestPortfolioPerformanceDownsample(t *testing.T) {
	in := perfVals(200)
	out := PortfolioPerformance(in, 20)
	assertAscendingAndEndpoints(t, in, out, 20, func(p model.PortfolioPerformance) time.Time { return p.Date })
	if !reflect.DeepEqual(PortfolioPerformance(in, 20), out) {
		t.Fatal("downsampling not deterministic")
	}
}

func TestPortfolioPerformanceIdentity(t *testing.T) {
	in := perfVals(50)
	assertIdentity(t, in, PortfolioPerformance(in, 500))
}

func TestPositionPointsDownsample(t *testing.T) {
	in := posVals(150)
	out := PositionPoints(in, 15)
	assertAscendingAndEndpoints(t, in, out, 15, func(p model.PositionPoint) time.Time { return p.Date })
	if !reflect.DeepEqual(PositionPoints(in, 15), out) {
		t.Fatal("downsampling not deterministic")
	}
}

func TestPositionPointsIdentity(t *testing.T) {
	in := posVals(50)
	assertIdentity(t, in, PositionPoints(in, 500))
}

func stepPositions(n int) []model.PositionPoint {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	qtySteps := map[int]decimal.Decimal{
		5:  decimal.NewFromInt(12),
		12: decimal.NewFromInt(15),
		28: decimal.NewFromInt(20),
	}
	costSteps := map[int]decimal.Decimal{
		8:  decimal.NewFromInt(1250),
		20: decimal.NewFromInt(1600),
	}
	realizedSteps := map[int]decimal.Decimal{
		15: decimal.NewFromInt(80),
		33: decimal.NewFromInt(140),
	}
	out := make([]model.PositionPoint, n)
	qty := decimal.NewFromInt(10)
	cost := decimal.NewFromInt(1000)
	realized := decimal.Zero
	for i := range out {
		if v, ok := qtySteps[i]; ok {
			qty = v
		}
		if v, ok := costSteps[i]; ok {
			cost = v
		}
		if v, ok := realizedSteps[i]; ok {
			realized = v
		}
		out[i] = model.PositionPoint{
			Date:        start.AddDate(0, 0, i),
			Qty:         qty,
			CostBasis:   cost,
			MarketValue: qty.Mul(decimal.NewFromFloat(100 + 30*math.Sin(float64(i)/3))),
			Realized:    realized,
		}
	}
	return out
}

func TestPositionPointsKeepsStepEvents(t *testing.T) {
	in := stepPositions(40)
	out := PositionPoints(in, 10)
	for _, i := range []int{5, 8, 12, 15, 20, 28, 33} {
		found := false
		for _, p := range out {
			if p.Date.Equal(in[i].Date) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("event day at index %d missing from output", i)
		}
	}
}

func priceVals(n int) []model.Price {
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	out := make([]model.Price, n)
	for i := range out {
		open := decimal.NewFromInt(int64(100 + i))
		high := open.Add(decimal.NewFromInt(int64(10 + (i*3)%7)))
		low := open.Sub(decimal.NewFromInt(int64(5 + (i*2)%5)))
		close := open.Add(decimal.NewFromInt(int64((i * 5) % 11)))
		out[i] = model.Price{
			ID:        uuid.New(),
			AssetID:   uuid.New(),
			Date:      start.AddDate(0, 0, i),
			Open:      open,
			High:      high,
			Low:       low,
			Close:     close,
			Volume:    int64(1000 + i*37),
			Source:    "test",
			CreatedAt: start.AddDate(0, 0, i),
		}
	}
	return out
}

func TestPricesDownsample(t *testing.T) {
	in := priceVals(100)
	out := Prices(in, 10)
	if len(out) > 10 {
		t.Fatalf("output len %d exceeds budget", len(out))
	}
	if len(out) == 0 {
		t.Fatal("empty output")
	}
	if !out[len(out)-1].Date.Equal(in[len(in)-1].Date) {
		t.Fatal("last date not preserved")
	}

	bs := (len(in) + 9) / 10
	var ranges [][2]int
	for start := 0; start < len(in); start += bs {
		end := start + bs
		if end > len(in) {
			end = len(in)
		}
		ranges = append(ranges, [2]int{start, end})
	}
	if len(ranges) != len(out) {
		t.Fatalf("expected %d buckets, got %d", len(ranges), len(out))
	}

	for b, r := range ranges {
		wantHigh := in[r[0]].High
		wantLow := in[r[0]].Low
		var wantVol int64
		for i := r[0]; i < r[1]; i++ {
			if in[i].High.GreaterThan(wantHigh) {
				wantHigh = in[i].High
			}
			if in[i].Low.LessThan(wantLow) {
				wantLow = in[i].Low
			}
			wantVol += in[i].Volume
		}
		got := out[b]
		if got.Open.Cmp(in[r[0]].Open) != 0 {
			t.Fatalf("bucket %d: open mismatch", b)
		}
		if got.Close.Cmp(in[r[1]-1].Close) != 0 {
			t.Fatalf("bucket %d: close mismatch", b)
		}
		if got.High.Cmp(wantHigh) != 0 {
			t.Fatalf("bucket %d: high mismatch", b)
		}
		if got.Low.Cmp(wantLow) != 0 {
			t.Fatalf("bucket %d: low mismatch", b)
		}
		if got.Volume != wantVol {
			t.Fatalf("bucket %d: volume mismatch", b)
		}
		if !got.Date.Equal(in[r[1]-1].Date) {
			t.Fatalf("bucket %d: date must be the last of the bucket", b)
		}
		if got.High.LessThan(got.Open) || got.High.LessThan(got.Close) {
			t.Fatalf("bucket %d: high below open/close", b)
		}
		if got.Low.GreaterThan(got.Open) || got.Low.GreaterThan(got.Close) {
			t.Fatalf("bucket %d: low above open/close", b)
		}
	}

	if !reflect.DeepEqual(Prices(in, 10), out) {
		t.Fatal("downsampling not deterministic")
	}
}

func TestPricesIdentity(t *testing.T) {
	in := priceVals(20)
	assertIdentity(t, in, Prices(in, 500))
}
