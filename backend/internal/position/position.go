package position

import (
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/alv67/peculium/internal/model"
)

type State struct {
	Qty         decimal.Decimal
	Avg         decimal.Decimal
	AvgCCY      decimal.Decimal
	Cost        decimal.Decimal
	CostCCY     decimal.Decimal
	Realized    decimal.Decimal
	RealizedCCY decimal.Decimal

	ClosedCost    decimal.Decimal // AVCO cost of sold lots, portfolio currency
	ClosedCostCCY decimal.Decimal // AVCO cost of sold lots, asset currency
	Proceeds      decimal.Decimal // net sale proceeds, portfolio currency
	ProceedsCCY   decimal.Decimal // net sale proceeds, asset currency
	Dividends     decimal.Decimal // dividends, portfolio currency
	DividendsCCY  decimal.Decimal // dividends, asset currency
}

func Apply(s *State, tx model.TransactionWithAsset) {
	switch tx.Type {
	case model.TxBuy:
		costPF := tx.Quantity.Mul(tx.Price).Add(tx.Fees)
		costCCY := tx.Quantity.Mul(tx.Price)
		newQty := s.Qty.Add(tx.Quantity)
		if newQty.IsPositive() {
			s.Avg = s.Cost.Add(costPF).Div(newQty)
			s.AvgCCY = s.CostCCY.Add(costCCY).Div(newQty)
		}
		s.Qty = newQty
		s.Cost = s.Cost.Add(costPF)
		s.CostCCY = s.CostCCY.Add(costCCY)
	case model.TxSell:
		costSold := s.Avg.Mul(tx.Quantity)
		costSoldCCY := s.AvgCCY.Mul(tx.Quantity)
		proceedsPF := tx.Quantity.Mul(tx.Price).Sub(tx.Fees)
		proceedsCCY := tx.Quantity.Mul(tx.Price)
		s.Realized = s.Realized.Add(proceedsPF.Sub(costSold))
		s.RealizedCCY = s.RealizedCCY.Add(proceedsCCY.Sub(costSoldCCY))
		s.ClosedCost = s.ClosedCost.Add(costSold)
		s.ClosedCostCCY = s.ClosedCostCCY.Add(costSoldCCY)
		s.Proceeds = s.Proceeds.Add(proceedsPF)
		s.ProceedsCCY = s.ProceedsCCY.Add(proceedsCCY)
		s.Qty = s.Qty.Sub(tx.Quantity)
		s.Cost = s.Cost.Sub(costSold)
		s.CostCCY = s.CostCCY.Sub(costSoldCCY)
	case model.TxSplit:
		if s.Qty.IsPositive() {
			s.Qty = s.Qty.Mul(tx.Quantity)
			s.Avg = s.Avg.Div(tx.Quantity)
			s.AvgCCY = s.AvgCCY.Div(tx.Quantity)
		}
	case model.TxFee:
		var feePF, feeCCY decimal.Decimal
		if tx.Quantity.IsPositive() {
			feePF = tx.Price.Mul(tx.Quantity)
			feeCCY = tx.Price.Mul(tx.Quantity)
		} else {
			feePF = tx.Price
			feeCCY = tx.Price
		}
		s.Cost = s.Cost.Add(feePF)
		s.CostCCY = s.CostCCY.Add(feeCCY)
		if s.Qty.IsPositive() {
			s.Avg = s.Cost.Div(s.Qty)
			s.AvgCCY = s.CostCCY.Div(s.Qty)
		}
	case model.TxDividend:
		var divPF, divCCY decimal.Decimal
		if tx.Quantity.IsPositive() {
			divPF = tx.Price.Mul(tx.Quantity)
			divCCY = tx.Price.Mul(tx.Quantity)
		} else {
			divPF = tx.Price
			divCCY = tx.Price
		}
		s.Realized = s.Realized.Add(divPF)
		s.RealizedCCY = s.RealizedCCY.Add(divCCY)
		s.Dividends = s.Dividends.Add(divPF)
		s.DividendsCCY = s.DividendsCCY.Add(divCCY)
	}
}

type SplitEvent struct {
	AssetID uuid.UUID
	Date    time.Time
	Ratio   decimal.Decimal
}

// Walk applies the running AVCO timeline per portfolio+asset, injecting the
// asset-level split events into each portfolio's timeline.
func Walk(txs []model.TransactionWithAsset, splits []SplitEvent) map[string]*State {
	byKey := map[string][]model.TransactionWithAsset{}
	byAsset := map[uuid.UUID][]SplitEvent{}
	for _, tx := range txs {
		key := fmt.Sprintf("%s|%s", tx.PortfolioID, tx.AssetID)
		byKey[key] = append(byKey[key], tx)
	}
	for _, s := range splits {
		byAsset[s.AssetID] = append(byAsset[s.AssetID], s)
	}

	states := map[string]*State{}
	for key, group := range byKey {
		events := make([]model.TransactionWithAsset, 0, len(group)+len(byAsset[group[0].AssetID]))
		events = append(events, group...)
		for _, e := range byAsset[group[0].AssetID] {
			events = append(events, model.TransactionWithAsset{
				PortfolioID: group[0].PortfolioID,
				AssetID:     e.AssetID,
				Type:        model.TxSplit,
				Quantity:    e.Ratio,
				Date:        e.Date,
				CreatedAt:   e.Date,
			})
		}
		sort.SliceStable(events, func(i, j int) bool {
			if !events[i].Date.Equal(events[j].Date) {
				return events[i].Date.Before(events[j].Date)
			}
			if events[i].Type == model.TxSplit != (events[j].Type == model.TxSplit) {
				return events[j].Type == model.TxSplit
			}
			return events[i].CreatedAt.Before(events[j].CreatedAt)
		})
		st := &State{}
		for _, ev := range events {
			Apply(st, ev)
		}
		states[key] = st
	}
	return states
}
