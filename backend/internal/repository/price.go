package repository

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/alv67/peculium/internal/model"
)

type PriceRepository interface {
	Create(ctx context.Context, price *model.Price) (*model.Price, error)
	FindByAsset(ctx context.Context, assetID uuid.UUID) ([]*model.Price, error)
	FindLatest(ctx context.Context, assetID uuid.UUID) (*model.Price, error)
	FindLatestForAssets(ctx context.Context, assetIDs []uuid.UUID) (map[uuid.UUID]*model.Price, error)
	FindForPortfolio(ctx context.Context, portfolioID uuid.UUID) ([]model.Price, error)
	MinMaxDate(ctx context.Context, assetID uuid.UUID) (earliest, latest *time.Time, err error)
	ReferenceCloses(ctx context.Context, assetID uuid.UUID, dates []time.Time) (map[time.Time]decimal.Decimal, error)
}

type priceRepo struct {
	db DBTX
}

func (r *priceRepo) Create(ctx context.Context, price *model.Price) (*model.Price, error) {
	p := &model.Price{}
	err := r.db.QueryRow(ctx,
		`INSERT INTO prices (asset_id, date, open, high, low, close, volume, source)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 ON CONFLICT (asset_id, date) DO UPDATE
		   SET open = $3, high = $4, low = $5, close = $6, volume = $7, source = $8
		 RETURNING id, asset_id, date, open, high, low, close, volume, source, created_at`,
		price.AssetID, price.Date, price.Open, price.High, price.Low, price.Close, price.Volume, price.Source,
	).Scan(&p.ID, &p.AssetID, &p.Date, &p.Open, &p.High, &p.Low, &p.Close, &p.Volume, &p.Source, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (r *priceRepo) FindByAsset(ctx context.Context, assetID uuid.UUID) ([]*model.Price, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, asset_id, date, open, high, low, close, volume, source, created_at
		 FROM prices WHERE asset_id = $1 ORDER BY date DESC`,
		assetID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	prices := make([]*model.Price, 0)
	for rows.Next() {
		p := &model.Price{}
		if err := rows.Scan(&p.ID, &p.AssetID, &p.Date, &p.Open, &p.High, &p.Low, &p.Close, &p.Volume, &p.Source, &p.CreatedAt); err != nil {
			return nil, err
		}
		prices = append(prices, p)
	}
	return prices, nil
}

func (r *priceRepo) FindLatest(ctx context.Context, assetID uuid.UUID) (*model.Price, error) {
	p := &model.Price{}
	err := r.db.QueryRow(ctx,
		`SELECT id, asset_id, date, open, high, low, close, volume, source, created_at
		 FROM prices WHERE asset_id = $1 ORDER BY date DESC LIMIT 1`,
		assetID,
	).Scan(&p.ID, &p.AssetID, &p.Date, &p.Open, &p.High, &p.Low, &p.Close, &p.Volume, &p.Source, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (r *priceRepo) FindLatestForAssets(ctx context.Context, assetIDs []uuid.UUID) (map[uuid.UUID]*model.Price, error) {
	if len(assetIDs) == 0 {
		return map[uuid.UUID]*model.Price{}, nil
	}
	rows, err := r.db.Query(ctx,
		`SELECT DISTINCT ON (asset_id) asset_id, id, date, open, high, low, close, volume, source, created_at
		 FROM prices
		 WHERE asset_id = ANY($1::uuid[])
		 ORDER BY asset_id, date DESC`,
		assetIDs,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	latest := make(map[uuid.UUID]*model.Price)
	for rows.Next() {
		p := &model.Price{}
		if err := rows.Scan(&p.AssetID, &p.ID, &p.Date, &p.Open, &p.High, &p.Low, &p.Close, &p.Volume, &p.Source, &p.CreatedAt); err != nil {
			return nil, err
		}
		latest[p.AssetID] = p
	}
	return latest, rows.Err()
}

func (r *priceRepo) FindForPortfolio(ctx context.Context, portfolioID uuid.UUID) ([]model.Price, error) {
	rows, err := r.db.Query(ctx,
		`SELECT p.asset_id, p.date, p.close
		 FROM prices p
		 WHERE p.asset_id IN (SELECT DISTINCT asset_id FROM transactions WHERE portfolio_id = $1)
		   AND p.date >= (SELECT MIN(date::date) FROM transactions WHERE portfolio_id = $1)
		 ORDER BY p.date ASC`,
		portfolioID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	prices := make([]model.Price, 0)
	for rows.Next() {
		var pr model.Price
		if err := rows.Scan(&pr.AssetID, &pr.Date, &pr.Close); err != nil {
			return nil, err
		}
		prices = append(prices, pr)
	}
	return prices, rows.Err()
}

func (r *priceRepo) MinMaxDate(ctx context.Context, assetID uuid.UUID) (*time.Time, *time.Time, error) {
	var earliest, latest *time.Time
	err := r.db.QueryRow(ctx,
		`SELECT MIN(date), MAX(date) FROM prices WHERE asset_id = $1`,
		assetID,
	).Scan(&earliest, &latest)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	return earliest, latest, nil
}

// ReferenceCloses returns, for each reference date, the close of the most
// recent stored price not later than that date. Dates with no such price are
// absent from the returned map.
func (r *priceRepo) ReferenceCloses(ctx context.Context, assetID uuid.UUID, dates []time.Time) (map[time.Time]decimal.Decimal, error) {
	if len(dates) == 0 {
		return map[time.Time]decimal.Decimal{}, nil
	}

	refDates := make([]time.Time, len(dates))
	for i, d := range dates {
		refDates[i] = time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
	}
	sort.Slice(refDates, func(i, j int) bool { return refDates[i].Before(refDates[j]) })

	rows, err := r.db.Query(ctx,
		`SELECT date, close FROM prices WHERE asset_id = $1 ORDER BY date ASC`,
		assetID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type closeBar struct {
		date  time.Time
		close decimal.Decimal
	}
	var bars []closeBar
	for rows.Next() {
		var b closeBar
		if err := rows.Scan(&b.date, &b.close); err != nil {
			return nil, err
		}
		bars = append(bars, b)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	closes := make(map[time.Time]decimal.Decimal, len(refDates))
	pos := 0
	for _, ref := range refDates {
		for pos < len(bars) && !bars[pos].date.After(ref) {
			pos++
		}
		if pos > 0 {
			closes[ref] = bars[pos-1].close
		}
	}
	return closes, nil
}
