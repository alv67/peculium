package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/alv67/peculium/internal/model"
)

type SplitRepository interface {
	Upsert(ctx context.Context, s *model.Split) error
	FindByAssets(ctx context.Context, assetIDs []uuid.UUID) ([]*model.Split, error)
}

type splitRepo struct {
	db DBTX
}

func (r *splitRepo) Upsert(ctx context.Context, s *model.Split) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO splits (asset_id, date, numerator, denominator, source)
		 VALUES ($1, $2, $3, $4, 'yahoo')
		 ON CONFLICT (asset_id, date) DO UPDATE
		   SET numerator = EXCLUDED.numerator, denominator = EXCLUDED.denominator, source = 'yahoo'`,
		s.AssetID, s.Date, s.Numerator, s.Denominator,
	)
	return err
}

func (r *splitRepo) FindByAssets(ctx context.Context, assetIDs []uuid.UUID) ([]*model.Split, error) {
	if len(assetIDs) == 0 {
		return []*model.Split{}, nil
	}
	rows, err := r.db.Query(ctx,
		`SELECT asset_id, date, numerator, denominator FROM splits WHERE asset_id = ANY($1::uuid[]) ORDER BY date ASC`,
		assetIDs,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	splits := make([]*model.Split, 0)
	for rows.Next() {
		s := &model.Split{}
		if err := rows.Scan(&s.AssetID, &s.Date, &s.Numerator, &s.Denominator); err != nil {
			return nil, err
		}
		splits = append(splits, s)
	}
	return splits, rows.Err()
}
