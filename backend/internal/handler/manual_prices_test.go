package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/alv67/peculium/internal/auth"
	"github.com/alv67/peculium/internal/cache"
	"github.com/alv67/peculium/internal/model"
	"github.com/alv67/peculium/internal/repository"
	"github.com/alv67/peculium/internal/service"
)

// manualTestAssetRepo is the minimal AssetRepository for the manual-price
// routes: FindByID serves one canned asset, anything else is unknown.
type manualTestAssetRepo struct {
	repository.AssetRepository
	asset *model.Asset
}

func (f *manualTestAssetRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.Asset, error) {
	if f.asset == nil || f.asset.ID != id {
		return nil, pgx.ErrNoRows
	}
	return f.asset, nil
}

// manualTestPriceRepo implements the manual-price subset of
// repository.PriceRepository, upserting on (asset_id, date) and filtering
// source='manual' like the real queries.
type manualTestPriceRepo struct {
	repository.PriceRepository
	rows []*model.Price
}

func (f *manualTestPriceRepo) Create(ctx context.Context, price *model.Price) (*model.Price, error) {
	for _, p := range f.rows {
		if p.AssetID == price.AssetID && p.Date.Equal(price.Date) {
			p.Open, p.High, p.Low, p.Close, p.Volume, p.Source = price.Open, price.High, price.Low, price.Close, price.Volume, price.Source
			return p, nil
		}
	}
	stored := *price
	f.rows = append(f.rows, &stored)
	return &stored, nil
}

func (f *manualTestPriceRepo) FindManualByAssets(ctx context.Context, assetIDs []uuid.UUID) ([]*model.Price, error) {
	var out []*model.Price
	for _, p := range f.rows {
		if p.Source != "manual" {
			continue
		}
		for _, id := range assetIDs {
			if p.AssetID == id {
				out = append(out, p)
				break
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Date.Before(out[j].Date) })
	return out, nil
}

func (f *manualTestPriceRepo) DeleteManual(ctx context.Context, assetID uuid.UUID, date time.Time) (int64, error) {
	var kept []*model.Price
	deleted := int64(0)
	for _, p := range f.rows {
		if p.AssetID == assetID && p.Date.Equal(date) && p.Source == "manual" {
			deleted++
			continue
		}
		kept = append(kept, p)
	}
	f.rows = kept
	return deleted, nil
}

func newManualPriceTestHandler(source string) (*Handler, uuid.UUID, *manualTestPriceRepo) {
	asset := &model.Asset{ID: uuid.New(), Ticker: "BND", PriceSource: source}
	fp := &manualTestPriceRepo{}
	svc := service.New(&repository.Repository{Asset: &manualTestAssetRepo{asset: asset}, Price: fp},
		nil, nil, nil, time.Minute, time.Hour, cache.New(nil), 0, 0, nil, nil)
	return New(svc, nil), asset.ID, fp
}

// manualPriceRequest builds an authenticated request with a JSON body and chi
// URL params, extending authRequest for the routes that read a body.
func manualPriceRequest(method, target, body string, params map[string]string) *http.Request {
	r := httptest.NewRequest(method, target, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	rc := chi.NewRouteContext()
	for k, v := range params {
		rc.URLParams.Add(k, v)
	}
	ctx := context.WithValue(r.Context(), chi.RouteCtxKey, rc)
	claims := &auth.Claims{UserID: uuid.New(), Email: "user@example.com", Role: "user"}
	return r.WithContext(context.WithValue(ctx, auth.UserContextKey, claims))
}

func TestUpsertManualPrice_ResponseShape(t *testing.T) {
	h, assetID, _ := newManualPriceTestHandler("manual")
	rec := httptest.NewRecorder()
	h.UpsertManualPrice(rec, manualPriceRequest(http.MethodPut, "/assets/"+assetID.String()+"/prices/manual",
		`{"date":"2026-01-15","close":103.2}`, map[string]string{"id": assetID.String()}))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
	// The PUT body decodes close from a JSON number; the response must be the
	// stored row in the contract shape: YYYY-MM-DD date, quoted decimal close.
	var row map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &row); err != nil {
		t.Fatal(err)
	}
	if len(row) != 3 || string(row["date"]) != `"2026-01-15"` || string(row["close"]) != `"103.2"` || string(row["volume"]) != `0` {
		t.Fatalf("row = %s, want {date:\"2026-01-15\", close:\"103.2\", volume:0}", rec.Body.String())
	}

	rec = httptest.NewRecorder()
	h.ListManualPrices(rec, authRequest(http.MethodGet, "/assets/"+assetID.String()+"/prices/manual", map[string]string{"id": assetID.String()}))
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d, want 200", rec.Code)
	}
	var rows []map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || string(rows[0]["date"]) != `"2026-01-15"` {
		t.Fatalf("list = %s, want the one manual bar", rec.Body.String())
	}
}

func TestUpsertManualPrice_Errors(t *testing.T) {
	cases := []struct {
		name string
		src  string
		body string
		want int
	}{
		{"future date", "manual", `{"date":"2099-01-01","close":"1"}`, http.StatusBadRequest},
		{"zero close", "manual", `{"date":"2026-01-15","close":"0"}`, http.StatusBadRequest},
		{"negative volume", "manual", `{"date":"2026-01-15","close":"1","volume":-2}`, http.StatusBadRequest},
		{"malformed date", "manual", `{"date":"15-01-2026","close":"1"}`, http.StatusBadRequest},
		{"yahoo asset", "yahoo", `{"date":"2026-01-15","close":"1"}`, http.StatusBadRequest},
		{"legacy empty source", "", `{"date":"2026-01-15","close":"1"}`, http.StatusBadRequest},
		{"bad json", "manual", `not json`, http.StatusBadRequest},
	}
	for _, tc := range cases {
		h, assetID, _ := newManualPriceTestHandler(tc.src)
		rec := httptest.NewRecorder()
		h.UpsertManualPrice(rec, manualPriceRequest(http.MethodPut, "/assets/"+assetID.String()+"/prices/manual",
			tc.body, map[string]string{"id": assetID.String()}))
		if rec.Code != tc.want {
			t.Fatalf("%s: status = %d, want %d (body %q)", tc.name, rec.Code, tc.want, rec.Body.String())
		}
	}

	// Unknown asset: 404 on PUT and GET.
	h, _, _ := newManualPriceTestHandler("manual")
	rec := httptest.NewRecorder()
	h.UpsertManualPrice(rec, manualPriceRequest(http.MethodPut, "/prices/manual",
		`{"date":"2026-01-15","close":"1"}`, map[string]string{"id": uuid.New().String()}))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown asset PUT status = %d, want 404", rec.Code)
	}
}

func TestManualPricesFlow_ListOverwriteAndDelete(t *testing.T) {
	h, assetID, fp := newManualPriceTestHandler("none")
	params := map[string]string{"id": assetID.String()}
	get := func() *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		h.ListManualPrices(rec, authRequest(http.MethodGet, "/prices/manual", params))
		return rec
	}

	// A stored Yahoo bar is invisible to the manual API.
	fp.rows = append(fp.rows, &model.Price{
		AssetID: assetID, Date: time.Date(2026, 1, 17, 0, 0, 0, 0, time.UTC),
		Close: decimal.RequireFromString("200"), Source: "yahoo",
	})

	rec := httptest.NewRecorder()
	h.UpsertManualPrice(rec, manualPriceRequest(http.MethodPut, "/prices/manual", `{"date":"2026-01-15","close":"1.5"}`, params))
	if rec.Code != http.StatusOK {
		t.Fatalf("put status = %d, want 200", rec.Code)
	}
	// PUTs on later dates are unasserted on status (the reused recorder would
	// swallow it); their effect is verified through the GET below.
	h.UpsertManualPrice(rec, manualPriceRequest(http.MethodPut, "/prices/manual", `{"date":"2026-01-16","close":"2"}`, params))
	h.UpsertManualPrice(rec, manualPriceRequest(http.MethodPut, "/prices/manual", `{"date":"2026-01-16","close":"2.5"}`, params))

	rec = get()
	var rows []model.ManualPrice
	if err := json.Unmarshal(rec.Body.Bytes(), &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0].Date != "2026-01-16" || rows[0].Close.String() != "2.5" || rows[1].Date != "2026-01-15" {
		t.Fatalf("list = %v, want newest first with the overwritten close", rows)
	}

	del := func(date string) int {
		rec := httptest.NewRecorder()
		h.DeleteManualPrice(rec, manualPriceRequest(http.MethodDelete, "/prices/manual/"+date, "",
			map[string]string{"id": params["id"], "date": date}))
		return rec.Code
	}
	if code := del("2026-01-16"); code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204", code)
	}
	if code := del("2026-01-16"); code != http.StatusNotFound {
		t.Fatalf("repeated delete status = %d, want 404", code)
	}
	// A Yahoo bar is unreachable: its date yields 404, not a silent delete.
	if code := del("2026-01-17"); code != http.StatusNotFound {
		t.Fatalf("delete yahoo bar status = %d, want 404", code)
	}
	if len(fp.rows) != 2 || fp.rows[0].Source != "yahoo" {
		t.Fatalf("rows after deletes = %+v, want the yahoo bar and 2026-01-15 left", fp.rows)
	}
}
