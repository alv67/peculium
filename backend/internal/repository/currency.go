package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/alv67/peculium/internal/model"
)

type CurrencyRepository interface {
	ListEnabled(ctx context.Context) ([]model.Currency, error)
	ListAll(ctx context.Context) ([]model.Currency, error)
	Get(ctx context.Context, code string) (*model.Currency, error)
	Create(ctx context.Context, c *model.Currency) error
	Delete(ctx context.Context, code string) error
	EnabledByCodes(ctx context.Context, codes []string) ([]string, error)
	CountInUse(ctx context.Context, code string) (int, error)
}

type currencyRepo struct {
	db DBTX
}

const currencyColumns = `code, name, symbol, enabled, sort, created_at`

func scanCurrency(row pgx.Row) (*model.Currency, error) {
	c := &model.Currency{}
	err := row.Scan(&c.Code, &c.Name, &c.Symbol, &c.Enabled, &c.Sort, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (r *currencyRepo) ListEnabled(ctx context.Context) ([]model.Currency, error) {
	return r.list(ctx, `WHERE enabled ORDER BY sort, code`)
}

func (r *currencyRepo) ListAll(ctx context.Context) ([]model.Currency, error) {
	return r.list(ctx, `ORDER BY sort, code`)
}

func (r *currencyRepo) list(ctx context.Context, where string) ([]model.Currency, error) {
	rows, err := r.db.Query(ctx, `SELECT `+currencyColumns+` FROM supported_currencies `+where)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	currencies := make([]model.Currency, 0)
	for rows.Next() {
		c := &model.Currency{}
		if err := rows.Scan(&c.Code, &c.Name, &c.Symbol, &c.Enabled, &c.Sort, &c.CreatedAt); err != nil {
			return nil, err
		}
		currencies = append(currencies, *c)
	}
	return currencies, rows.Err()
}

func (r *currencyRepo) Get(ctx context.Context, code string) (*model.Currency, error) {
	row := r.db.QueryRow(ctx, `SELECT `+currencyColumns+` FROM supported_currencies WHERE code = $1`, code)
	c, err := scanCurrency(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return c, err
}

func (r *currencyRepo) Create(ctx context.Context, c *model.Currency) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO supported_currencies (code, name, symbol, enabled, sort)
		 VALUES ($1, $2, $3, $4, $5)`,
		c.Code, c.Name, c.Symbol, c.Enabled, c.Sort,
	)
	return err
}

func (r *currencyRepo) Delete(ctx context.Context, code string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM supported_currencies WHERE code = $1`, code)
	return err
}

// EnabledByCodes returns the subset of codes that are present in the enabled
// whitelist. Used to filter arbitrary currency values coming from assets.
func (r *currencyRepo) EnabledByCodes(ctx context.Context, codes []string) ([]string, error) {
	if len(codes) == 0 {
		return []string{}, nil
	}
	rows, err := r.db.Query(ctx,
		`SELECT code FROM supported_currencies WHERE enabled AND code = ANY($1::text[])`,
		codes,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	enabled := make([]string, 0)
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, err
		}
		enabled = append(enabled, code)
	}
	return enabled, rows.Err()
}

// CountInUse counts assets and portfolios referencing the currency.
func (r *currencyRepo) CountInUse(ctx context.Context, code string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT
			(SELECT COUNT(*) FROM assets WHERE currency = $1) +
			(SELECT COUNT(*) FROM portfolios WHERE currency = $1)`,
		code,
	).Scan(&count)
	return count, err
}
