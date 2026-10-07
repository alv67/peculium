package price

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/alv67/peculium/internal/model"
	"github.com/alv67/peculium/internal/repository"
)

// quotePriceRepo records created prices and serves them back as the latest
// close, so the post-refresh staleness check sees what was just written.
type quotePriceRepo struct {
	repository.PriceRepository
	created []*model.Price
}

func (p *quotePriceRepo) Create(ctx context.Context, pr *model.Price) (*model.Price, error) {
	p.created = append(p.created, pr)
	return pr, nil
}

func (p *quotePriceRepo) FindLatestForAssets(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*model.Price, error) {
	out := make(map[uuid.UUID]*model.Price, len(p.created))
	for _, pr := range p.created {
		out[pr.AssetID] = pr
	}
	return out, nil
}

type markFetchedAssetRepo struct {
	repository.AssetRepository
}

func (a *markFetchedAssetRepo) MarkPricesFetched(ctx context.Context, ids []uuid.UUID, at time.Time) error {
	return nil
}

func sparkClient(body string) *http.Client {
	return &http.Client{Transport: rtFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		}, nil
	})}
}

// The spark close series ends with a null (the not-yet-final current
// session): the last non-null close must be stored, and a symbol missing
// from the batch must be reported as an issue, never fetched from the chart.
func TestFetchQuotesBatchUsesLastNonNullCloseAndReportsMissing(t *testing.T) {
	pr := &quotePriceRepo{}
	f := NewYahooFetcher(&repository.Repository{Price: pr}, time.Hour, WithMinInterval(0))
	f.client = sparkClient(`{"AAPL":{"timestamp":[1700000000,1700086400,1700172800],"close":[1.5,2.5,null],"symbol":"AAPL"}}`)

	assets := []*model.Asset{{ID: uuid.New(), Ticker: "AAPL"}, {ID: uuid.New(), Ticker: "MISS"}}
	refreshed, issues := f.fetchQuotesBatch(context.Background(), assets)

	if len(refreshed) != 1 || refreshed[0] != "AAPL" {
		t.Fatalf("refreshed = %v, want [AAPL]", refreshed)
	}
	if len(pr.created) != 1 {
		t.Fatalf("created = %d, want 1", len(pr.created))
	}
	if !pr.created[0].Close.Equal(decimal.NewFromFloat(2.5)) {
		t.Fatalf("close = %s, want 2.5", pr.created[0].Close)
	}
	if !pr.created[0].Date.Equal(priceDate(1700086400)) {
		t.Fatalf("date = %v, want %v", pr.created[0].Date, priceDate(1700086400))
	}
	foundMissing := false
	for _, iss := range issues {
		if iss.Symbol == "MISS" {
			foundMissing = true
		}
	}
	if !foundMissing {
		t.Fatalf("missing symbol not reported: %+v", issues)
	}
}

// A refresh that stores a bar older than the expected trading day must leave
// a price_refresh/stale event, so a missing current close is not hidden
// behind the batch success summary.
func TestRefreshStaleRecordsStaleWhenCloseIsBehind(t *testing.T) {
	id := uuid.New()
	behind := time.Now().UTC().AddDate(0, 0, -3).Truncate(24 * time.Hour)
	pr := &quotePriceRepo{}
	rec := &recStub{}
	f := NewYahooFetcher(&repository.Repository{Price: pr, Asset: &markFetchedAssetRepo{}}, time.Hour,
		WithHealthRecorder(rec), WithMinInterval(0))
	f.client = sparkClient(fmt.Sprintf(`{"AAPL":{"timestamp":[%d],"close":[2.5],"symbol":"AAPL"}}`, behind.Unix()))

	assets := []*model.Asset{{ID: id, Ticker: "AAPL", PriceSource: "yahoo"}}
	if _, err := f.RefreshStale(context.Background(), assets); err != nil {
		t.Fatal(err)
	}

	var stale *model.HealthEvent
	for _, ev := range rec.events {
		if ev.EventType == "price_refresh" && ev.Code == "stale" {
			stale = ev
		}
	}
	if stale == nil {
		t.Fatalf("no stale event recorded: %+v", rec.events)
	}
	if stale.Status != "failure" {
		t.Fatalf("stale status = %q, want failure", stale.Status)
	}
}
