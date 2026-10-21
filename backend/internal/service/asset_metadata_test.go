package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/alv67/peculium/internal/cache"
	"github.com/alv67/peculium/internal/model"
	"github.com/alv67/peculium/internal/repository"
)

// newAssetMetaService builds a service over the given asset, with the exposure
// fake returned so the caller can assert what the save persisted.
func newAssetMetaService(asset *model.Asset) (*Service, *fakeExposureRepo) {
	ex := &fakeExposureRepo{}
	repos := &repository.Repository{
		Asset:     &fakeAssetRepo{asset: asset},
		Portfolio: &fakePortfolioRepo{},
		Exposure:  ex,
		FX:        &fakeFXRepo{},
	}
	return New(repos, nil, nil, nil, time.Minute, time.Hour, cache.New(nil), 0, 0, nil, nil), ex
}

// EPIC J.2: saving a fixed-income asset with an issuer country seeds that
// country as its whole geography — once, and never over stored weights.
// Equities have no issuer geography and are left alone.
func TestUpdateAssetSeedsIssuerCountry(t *testing.T) {
	id := uuid.New()
	issuer := "IT"
	patch := &model.AssetPatch{IssuerCountry: &issuer}

	run := func(assetType model.AssetType, stored []model.ExposureRow) *fakeExposureRepo {
		t.Helper()
		svc, ex := newAssetMetaService(&model.Asset{ID: id, Type: assetType, Currency: "EUR"})
		if len(stored) > 0 {
			ex.countries = map[string][]model.ExposureRow{id.String(): stored}
		}
		if _, err := svc.UpdateAsset(context.Background(), id, patch); err != nil {
			t.Fatalf("UpdateAsset(%s): %v", assetType, err)
		}
		return ex
	}

	ex := run(model.AssetTypeBond, nil)
	rows := ex.countries[id.String()]
	if len(rows) != 1 || rows[0].Name != "IT" || !rows[0].Weight.Equal(decimal.NewFromInt(100)) {
		t.Fatalf("bond exposure = %+v, want one IT 100%% row", rows)
	}
	if ex.setProvenanceCalls != 1 {
		t.Fatalf("provenance writes = %d, want 1", ex.setProvenanceCalls)
	}

	// A stored breakdown wins: the seed must not overwrite it.
	stored := []model.ExposureRow{{Name: "DE", Weight: decimal.NewFromInt(60)}}
	ex = run(model.AssetTypeCertificate, stored)
	if rows := ex.countries[id.String()]; len(rows) != 1 || rows[0].Name != "DE" {
		t.Fatalf("certificate exposure = %+v, want the stored DE row kept", rows)
	}

	ex = run(model.AssetTypeStock, nil)
	if rows := ex.countries[id.String()]; len(rows) != 0 {
		t.Fatalf("stock exposure = %+v, want none", rows)
	}
}

func TestUpdateAssetRefreshesOnlyIssuerDerivedCountry(t *testing.T) {
	id := uuid.New()
	asset := &model.Asset{ID: id, Type: model.AssetTypeBond, IssuerCountry: "IT"}
	svc, ex := newAssetMetaService(asset)
	if _, err := svc.UpdateAsset(context.Background(), id, &model.AssetPatch{}); err != nil {
		t.Fatal(err)
	}
	set := "DE"
	if _, err := svc.UpdateAsset(context.Background(), id, &model.AssetPatch{IssuerCountry: &set}); err != nil {
		t.Fatal(err)
	}
	rows := ex.countries[id.String()]
	if len(rows) != 1 || rows[0].Name != "DE" {
		t.Fatalf("issuer-derived exposure = %+v, want DE", rows)
	}

	// Once the exposure has been edited manually, future issuer changes must
	// preserve that user-owned split.
	ex.provenance[id.String()][model.ExposureDimensionCountries] = model.ExposureProvenance{Source: "manual"}
	ex.countries[id.String()] = []model.ExposureRow{{Name: "FR", Weight: decimal.NewFromInt(100)}}
	set = "ES"
	if _, err := svc.UpdateAsset(context.Background(), id, &model.AssetPatch{IssuerCountry: &set}); err != nil {
		t.Fatal(err)
	}
	rows = ex.countries[id.String()]
	if len(rows) != 1 || rows[0].Name != "FR" {
		t.Fatalf("manual exposure = %+v, want FR preserved", rows)
	}
}

// The maturity date travels as YYYY-MM-DD: a valid date is stored, an empty
// string clears it and anything else is a 400, not a 500.
func TestUpdateAssetMaturityDate(t *testing.T) {
	id := uuid.New()
	svc, _ := newAssetMetaService(&model.Asset{ID: id, Type: model.AssetTypeBond})
	ctx := context.Background()

	set := "2032-05-15"
	asset, err := svc.UpdateAsset(ctx, id, &model.AssetPatch{MaturityDate: &set})
	if err != nil {
		t.Fatalf("set: %v", err)
	}
	if asset.MaturityDate == nil || !asset.MaturityDate.Equal(time.Date(2032, 5, 15, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("maturity = %v, want 2032-05-15", asset.MaturityDate)
	}

	empty := ""
	asset, err = svc.UpdateAsset(ctx, id, &model.AssetPatch{MaturityDate: &empty})
	if err != nil {
		t.Fatalf("clear: %v", err)
	}
	if asset.MaturityDate != nil {
		t.Fatalf("maturity = %v, want nil after clearing", asset.MaturityDate)
	}

	bad := "15/05/2032"
	if _, err = svc.UpdateAsset(ctx, id, &model.AssetPatch{MaturityDate: &bad}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("err = %v, want ErrInvalidInput", err)
	}
}
