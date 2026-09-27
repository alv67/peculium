package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/alv67/peculium/internal/model"
)

type ExposureRepository interface {
	FindRegions(ctx context.Context, assetID uuid.UUID) ([]model.ExposureRow, error)
	FindSectors(ctx context.Context, assetID uuid.UUID) ([]model.ExposureRow, error)
	FindCountries(ctx context.Context, assetID uuid.UUID) ([]model.ExposureRow, error)
	ReplaceRegions(ctx context.Context, assetID uuid.UUID, rows []model.ExposureRow) error
	ReplaceSectors(ctx context.Context, assetID uuid.UUID, rows []model.ExposureRow) error
	ReplaceCountries(ctx context.Context, assetID uuid.UUID, rows []model.ExposureRow) error
	FindRegionsByAssets(ctx context.Context, assetIDs []uuid.UUID) (map[string][]model.ExposureRow, error)
	FindSectorsByAssets(ctx context.Context, assetIDs []uuid.UUID) (map[string][]model.ExposureRow, error)
	FindCountriesByAssets(ctx context.Context, assetIDs []uuid.UUID) (map[string][]model.ExposureRow, error)
	FindProvenance(ctx context.Context, assetID uuid.UUID) (map[string]model.ExposureProvenance, error)
	SetProvenance(ctx context.Context, assetID uuid.UUID, dimension, source string) error
}

type exposureRepo struct {
	db DBTX
}

func (r *exposureRepo) FindRegions(ctx context.Context, assetID uuid.UUID) ([]model.ExposureRow, error) {
	rows, err := r.db.Query(ctx,
		`SELECT region, weight FROM asset_region_weights WHERE asset_id = $1 ORDER BY region`,
		assetID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]model.ExposureRow, 0)
	for rows.Next() {
		var row model.ExposureRow
		if err := rows.Scan(&row.Name, &row.Weight); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *exposureRepo) FindCountries(ctx context.Context, assetID uuid.UUID) ([]model.ExposureRow, error) {
	rows, err := r.db.Query(ctx,
		`SELECT country, weight FROM asset_country_weights WHERE asset_id = $1 ORDER BY country`,
		assetID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]model.ExposureRow, 0)
	for rows.Next() {
		var row model.ExposureRow
		if err := rows.Scan(&row.Name, &row.Weight); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *exposureRepo) FindSectors(ctx context.Context, assetID uuid.UUID) ([]model.ExposureRow, error) {
	rows, err := r.db.Query(ctx,
		`SELECT sector, weight FROM asset_sector_weights WHERE asset_id = $1 ORDER BY sector`,
		assetID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]model.ExposureRow, 0)
	for rows.Next() {
		var row model.ExposureRow
		if err := rows.Scan(&row.Name, &row.Weight); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *exposureRepo) FindRegionsByAssets(ctx context.Context, assetIDs []uuid.UUID) (map[string][]model.ExposureRow, error) {
	return r.findByAssets(ctx, assetIDs, "asset_region_weights", "region")
}

func (r *exposureRepo) FindSectorsByAssets(ctx context.Context, assetIDs []uuid.UUID) (map[string][]model.ExposureRow, error) {
	return r.findByAssets(ctx, assetIDs, "asset_sector_weights", "sector")
}

func (r *exposureRepo) FindCountriesByAssets(ctx context.Context, assetIDs []uuid.UUID) (map[string][]model.ExposureRow, error) {
	return r.findByAssets(ctx, assetIDs, "asset_country_weights", "country")
}

// findByAssets fetches the stored exposure rows for a set of assets in a single
// batch query, keyed by asset id. Rows with an empty name or a non-positive
// weight are skipped, mirroring replace's semantics. Table and column are
// internal constants, so the formatted names are safe.
func (r *exposureRepo) findByAssets(ctx context.Context, assetIDs []uuid.UUID, table, column string) (map[string][]model.ExposureRow, error) {
	if len(assetIDs) == 0 {
		return map[string][]model.ExposureRow{}, nil
	}
	query := fmt.Sprintf(`SELECT asset_id, %s, weight FROM %s WHERE asset_id = ANY($1::uuid[]) ORDER BY asset_id, %s`, column, table, column)
	rows, err := r.db.Query(ctx, query, assetIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string][]model.ExposureRow{}
	for rows.Next() {
		var assetID uuid.UUID
		var row model.ExposureRow
		if err := rows.Scan(&assetID, &row.Name, &row.Weight); err != nil {
			return nil, err
		}
		if row.Name == "" || !row.Weight.IsPositive() {
			continue
		}
		key := assetID.String()
		out[key] = append(out[key], row)
	}
	return out, rows.Err()
}

func (r *exposureRepo) ReplaceRegions(ctx context.Context, assetID uuid.UUID, rows []model.ExposureRow) error {
	return r.replace(ctx, assetID, rows, "asset_region_weights", "region")
}

func (r *exposureRepo) ReplaceSectors(ctx context.Context, assetID uuid.UUID, rows []model.ExposureRow) error {
	return r.replace(ctx, assetID, rows, "asset_sector_weights", "sector")
}

func (r *exposureRepo) ReplaceCountries(ctx context.Context, assetID uuid.UUID, rows []model.ExposureRow) error {
	return r.replace(ctx, assetID, rows, "asset_country_weights", "country")
}

// replace deletes the stored rows for an asset and inserts the given ones,
// skipping rows with an empty name or a non-positive weight. Table and column
// are internal constants, so the formatted names are safe.
func (r *exposureRepo) replace(ctx context.Context, assetID uuid.UUID, rows []model.ExposureRow, table, column string) error {
	if _, err := r.db.Exec(ctx, fmt.Sprintf(`DELETE FROM %s WHERE asset_id = $1`, table), assetID); err != nil {
		return err
	}
	for _, row := range rows {
		if row.Name == "" || !row.Weight.IsPositive() {
			continue
		}
		if _, err := r.db.Exec(ctx,
			fmt.Sprintf(`INSERT INTO %s (asset_id, %s, weight) VALUES ($1, $2, $3)`, table, column),
			assetID, row.Name, row.Weight,
		); err != nil {
			return err
		}
	}
	return nil
}

// FindProvenance returns the persisted provenance of each exposure dimension
// of an asset, keyed by dimension name ("countries", "regions", "sectors").
// Dimensions never saved are simply absent from the map.
func (r *exposureRepo) FindProvenance(ctx context.Context, assetID uuid.UUID) (map[string]model.ExposureProvenance, error) {
	rows, err := r.db.Query(ctx,
		`SELECT dimension, source, updated_at FROM asset_exposure_provenance WHERE asset_id = $1 ORDER BY dimension`,
		assetID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string]model.ExposureProvenance{}
	for rows.Next() {
		var dimension string
		var prov model.ExposureProvenance
		if err := rows.Scan(&dimension, &prov.Source, &prov.UpdatedAt); err != nil {
			return nil, err
		}
		out[dimension] = prov
	}
	return out, rows.Err()
}

// SetProvenance upserts the provenance of one exposure dimension, refreshing
// updated_at on every write.
func (r *exposureRepo) SetProvenance(ctx context.Context, assetID uuid.UUID, dimension, source string) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO asset_exposure_provenance (asset_id, dimension, source)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (asset_id, dimension) DO UPDATE SET source = EXCLUDED.source, updated_at = now()`,
		assetID, dimension, source,
	)
	return err
}
