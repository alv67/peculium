package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/alv67/peculium/internal/model"
	"github.com/alv67/peculium/internal/position"
)

type PortfolioRepository interface {
	Create(ctx context.Context, p *model.Portfolio) (*model.Portfolio, error)
	FindByID(ctx context.Context, id uuid.UUID) (*model.Portfolio, error)
	FindByUser(ctx context.Context, userID uuid.UUID) ([]*model.Portfolio, error)
	Update(ctx context.Context, p *model.Portfolio) error
	Delete(ctx context.Context, id uuid.UUID) error
	HeldAssets(ctx context.Context, portfolioID uuid.UUID) ([]*model.Asset, error)
	HoldingsDetailed(ctx context.Context, portfolioIDs []uuid.UUID) ([]*model.Holding, error)
	FindAll(ctx context.Context) ([]uuid.UUID, error)
}

type portfolioRepo struct {
	db           DBTX
	transactions TransactionRepository
	splits       SplitRepository
}

func (r *portfolioRepo) Create(ctx context.Context, p *model.Portfolio) (*model.Portfolio, error) {
	portfolio := &model.Portfolio{}
	err := r.db.QueryRow(ctx,
		`INSERT INTO portfolios (user_id, name, description, currency)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, user_id, name, description, currency, created_at, updated_at`,
		p.UserID, p.Name, p.Description, p.Currency,
	).Scan(&portfolio.ID, &portfolio.UserID, &portfolio.Name, &portfolio.Description, &portfolio.Currency, &portfolio.CreatedAt, &portfolio.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return portfolio, nil
}

func (r *portfolioRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.Portfolio, error) {
	p := &model.Portfolio{}
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, name, description, currency, created_at, updated_at FROM portfolios WHERE id = $1`,
		id,
	).Scan(&p.ID, &p.UserID, &p.Name, &p.Description, &p.Currency, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (r *portfolioRepo) FindByUser(ctx context.Context, userID uuid.UUID) ([]*model.Portfolio, error) {
	rows, err := r.db.Query(ctx,
		`SELECT p.id, p.user_id, p.name, p.description, p.currency, p.created_at, p.updated_at
		 FROM portfolios p
		 LEFT JOIN portfolio_shares ps ON ps.portfolio_id = p.id
		 WHERE p.user_id = $1 OR ps.user_id = $1
		 ORDER BY p.created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	portfolios := make([]*model.Portfolio, 0)
	for rows.Next() {
		p := &model.Portfolio{}
		if err := rows.Scan(&p.ID, &p.UserID, &p.Name, &p.Description, &p.Currency, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		portfolios = append(portfolios, p)
	}
	return portfolios, nil
}

// FindAll returns the ids of every portfolio in the system.
func (r *portfolioRepo) FindAll(ctx context.Context) ([]uuid.UUID, error) {
	rows, err := r.db.Query(ctx, `SELECT id FROM portfolios`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make([]uuid.UUID, 0)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *portfolioRepo) Update(ctx context.Context, p *model.Portfolio) error {
	_, err := r.db.Exec(ctx,
		`UPDATE portfolios SET name = $1, description = $2, currency = $3, updated_at = NOW() WHERE id = $4`,
		p.Name, p.Description, p.Currency, p.ID,
	)
	return err
}

func (r *portfolioRepo) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM portfolios WHERE id = $1`, id)
	return err
}

func (r *portfolioRepo) HeldAssets(ctx context.Context, portfolioID uuid.UUID) ([]*model.Asset, error) {
	rows, err := r.db.Query(ctx, `
		WITH buy AS (
			SELECT asset_id, SUM(quantity) AS qty
			FROM transactions
			WHERE portfolio_id = $1 AND type = 'buy'
			GROUP BY asset_id
		),
		sell AS (
			SELECT asset_id, SUM(quantity) AS qty
			FROM transactions
			WHERE portfolio_id = $1 AND type = 'sell'
			GROUP BY asset_id
		)
		SELECT a.id, a.ticker, a.isin, a.name, a.type, a.sector, a.asset_class, a.country, a.currency, a.created_at, a.price_fetched_at
		FROM buy b
		LEFT JOIN sell s ON s.asset_id = b.asset_id
		JOIN assets a ON a.id = b.asset_id
		WHERE b.qty - COALESCE(s.qty, 0) > 0
	`, portfolioID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	assets := make([]*model.Asset, 0)
	for rows.Next() {
		a := &model.Asset{}
		if err := rows.Scan(&a.ID, &a.Ticker, &a.ISIN, &a.Name, &a.Type, &a.Sector, &a.AssetClass, &a.Country, &a.Currency, &a.CreatedAt, &a.PriceFetchedAt); err != nil {
			return nil, err
		}
		assets = append(assets, a)
	}
	return assets, rows.Err()
}

func (r *portfolioRepo) HoldingsDetailed(ctx context.Context, portfolioIDs []uuid.UUID) ([]*model.Holding, error) {
	if len(portfolioIDs) == 0 {
		return []*model.Holding{}, nil
	}
	rows, err := r.db.Query(ctx, `
		SELECT DISTINCT t.portfolio_id, t.asset_id, a.ticker, a.name, a.currency,
			COALESCE(p.close, 0) AS last_close, (p.close IS NOT NULL) AS has_price,
			a.country, a.sector, a.asset_class, a.type, a.price_fetched_at
		FROM transactions t
		JOIN assets a ON a.id = t.asset_id
		LEFT JOIN LATERAL (
			SELECT close FROM prices WHERE asset_id = t.asset_id ORDER BY date DESC LIMIT 1
		) p ON TRUE
		WHERE t.portfolio_id = ANY($1::uuid[])
		ORDER BY t.portfolio_id, a.ticker`, portfolioIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	base := make([]*model.Holding, 0)
	for rows.Next() {
		h := &model.Holding{}
		if err := rows.Scan(&h.PortfolioID, &h.AssetID, &h.Ticker, &h.Name, &h.Currency, &h.LastClose, &h.HasPrice, &h.Country, &h.Sector, &h.AssetClass, &h.Type, &h.PriceFetchedAt); err != nil {
			return nil, err
		}
		base = append(base, h)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	txs, err := r.transactions.FindByPortfoliosAsc(ctx, portfolioIDs)
	if err != nil {
		return nil, err
	}

	assetIDs := make([]uuid.UUID, 0, len(txs))
	seen := map[uuid.UUID]bool{}
	for _, tx := range txs {
		if !seen[tx.AssetID] {
			seen[tx.AssetID] = true
			assetIDs = append(assetIDs, tx.AssetID)
		}
	}
	splitRows, err := r.splits.FindByAssets(ctx, assetIDs)
	if err != nil {
		return nil, err
	}
	splitEvents := make([]position.SplitEvent, 0, len(splitRows))
	for _, sp := range splitRows {
		splitEvents = append(splitEvents, position.SplitEvent{
			AssetID: sp.AssetID,
			Date:    sp.Date,
			Ratio:   sp.Numerator.Div(sp.Denominator),
		})
	}
	states := position.Walk(txs, splitEvents)

	for _, h := range base {
		if st, ok := states[h.PortfolioID+"|"+h.AssetID]; ok {
			h.Qty = st.Qty
			h.Cost = st.Cost
			h.CostCCY = st.CostCCY
			h.Realized = st.Realized
			h.RealizedCCY = st.RealizedCCY
			h.ClosedCost = st.ClosedCost
			h.ClosedCostCCY = st.ClosedCostCCY
			h.Proceeds = st.Proceeds
			h.ProceedsCCY = st.ProceedsCCY
			h.Dividends = st.Dividends
			h.DividendsCCY = st.DividendsCCY
			h.AvgCost = st.Avg
		}
	}
	return base, nil
}
