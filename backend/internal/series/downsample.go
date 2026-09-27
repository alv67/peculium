package series

import (
	"math"
	"sort"

	"github.com/shopspring/decimal"

	"github.com/alv67/peculium/internal/model"
)

// IndicesLTTB selects up to maxPoints indices preserving the visual shape via
// the Largest-Triangle-Three-Buckets heuristic. Input values must be in
// ascending order and the returned indices keep that order. It runs in O(n).
// maxPoints < 2 is clamped to 2; tie-breaks keep the first match.
func IndicesLTTB(values []float64, maxPoints int) []int {
	if maxPoints < 2 {
		maxPoints = 2
	}
	n := len(values)
	if n <= maxPoints {
		idx := make([]int, n)
		for i := range idx {
			idx[i] = i
		}
		return idx
	}
	if maxPoints == 2 {
		return []int{0, n - 1}
	}

	every := float64(n-2) / float64(maxPoints-2)
	out := make([]int, 0, maxPoints)
	out = append(out, 0)
	a := 0
	for i := 0; i < maxPoints-2; i++ {
		avgStart := int(math.Floor(float64(i+1)*every)) + 1
		avgEnd := int(math.Floor(float64(i+2)*every)) + 1
		if avgEnd > n {
			avgEnd = n
		}
		var avgX, avgY float64
		if avgLen := avgEnd - avgStart; avgLen > 0 {
			for j := avgStart; j < avgEnd; j++ {
				avgX += float64(j)
				avgY += values[j]
			}
			avgX /= float64(avgLen)
			avgY /= float64(avgLen)
		} else {
			avgX = float64(a)
			avgY = values[a]
		}

		rangeStart := int(math.Floor(float64(i)*every)) + 1
		rangeEnd := int(math.Floor(float64(i+1)*every)) + 1
		maxArea := -1.0
		chosen := rangeStart
		for j := rangeStart; j < rangeEnd; j++ {
			area := math.Abs((float64(a)-avgX)*(values[j]-values[a]) - (float64(a)-float64(j))*(avgY-values[a]))
			if area > maxArea {
				maxArea = area
				chosen = j
			}
		}
		out = append(out, chosen)
		a = chosen
	}
	out = append(out, n-1)
	return out
}

// EventIndices returns the indices i>0 where proj(series[i]) differs from
// proj(series[i-1]) using exact decimal comparison. When the events exceed
// maxPoints, maxPoints uniformly spaced events are kept (first and last
// event always included).
func EventIndices[T any](series []T, maxPoints int, proj func(T) decimal.Decimal) []int {
	if maxPoints < 2 {
		maxPoints = 2
	}
	var events []int
	for i := 1; i < len(series); i++ {
		if !proj(series[i]).Equal(proj(series[i-1])) {
			events = append(events, i)
		}
	}
	if len(events) <= maxPoints {
		return events
	}
	out := make([]int, 0, maxPoints)
	step := float64(len(events)-1) / float64(maxPoints-1)
	for i := 0; i < maxPoints; i++ {
		out = append(out, events[int(math.Round(float64(i)*step))])
	}
	return out
}

// Merge unions lttb and events, sorts and deduplicates them, then truncates
// to maxPoints keeping the endpoints and prioritizing events.
func Merge(lttb, events []int, maxPoints int) []int {
	if maxPoints < 2 {
		maxPoints = 2
	}
	seen := make(map[int]bool, len(lttb)+len(events))
	merged := make([]int, 0, len(lttb)+len(events))
	for _, i := range lttb {
		if !seen[i] {
			seen[i] = true
			merged = append(merged, i)
		}
	}
	for _, i := range events {
		if !seen[i] {
			seen[i] = true
			merged = append(merged, i)
		}
	}
	sort.Ints(merged)
	if len(merged) <= maxPoints {
		return merged
	}

	first, last := merged[0], merged[len(merged)-1]
	eventSet := make(map[int]bool, len(events))
	for _, i := range events {
		eventSet[i] = true
	}
	kept := []int{first, last}
	for _, i := range merged {
		if eventSet[i] && i != first && i != last && len(kept) < maxPoints {
			kept = append(kept, i)
		}
	}
	if len(kept) >= maxPoints {
		sort.Ints(kept[:maxPoints])
		return kept[:maxPoints]
	}

	forced := make(map[int]bool, len(kept))
	for _, i := range kept {
		forced[i] = true
	}
	free := make([]int, 0, len(merged)-len(kept))
	for _, i := range merged {
		if !forced[i] {
			free = append(free, i)
		}
	}
	k := maxPoints - len(kept)
	if k > 0 {
		if k == 1 {
			kept = append(kept, free[len(free)/2])
		} else {
			step := float64(len(free)-1) / float64(k-1)
			for i := 0; i < k; i++ {
				kept = append(kept, free[int(math.Round(float64(i)*step))])
			}
		}
	}
	sort.Ints(kept)
	return kept
}

// PortfolioPerformance downsamples a series with LTTB on the Value field.
// The original structs are returned by index, preserving decimal precision.
func PortfolioPerformance(series []model.PortfolioPerformance, maxPoints int) []model.PortfolioPerformance {
	if maxPoints < 2 {
		maxPoints = 2
	}
	if len(series) <= maxPoints {
		return series
	}
	vals := make([]float64, len(series))
	for i, p := range series {
		vals[i], _ = p.Value.Float64()
	}
	idx := IndicesLTTB(vals, maxPoints)
	out := make([]model.PortfolioPerformance, len(idx))
	for i, j := range idx {
		out[i] = series[j]
	}
	return out
}

// PositionPoints downsamples a daily position series: LTTB on MarketValue,
// event indices on qty/cost_basis/realized and a merged, truncated result.
func PositionPoints(series []model.PositionPoint, maxPoints int) []model.PositionPoint {
	if maxPoints < 2 {
		maxPoints = 2
	}
	if len(series) <= maxPoints {
		return series
	}
	vals := make([]float64, len(series))
	for i, p := range series {
		vals[i], _ = p.MarketValue.Float64()
	}
	lttb := IndicesLTTB(vals, maxPoints)
	events := make([]int, 0, maxPoints*3)
	events = append(events, EventIndices(series, maxPoints, func(p model.PositionPoint) decimal.Decimal { return p.Qty })...)
	events = append(events, EventIndices(series, maxPoints, func(p model.PositionPoint) decimal.Decimal { return p.CostBasis })...)
	events = append(events, EventIndices(series, maxPoints, func(p model.PositionPoint) decimal.Decimal { return p.Realized })...)
	idx := Merge(lttb, events, maxPoints)
	out := make([]model.PositionPoint, len(idx))
	for i, j := range idx {
		out[i] = series[j]
	}
	return out
}

// Prices buckets an ascending-by-date series on consecutive index ranges of
// size ceil(n/maxPoints). Each bucket keeps the first open, the last close,
// the max high, the min low and the summed volume; the bucket date and the
// ID/Source/CreatedAt are inherited from the last point of the bucket.
func Prices(series []model.Price, maxPoints int) []model.Price {
	if maxPoints < 2 {
		maxPoints = 2
	}
	n := len(series)
	if n <= maxPoints {
		return series
	}
	bucketSize := (n + maxPoints - 1) / maxPoints
	out := make([]model.Price, 0, (n+bucketSize-1)/bucketSize)
	for start := 0; start < n; start += bucketSize {
		end := start + bucketSize
		if end > n {
			end = n
		}
		last := series[end-1]
		agg := model.Price{
			ID:        last.ID,
			AssetID:   last.AssetID,
			Date:      last.Date,
			Open:      series[start].Open,
			High:      series[start].High,
			Low:       series[start].Low,
			Close:     last.Close,
			Source:    last.Source,
			CreatedAt: last.CreatedAt,
		}
		for i := start + 1; i < end; i++ {
			if series[i].High.GreaterThan(agg.High) {
				agg.High = series[i].High
			}
			if series[i].Low.LessThan(agg.Low) {
				agg.Low = series[i].Low
			}
		}
		for i := start; i < end; i++ {
			agg.Volume += series[i].Volume
		}
		out = append(out, agg)
	}
	return out
}
