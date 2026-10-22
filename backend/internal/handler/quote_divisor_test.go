package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/alv67/peculium/internal/cache"
	"github.com/alv67/peculium/internal/model"
	"github.com/alv67/peculium/internal/repository"
	"github.com/alv67/peculium/internal/service"
)

// divisorTestAssetRepo serves one canned asset and persists patches onto it,
// the minimum the PATCH/GET asset routes need to exercise quote_divisor.
type divisorTestAssetRepo struct {
	repository.AssetRepository
	asset *model.Asset
}

func (f *divisorTestAssetRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.Asset, error) {
	if f.asset == nil || f.asset.ID != id {
		return nil, pgx.ErrNoRows
	}
	return f.asset, nil
}

func (f *divisorTestAssetRepo) Update(ctx context.Context, asset *model.Asset) (*model.Asset, error) {
	f.asset = asset
	return asset, nil
}

func newDivisorTestHandler() (*Handler, *model.Asset) {
	asset := &model.Asset{ID: uuid.New(), Ticker: "BND", Name: "Bond", Type: model.AssetTypeBond,
		PriceSource: "none", AssetClass: "bond", QuoteDivisor: decimal.NewFromInt(1)}
	svc := service.New(&repository.Repository{Asset: &divisorTestAssetRepo{asset: asset}},
		nil, nil, nil, time.Minute, time.Hour, cache.New(nil), 0, 0, nil, nil)
	return New(svc, nil), asset
}

func TestPatchAssetQuoteDivisor(t *testing.T) {
	patch := func(body string) *httptest.ResponseRecorder {
		h, asset := newDivisorTestHandler()
		rec := httptest.NewRecorder()
		h.UpdateAsset(rec, manualPriceRequest(http.MethodPatch, "/assets/"+asset.ID.String(), body,
			map[string]string{"id": asset.ID.String()}))
		return rec
	}

	rec := patch(`{"quote_divisor":"100"}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"quote_divisor":"100"`) {
		t.Fatalf("valid patch: status %d, body %q", rec.Code, rec.Body.String())
	}
	for _, bad := range []string{`{"quote_divisor":"0"}`, `{"quote_divisor":-5}`, `{"quote_divisor":"abc"}`, `{"quote_divisor":""}`} {
		if rec := patch(bad); rec.Code != http.StatusBadRequest {
			t.Fatalf("%s: status = %d, want 400 (body %q)", bad, rec.Code, rec.Body.String())
		}
	}

	// An absent field keeps the stored divisor.
	h, asset := newDivisorTestHandler()
	asset.QuoteDivisor = decimal.NewFromInt(100)
	rec = httptest.NewRecorder()
	h.UpdateAsset(rec, manualPriceRequest(http.MethodPatch, "/assets/"+asset.ID.String(), `{"name":"X"}`,
		map[string]string{"id": asset.ID.String()}))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"quote_divisor":"100"`) {
		t.Fatalf("unchanged divisor: status %d, body %q", rec.Code, rec.Body.String())
	}
}

func TestGetAssetQuoteDivisor(t *testing.T) {
	h, asset := newDivisorTestHandler()
	rec := httptest.NewRecorder()
	h.GetAsset(rec, authRequest(http.MethodGet, "/assets/"+asset.ID.String(), map[string]string{"id": asset.ID.String()}))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var got map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if string(got["quote_divisor"]) != `"1"` {
		t.Fatalf("quote_divisor = %s, want the quoted default \"1\"", got["quote_divisor"])
	}
}
