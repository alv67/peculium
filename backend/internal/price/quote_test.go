package price

import (
	"context"
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

// quotePriceRepo records created prices for the batch-fetch assertions.
type quotePriceRepo struct {
	repository.PriceRepository
	created []*model.Price
}

func (p *quotePriceRepo) Create(ctx context.Context, pr *model.Price) (*model.Price, error) {
	p.created = append(p.created, pr)
	return pr, nil
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
