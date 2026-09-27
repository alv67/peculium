// Package series materializes the daily position series of portfolios so that
// dashboard, history and performance reads hit Postgres instead of replaying
// the running AVCO engine per request.
package series

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"

	"github.com/alv67/peculium/internal/model"
	"github.com/alv67/peculium/internal/position"
	"github.com/alv67/peculium/internal/repository"
)

// Recompute rebuilds the materialized daily series of a portfolio from its
// transactions, splits and prices, replacing the stored rows. It does not
// trigger any Yahoo fetches.
func Recompute(ctx context.Context, repos *repository.Repository, portfolioID uuid.UUID) error {
	p, err := repos.Portfolio.FindByID(ctx, portfolioID)
	if err != nil {
		return err
	}
	txs, err := repos.Transaction.FindByPortfoliosAsc(ctx, []uuid.UUID{portfolioID})
	if err != nil {
		return err
	}
	holdings, err := repos.Portfolio.HoldingsDetailed(ctx, []uuid.UUID{portfolioID})
	if err != nil {
		return err
	}
	currencySet := map[string]bool{p.Currency: true, "USD": true}
	for _, h := range holdings {
		if h.Currency != "" {
			currencySet[h.Currency] = true
		}
	}
	quotes := make([]string, 0, len(currencySet))
	for c := range currencySet {
		quotes = append(quotes, c)
	}
	dr, err := LoadDateRates(ctx, repos, "USD", quotes)
	if err != nil {
		return err
	}
	prices, err := repos.Price.FindForPortfolio(ctx, portfolioID)
	if err != nil {
		return err
	}

	txByAsset := map[uuid.UUID][]model.TransactionWithAsset{}
	for _, tx := range txs {
		txByAsset[tx.AssetID] = append(txByAsset[tx.AssetID], tx)
	}

	assetIDs := make([]uuid.UUID, 0, len(txByAsset))
	for id := range txByAsset {
		assetIDs = append(assetIDs, id)
	}

	splitRows, err := repos.Split.FindByAssets(ctx, assetIDs)
	if err != nil {
		return err
	}
	splitsByAsset := map[uuid.UUID][]model.Split{}
	for _, sp := range splitRows {
		splitsByAsset[sp.AssetID] = append(splitsByAsset[sp.AssetID], *sp)
	}
	for _, list := range splitsByAsset {
		sort.Slice(list, func(i, j int) bool { return list[i].Date.Before(list[j].Date) })
	}

	assetCurrencies := map[uuid.UUID]string{}
	for _, h := range holdings {
		if h.PortfolioID == portfolioID.String() {
			assetCurrencies[parseUUID(h.AssetID)] = h.Currency
		}
	}

	priceByAsset := map[uuid.UUID]map[time.Time]decimal.Decimal{}
	priceDatesByAsset := map[uuid.UUID][]time.Time{}
	for _, pr := range prices {
		d := DayOf(pr.Date)
		if priceByAsset[pr.AssetID] == nil {
			priceByAsset[pr.AssetID] = map[time.Time]decimal.Decimal{}
		}
		priceByAsset[pr.AssetID][d] = pr.Close
		priceDatesByAsset[pr.AssetID] = append(priceDatesByAsset[pr.AssetID], d)
	}
	for _, ds := range priceDatesByAsset {
		sort.Slice(ds, func(i, j int) bool { return ds[i].Before(ds[j]) })
	}

	var dates []time.Time
	if len(txs) > 0 {
		firstDate := DayOf(txs[0].Date)
		lastDate := DayOf(time.Now())
		dates = make([]time.Time, 0, int(lastDate.Sub(firstDate)/(24*time.Hour))+1)
		for d := firstDate; !d.After(lastDate); d = d.Add(24 * time.Hour) {
			dates = append(dates, d)
		}
	}

	agg := make([]model.PositionPoint, len(dates))
	assets := []model.AssetPositionSeries{}

	sort.Slice(assetIDs, func(i, j int) bool {
		return txByAsset[assetIDs[i]][0].AssetTicker < txByAsset[assetIDs[j]][0].AssetTicker
	})

	for _, aid := range assetIDs {
		assetTxs := txByAsset[aid]
		currency := assetCurrencies[aid]
		st := &position.State{}
		pos := 0
		series := make([]model.PositionPoint, 0, len(dates))

		priceDates := priceDatesByAsset[aid]
		pricePos := 0

		splits := splitsByAsset[aid]
		firstTx := DayOf(assetTxs[0].Date)
		splitPos := 0
		for splitPos < len(splits) && !DayOf(splits[splitPos].Date).After(firstTx) {
			splitPos++
		}
		rawFactor := decimal.NewFromInt(1)
		for _, sp := range splits[splitPos:] {
			rawFactor = rawFactor.Mul(sp.Numerator.Div(sp.Denominator))
		}
		splitInfo := make([]model.SplitInfo, 0, len(splits)-splitPos)
		for _, sp := range splits[splitPos:] {
			splitInfo = append(splitInfo, model.SplitInfo{
				Date:  DayOf(sp.Date),
				Ratio: fmt.Sprintf("%s:%s", sp.Numerator.String(), sp.Denominator.String()),
			})
		}
		var lastMV decimal.Decimal
		for i, d := range dates {
			for pos < len(assetTxs) && !DayOf(assetTxs[pos].Date).After(d) {
				position.Apply(st, assetTxs[pos])
				pos++
			}
			for splitPos < len(splits) && !DayOf(splits[splitPos].Date).After(d) {
				ratio := splits[splitPos].Numerator.Div(splits[splitPos].Denominator)
				position.Apply(st, model.TransactionWithAsset{
					PortfolioID: portfolioID,
					AssetID:     aid,
					Type:        model.TxSplit,
					Quantity:    ratio,
					Date:        splits[splitPos].Date,
					CreatedAt:   splits[splitPos].Date,
				})
				rawFactor = rawFactor.Div(ratio)
				splitPos++
			}
			for pricePos < len(priceDates) && !priceDates[pricePos].After(d) {
				pricePos++
			}
			factor, hasFX := dr.Factor(currency, p.Currency, d)
			mv := decimal.Zero
			if st.Qty.IsPositive() && pricePos > 0 {
				if hasFX {
					mv = st.Qty.Mul(priceByAsset[aid][priceDates[pricePos-1]]).Mul(rawFactor).Mul(factor)
				} else {
					mv = lastMV // FX missing: forward-fill last known value to avoid spurious gaps
				}
			}
			if mv.IsPositive() {
				lastMV = mv
			}
			pt := model.PositionPoint{
				Date:        d,
				Qty:         st.Qty,
				CostBasis:   st.Cost,
				MarketValue: mv,
				Realized:    st.Realized,
			}
			if !d.Before(firstTx) {
				series = append(series, pt)
			}
			agg[i].Date = d
			agg[i].CostBasis = agg[i].CostBasis.Add(pt.CostBasis)
			agg[i].MarketValue = agg[i].MarketValue.Add(pt.MarketValue)
			agg[i].Realized = agg[i].Realized.Add(pt.Realized)
		}
		assets = append(assets, model.AssetPositionSeries{
			AssetID:  aid.String(),
			Ticker:   assetTxs[0].AssetTicker,
			Name:     assetTxs[0].AssetName,
			Currency: currency,
			Series:   series,
			Splits:   splitInfo,
		})
	}

	return repos.Series.ReplacePortfolio(ctx, portfolioID, agg, assets)
}

// WaitForSchema blocks until the materialized series tables exist, or returns
// ctx.Err() on timeout. Used by the worker, which may start before the server
// has run migrations.
func WaitForSchema(ctx context.Context, repos *repository.Repository) error {
	for {
		var name *string
		if err := repos.DB.QueryRow(ctx, `SELECT to_regclass('public.portfolio_series')`).Scan(&name); err == nil && name != nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}

// RecomputeAll rebuilds the materialized series of every portfolio. Failures
// on individual portfolios are logged and do not stop the others.
func RecomputeAll(ctx context.Context, repos *repository.Repository) error {
	ids, err := repos.Portfolio.FindAll(ctx)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if err := Recompute(ctx, repos, id); err != nil {
			log.Warn().Err(err).Str("portfolio_id", id.String()).Msg("series recompute failed")
		}
	}
	return nil
}

// LoadRates loads USD->X rates for every currency appearing in the holdings,
// plus the given baseCurrency, any extra currencies (e.g. portfolio
// currencies) and USD.
func LoadRates(ctx context.Context, repos *repository.Repository, holdings []*model.Holding, baseCurrency string, extra ...string) (map[string]decimal.Decimal, error) {
	quotes := map[string]bool{}
	for _, h := range holdings {
		quotes[h.Currency] = true
	}
	if baseCurrency != "" {
		quotes[baseCurrency] = true
	}
	for _, c := range extra {
		if c != "" {
			quotes[c] = true
		}
	}
	quotes["USD"] = true
	list := make([]string, 0, len(quotes))
	for c := range quotes {
		if c != "" {
			list = append(list, c)
		}
	}
	return repos.FX.LatestByQuotes(ctx, list)
}

// FxFactor returns the factor to convert an amount from currency `from` to
// currency `to`, computed via USD cross rates. Returns ok=false when a needed
// rate is missing.
func FxFactor(rates map[string]decimal.Decimal, from, to string) (decimal.Decimal, bool) {
	if from == to {
		return decimal.NewFromInt(1), true
	}
	usdToFrom, okFrom := usdRate(rates, from)
	if !okFrom {
		return decimal.Zero, false
	}
	usdToTo, okTo := usdRate(rates, to)
	if !okTo {
		return decimal.Zero, false
	}
	if usdToFrom.IsZero() {
		return decimal.Zero, false
	}
	return usdToTo.Div(usdToFrom), true
}

func usdRate(rates map[string]decimal.Decimal, currency string) (decimal.Decimal, bool) {
	if currency == "USD" {
		return decimal.NewFromInt(1), true
	}
	rate, ok := rates[currency]
	return rate, ok
}

// dateRates resolves USD cross rates as of a given calendar day from the
// persisted FX history, falling back to the latest fx_rates snapshot when a day
// has no stored rate.
type dateRates struct {
	base    string
	latest  map[string]decimal.Decimal
	history map[string][]model.FXRatePoint
}

// LoadDateRates loads the latest USD->quote snapshot and the per-date rate
// history for each of the given quote currencies. A missing snapshot entry is
// not an error: such currencies simply resolve to the history or to nothing.
func LoadDateRates(ctx context.Context, repos *repository.Repository, baseCurrency string, quotes []string) (*dateRates, error) {
	snapshot, err := repos.FX.LatestByQuotes(ctx, quotes)
	if err != nil {
		return nil, err
	}
	d := &dateRates{
		base:    baseCurrency,
		latest:  snapshot,
		history: make(map[string][]model.FXRatePoint, len(quotes)),
	}
	for _, q := range quotes {
		if q == "" || q == baseCurrency {
			continue
		}
		h, err := repos.FX.History(ctx, baseCurrency, q)
		if err != nil {
			return nil, err
		}
		d.history[q] = h
	}
	return d, nil
}

// Factor returns the factor to convert an amount from `from` to `to` as of
// `date`, computed via USD cross rates resolved per day. It returns ok=false
// when a needed rate cannot be resolved for that day.
func (d *dateRates) Factor(from, to string, date time.Time) (decimal.Decimal, bool) {
	if from == to {
		return decimal.NewFromInt(1), true
	}
	usdToFrom, okFrom := d.rate(from, date)
	if !okFrom {
		return decimal.Zero, false
	}
	usdToTo, okTo := d.rate(to, date)
	if !okTo {
		return decimal.Zero, false
	}
	if usdToFrom.IsZero() {
		return decimal.Zero, false
	}
	return usdToTo.Div(usdToFrom), true
}

// rate returns the USD->quote rate as of date: the last history point not
// after the target day, or the latest snapshot when no point qualifies.
func (d *dateRates) rate(quote string, date time.Time) (decimal.Decimal, bool) {
	if quote == d.base || quote == "USD" {
		return decimal.NewFromInt(1), true
	}
	points := d.history[quote]
	i := sort.Search(len(points), func(i int) bool { return points[i].Date.After(date) })
	if i > 0 {
		return points[i-1].Rate, true
	}
	if rate, ok := d.latest[quote]; ok {
		return rate, true
	}
	return decimal.Zero, false
}

// DayOf truncates t to its UTC calendar day.
func DayOf(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func parseUUID(s string) uuid.UUID {
	id, _ := uuid.Parse(s)
	return id
}
