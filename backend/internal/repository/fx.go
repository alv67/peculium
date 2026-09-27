package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/alv67/peculium/internal/model"
)

type FXRepository interface {
	Upsert(ctx context.Context, base, quote string, rate decimal.Decimal) error
	LatestByQuotes(ctx context.Context, quotes []string) (map[string]decimal.Decimal, error)
	FetchedAt(ctx context.Context, quote string) (*time.Time, error)
	UpsertHistory(ctx context.Context, base, quote string, date time.Time, rate decimal.Decimal, source string) error
	MinMaxDate(ctx context.Context, base, quote string) (earliest, latest *time.Time, err error)
	RateForDate(ctx context.Context, base, quote string, date time.Time) (decimal.Decimal, error)
	History(ctx context.Context, base, quote string) ([]model.FXRatePoint, error)
}

type fxRepo struct {
	db DBTX
}

func (r *fxRepo) Upsert(ctx context.Context, base, quote string, rate decimal.Decimal) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO fx_rates (base_currency, quote_currency, rate, fetched_at)
		 VALUES ($1, $2, $3, NOW())
		 ON CONFLICT (base_currency, quote_currency)
		 DO UPDATE SET rate = EXCLUDED.rate, fetched_at = NOW()`,
		base, quote, rate,
	)
	return err
}

func (r *fxRepo) LatestByQuotes(ctx context.Context, quotes []string) (map[string]decimal.Decimal, error) {
	if len(quotes) == 0 {
		return map[string]decimal.Decimal{}, nil
	}
	rows, err := r.db.Query(ctx,
		`SELECT quote_currency, rate FROM fx_rates WHERE base_currency = 'USD' AND quote_currency = ANY($1::text[])`,
		quotes,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rates := make(map[string]decimal.Decimal)
	for rows.Next() {
		var quote string
		var rate decimal.Decimal
		if err := rows.Scan(&quote, &rate); err != nil {
			return nil, err
		}
		rates[quote] = rate
	}
	return rates, rows.Err()
}

func (r *fxRepo) FetchedAt(ctx context.Context, quote string) (*time.Time, error) {
	var at time.Time
	err := r.db.QueryRow(ctx,
		`SELECT fetched_at FROM fx_rates WHERE base_currency = 'USD' AND quote_currency = $1`,
		quote,
	).Scan(&at)
	if err != nil {
		return nil, err
	}
	return &at, nil
}

func (r *fxRepo) UpsertHistory(ctx context.Context, base, quote string, date time.Time, rate decimal.Decimal, source string) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO fx_history (base_currency, quote_currency, date, rate, source)
		 VALUES ($1, $2, $3::date, $4, $5)
		 ON CONFLICT (base_currency, quote_currency, date)
		 DO UPDATE SET rate = EXCLUDED.rate, source = EXCLUDED.source`,
		base, quote, date, rate, source,
	)
	return err
}

func (r *fxRepo) MinMaxDate(ctx context.Context, base, quote string) (*time.Time, *time.Time, error) {
	var earliest, latest *time.Time
	err := r.db.QueryRow(ctx,
		`SELECT MIN(date)::date, MAX(date)::date FROM fx_history WHERE base_currency = $1 AND quote_currency = $2`,
		base, quote,
	).Scan(&earliest, &latest)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	return earliest, latest, nil
}

func (r *fxRepo) RateForDate(ctx context.Context, base, quote string, date time.Time) (decimal.Decimal, error) {
	var rate decimal.Decimal
	err := r.db.QueryRow(ctx,
		`SELECT rate FROM fx_history
		 WHERE base_currency = $1 AND quote_currency = $2 AND date <= $3::date
		 ORDER BY date DESC LIMIT 1`,
		base, quote, date,
	).Scan(&rate)
	if err == nil {
		return rate, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return decimal.Zero, err
	}
	err = r.db.QueryRow(ctx,
		`SELECT rate FROM fx_rates WHERE base_currency = $1 AND quote_currency = $2`,
		base, quote,
	).Scan(&rate)
	if err == nil {
		return rate, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return decimal.Zero, err
	}
	return decimal.Zero, pgx.ErrNoRows
}

func (r *fxRepo) History(ctx context.Context, base, quote string) ([]model.FXRatePoint, error) {
	rows, err := r.db.Query(ctx,
		`SELECT date, rate FROM fx_history
		 WHERE base_currency = $1 AND quote_currency = $2 ORDER BY date ASC`,
		base, quote,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	points := make([]model.FXRatePoint, 0)
	for rows.Next() {
		var p model.FXRatePoint
		if err := rows.Scan(&p.Date, &p.Rate); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	return points, rows.Err()
}
