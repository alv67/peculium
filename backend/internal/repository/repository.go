package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DBTX abstracts the subset of pgx used by the repositories so that they can
// run against either the connection pool or a transaction.
type DBTX interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

type Repository struct {
	DB          *pgxpool.Pool
	User        UserRepository
	Asset       AssetRepository
	Portfolio   PortfolioRepository
	Transaction TransactionRepository
	Price       PriceRepository
	Lookup      LookupRepository
	FX          FXRepository
	Split       SplitRepository
	Currency    CurrencyRepository
	Series      SeriesRepository
	Health      HealthRepository
	Exposure    ExposureRepository
	Settings    SettingsRepository
	Job         JobRepository
}

func New(db *pgxpool.Pool, lookup LookupRepository) *Repository {
	return &Repository{
		DB:          db,
		User:        &userRepo{db},
		Asset:       &assetRepo{db},
		Portfolio:   &portfolioRepo{db: db, transactions: &transactionRepo{db}, splits: &splitRepo{db}},
		Transaction: &transactionRepo{db},
		Price:       &priceRepo{db},
		Lookup:      lookup,
		FX:          &fxRepo{db},
		Split:       &splitRepo{db},
		Currency:    &currencyRepo{db},
		Series:      &seriesRepo{db},
		Health:      &healthRepo{db},
		Exposure:    &exposureRepo{db},
		Settings:    &settingsRepo{db},
		Job:         &jobRepo{db},
	}
}

func (r *Repository) Ping(ctx context.Context) error {
	return r.DB.Ping(ctx)
}

// WithTx runs fn with a Repository whose repositories are bound to a single
// database transaction. If fn returns an error the transaction is rolled back,
// otherwise it is committed.
func (r *Repository) WithTx(ctx context.Context, fn func(*Repository) error) error {
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	rr := &Repository{
		DB:          r.DB,
		User:        &userRepo{db: tx},
		Asset:       &assetRepo{db: tx},
		Portfolio:   &portfolioRepo{db: tx, transactions: &transactionRepo{db: tx}, splits: &splitRepo{db: tx}},
		Transaction: &transactionRepo{db: tx},
		Price:       &priceRepo{db: tx},
		Lookup:      r.Lookup,
		FX:          &fxRepo{db: tx},
		Split:       &splitRepo{db: tx},
		Currency:    &currencyRepo{db: tx},
		Series:      &seriesRepo{db: tx},
		Exposure:    &exposureRepo{db: tx},
		Settings:    &settingsRepo{db: tx},
	}

	if err := fn(rr); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
