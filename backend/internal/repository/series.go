package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/alv67/peculium/internal/model"
)

// SeriesRepository exposes the materialized daily position series tables.
type SeriesRepository interface {
	ReplacePortfolio(ctx context.Context, portfolioID uuid.UUID, agg []model.PositionPoint, assets []model.AssetPositionSeries) error
	FindPortfolioAgg(ctx context.Context, portfolioID uuid.UUID) ([]model.PositionPoint, error)
	FindPortfolio(ctx context.Context, portfolioID uuid.UUID) ([]model.AssetPositionSeries, error)
	HasPortfolio(ctx context.Context, portfolioID uuid.UUID) (bool, error)
}

type seriesRepo struct {
	db DBTX
}

// batchDB is implemented by both *pgxpool.Pool and pgx.Tx so that
// ReplacePortfolio can run against a standalone transaction or inside a
// caller-provided one.
type batchDB interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	SendBatch(context.Context, *pgx.Batch) pgx.BatchResults
}

const seriesBatchSize = 500

// ReplacePortfolio atomically replaces the stored series of a portfolio with
// the given aggregate points and per-asset series. Empty slices only delete.
func (r *seriesRepo) ReplacePortfolio(ctx context.Context, portfolioID uuid.UUID, agg []model.PositionPoint, assets []model.AssetPositionSeries) error {
	if pool, ok := r.db.(*pgxpool.Pool); ok {
		tx, err := pool.Begin(ctx)
		if err != nil {
			return err
		}
		defer func() { _ = tx.Rollback(ctx) }()
		if err := replacePortfolio(ctx, tx, portfolioID, agg, assets); err != nil {
			return err
		}
		return tx.Commit(ctx)
	}
	q, ok := r.db.(batchDB)
	if !ok {
		return errors.New("series: database handle does not support transactions")
	}
	return replacePortfolio(ctx, q, portfolioID, agg, assets)
}

func replacePortfolio(ctx context.Context, q batchDB, portfolioID uuid.UUID, agg []model.PositionPoint, assets []model.AssetPositionSeries) error {
	if _, err := q.Exec(ctx, `DELETE FROM portfolio_series WHERE portfolio_id = $1`, portfolioID); err != nil {
		return err
	}
	if _, err := q.Exec(ctx, `DELETE FROM asset_series WHERE portfolio_id = $1`, portfolioID); err != nil {
		return err
	}
	if err := insertPortfolioAgg(ctx, q, portfolioID, agg); err != nil {
		return err
	}
	for _, a := range assets {
		if err := insertAssetSeries(ctx, q, portfolioID, a); err != nil {
			return err
		}
	}
	return nil
}

func insertPortfolioAgg(ctx context.Context, q batchDB, portfolioID uuid.UUID, agg []model.PositionPoint) error {
	for start := 0; start < len(agg); start += seriesBatchSize {
		end := min(start+seriesBatchSize, len(agg))
		batch := &pgx.Batch{}
		for _, pt := range agg[start:end] {
			batch.Queue(`INSERT INTO portfolio_series (portfolio_id, date, qty, cost_basis, market_value, realized)
				VALUES ($1, $2, $3, $4, $5, $6)`,
				portfolioID, pt.Date, pt.Qty, pt.CostBasis, pt.MarketValue, pt.Realized)
		}
		if err := sendBatch(ctx, q, batch); err != nil {
			return err
		}
	}
	return nil
}

func insertAssetSeries(ctx context.Context, q batchDB, portfolioID uuid.UUID, a model.AssetPositionSeries) error {
	assetID, err := uuid.Parse(a.AssetID)
	if err != nil {
		return err
	}
	for start := 0; start < len(a.Series); start += seriesBatchSize {
		end := min(start+seriesBatchSize, len(a.Series))
		batch := &pgx.Batch{}
		for _, pt := range a.Series[start:end] {
			batch.Queue(`INSERT INTO asset_series (portfolio_id, asset_id, date, qty, cost_basis, market_value, realized)
				VALUES ($1, $2, $3, $4, $5, $6, $7)`,
				portfolioID, assetID, pt.Date, pt.Qty, pt.CostBasis, pt.MarketValue, pt.Realized)
		}
		if err := sendBatch(ctx, q, batch); err != nil {
			return err
		}
	}
	return nil
}

func sendBatch(ctx context.Context, q batchDB, batch *pgx.Batch) error {
	br := q.SendBatch(ctx, batch)
	defer br.Close()
	for i := 0; i < batch.Len(); i++ {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}
	return nil
}

// FindPortfolioAgg returns the portfolio-level daily aggregate points.
func (r *seriesRepo) FindPortfolioAgg(ctx context.Context, portfolioID uuid.UUID) ([]model.PositionPoint, error) {
	rows, err := r.db.Query(ctx,
		`SELECT date, qty, cost_basis, market_value, realized
		 FROM portfolio_series WHERE portfolio_id = $1 ORDER BY date`,
		portfolioID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pts := make([]model.PositionPoint, 0)
	for rows.Next() {
		var pt model.PositionPoint
		if err := rows.Scan(&pt.Date, &pt.Qty, &pt.CostBasis, &pt.MarketValue, &pt.Realized); err != nil {
			return nil, err
		}
		pts = append(pts, pt)
	}
	return pts, rows.Err()
}

// FindPortfolio returns the per-asset daily series grouped by asset.
func (r *seriesRepo) FindPortfolio(ctx context.Context, portfolioID uuid.UUID) ([]model.AssetPositionSeries, error) {
	rows, err := r.db.Query(ctx,
		`SELECT s.asset_id, a.ticker, a.name, a.currency, s.date, s.qty, s.cost_basis, s.market_value, s.realized
		 FROM asset_series s
		 JOIN assets a ON a.id = s.asset_id
		 WHERE s.portfolio_id = $1
		 ORDER BY a.ticker, s.date`,
		portfolioID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	assets := make([]model.AssetPositionSeries, 0)
	lastID := uuid.Nil
	for rows.Next() {
		var assetID uuid.UUID
		var ticker, name, currency string
		var pt model.PositionPoint
		if err := rows.Scan(&assetID, &ticker, &name, &currency, &pt.Date, &pt.Qty, &pt.CostBasis, &pt.MarketValue, &pt.Realized); err != nil {
			return nil, err
		}
		if lastID == uuid.Nil || lastID != assetID {
			assets = append(assets, model.AssetPositionSeries{
				AssetID:  assetID.String(),
				Ticker:   ticker,
				Name:     name,
				Currency: currency,
				Series:   []model.PositionPoint{pt},
				Splits:   []model.SplitInfo{},
			})
			lastID = assetID
			continue
		}
		a := &assets[len(assets)-1]
		a.Series = append(a.Series, pt)
	}
	return assets, rows.Err()
}

// HasPortfolio reports whether the portfolio has a materialized aggregate.
func (r *seriesRepo) HasPortfolio(ctx context.Context, portfolioID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM portfolio_series WHERE portfolio_id = $1)`,
		portfolioID,
	).Scan(&exists)
	return exists, err
}
