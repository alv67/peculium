package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/alv67/peculium/internal/model"
	"github.com/alv67/peculium/internal/series"
	"github.com/shopspring/decimal"
)

// manualPriceLayout is the wire format for manual price dates (YYYY-MM-DD).
const manualPriceLayout = "2006-01-02"

// manualPriceMaxClose bounds absurd close values; 10^12 is far above any
// realistic per-unit price and catches exponent typos.
var manualPriceMaxClose = decimal.New(1, 12)

func toManualPrice(p *model.Price) model.ManualPrice {
	return model.ManualPrice{
		Date:   p.Date.UTC().Format(manualPriceLayout),
		Close:  p.Close,
		Volume: p.Volume,
	}
}

// ManualPrices returns the stored manual price bars of an asset, newest date
// first. Only rows with source "manual" are ever returned, so Yahoo bars can
// not leak into (or be edited through) this API. Missing assets yield
// ErrAssetNotFound; assets without manual bars yield an empty (non-nil) slice.
func (s *Service) ManualPrices(ctx context.Context, assetID uuid.UUID) ([]model.ManualPrice, error) {
	if _, err := s.repos.Asset.FindByID(ctx, assetID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAssetNotFound
		}
		return nil, err
	}
	rows, err := s.repos.Price.FindManualByAssets(ctx, []uuid.UUID{assetID})
	if err != nil {
		return nil, err
	}
	out := make([]model.ManualPrice, 0, len(rows))
	for i := len(rows) - 1; i >= 0; i-- {
		out = append(out, toManualPrice(rows[i]))
	}
	return out, nil
}

// PutManualPrice upserts one user-entered close bar on (asset_id, date): the
// same date overwrites, which is the edit path. Manual bars are stored only
// for assets whose price_source is manual/none — Yahoo-priced assets are
// rejected — so the Yahoo sync can never overwrite user input and never
// fetches these assets anyway (see isYahooPriced). Open/high/low mirror the
// close so the bar is self-consistent for the series engine.
func (s *Service) PutManualPrice(ctx context.Context, assetID uuid.UUID, in *model.ManualPriceInput) (*model.ManualPrice, error) {
	asset, err := s.repos.Asset.FindByID(ctx, assetID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAssetNotFound
		}
		return nil, err
	}
	if isYahooPriced(asset) {
		return nil, fmt.Errorf("%w: manual prices are only allowed for assets with price_source manual or none", ErrInvalidInput)
	}

	day, err := time.Parse(manualPriceLayout, in.Date)
	if err != nil {
		return nil, fmt.Errorf("%w: date must be YYYY-MM-DD", ErrInvalidInput)
	}
	if day.After(series.DayOf(time.Now())) {
		return nil, fmt.Errorf("%w: date is in the future", ErrInvalidInput)
	}
	if in.Close.LessThanOrEqual(decimal.Zero) || in.Close.GreaterThan(manualPriceMaxClose) {
		return nil, fmt.Errorf("%w: close must be greater than 0 and below 1e12", ErrInvalidInput)
	}
	if in.Volume < 0 {
		return nil, fmt.Errorf("%w: volume cannot be negative", ErrInvalidInput)
	}

	saved, err := s.repos.Price.Create(ctx, &model.Price{
		AssetID: assetID,
		Date:    day,
		Open:    in.Close,
		High:    in.Close,
		Low:     in.Close,
		Close:   in.Close,
		Volume:  in.Volume,
		Source:  "manual",
	})
	if err != nil {
		return nil, err
	}
	// Cached price reads must see the new bar right away.
	s.bumpRev(ctx)
	out := toManualPrice(saved)
	return &out, nil
}

// DeleteManualPrice removes one manual bar by date. Unknown assets yield
// ErrAssetNotFound, malformed dates ErrInvalidInput and missing bars
// ErrNotFound; Yahoo rows are never matched by the delete.
func (s *Service) DeleteManualPrice(ctx context.Context, assetID uuid.UUID, date string) error {
	if _, err := s.repos.Asset.FindByID(ctx, assetID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrAssetNotFound
		}
		return err
	}
	day, err := time.Parse(manualPriceLayout, date)
	if err != nil {
		return fmt.Errorf("%w: date must be YYYY-MM-DD", ErrInvalidInput)
	}
	deleted, err := s.repos.Price.DeleteManual(ctx, assetID, day)
	if err != nil {
		return err
	}
	if deleted == 0 {
		return ErrNotFound
	}
	s.bumpRev(ctx)
	return nil
}
