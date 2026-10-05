package price

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/alv67/peculium/internal/model"
	"github.com/alv67/peculium/internal/repository"
)

type recStub struct {
	events []*model.HealthEvent
}

func (r *recStub) RecordEvent(ctx context.Context, ev *model.HealthEvent) error {
	r.events = append(r.events, ev)
	return nil
}

func TestRecordHealthComputesRealDuration(t *testing.T) {
	rec := &recStub{}
	f := NewYahooFetcher(nil, time.Hour, WithHealthRecorder(rec))
	f.recordHealth(context.Background(), nil, "price_refresh", "success", "", "x", time.Now().Add(-120*time.Millisecond))
	if len(rec.events) != 1 {
		t.Fatalf("events = %d, want 1", len(rec.events))
	}
	if d := rec.events[0].DurationMs; d < 100 {
		t.Fatalf("duration_ms = %d, want >= 100", d)
	}
}

// Embedding the real interfaces keeps these fakes to just the methods
// EnsureHistory calls.
type histPriceRepo struct {
	repository.PriceRepository
	created, failed, failFrom int
}

func (p *histPriceRepo) MinMaxDate(ctx context.Context, id uuid.UUID) (*time.Time, *time.Time, error) {
	return nil, nil, nil
}

func (p *histPriceRepo) Create(ctx context.Context, pr *model.Price) (*model.Price, error) {
	p.created++
	if p.created > p.failFrom {
		p.failed++
		return nil, errors.New("db down")
	}
	return pr, nil
}

type histAssetRepo struct {
	repository.AssetRepository
	markErr error
	marks   int
}

func (a *histAssetRepo) MarkHistoryBackfilled(ctx context.Context, id uuid.UUID) error {
	a.marks++
	return a.markErr
}

type rtFunc func(*http.Request) (*http.Response, error)

func (r rtFunc) RoundTrip(req *http.Request) (*http.Response, error) { return r(req) }

// 3 daily bars from the chart endpoint.
const stubChart = `{"chart":{"result":[{"timestamp":[1700000000,1700086400,1700172800],"indicators":{"quote":[{"close":[1.5,2.5,3.5]}]}}]}}`

func TestEnsureHistoryAggregatesSaveFailures(t *testing.T) {
	pr := &histPriceRepo{failFrom: 1} // first bar saves, the next two fail
	ar := &histAssetRepo{markErr: errors.New("mark boom")}
	rec := &recStub{}
	f := NewYahooFetcher(&repository.Repository{Price: pr, Asset: ar}, time.Hour,
		WithHealthRecorder(rec), WithMinInterval(0))
	f.client = &http.Client{Transport: rtFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(stubChart)),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		}, nil
	})}

	id := uuid.New()
	err := f.EnsureHistory(context.Background(), []HistoryAsset{{ID: id, Ticker: "AAPL", From: time.Now().AddDate(0, 0, -5), Full: true}})
	if !errors.Is(err, ErrSyncIncomplete) {
		t.Fatalf("err = %v, want ErrSyncIncomplete (bars and mark both failed)", err)
	}
	if pr.failed != 2 || ar.marks != 1 {
		t.Fatalf("saves failed=%d marks=%d, want 2 and 1", pr.failed, ar.marks)
	}
	// One aggregate save event plus one mark event, never per-bar rows.
	if len(rec.events) != 2 {
		t.Fatalf("events = %d, want 2 (aggregated): %+v", len(rec.events), rec.events)
	}
	save := rec.events[0]
	if save.EventType != "history_save" || save.Status != "failure" {
		t.Fatalf("save event = %q/%q, want history_save/failure", save.EventType, save.Status)
	}
	if save.AssetID == nil || *save.AssetID != id {
		t.Fatalf("save event asset = %v, want %v", save.AssetID, id)
	}
	if !strings.Contains(save.Message, "2 of 3 bars failed to save") || !strings.Contains(save.Message, "db down") {
		t.Fatalf("save message = %q, want counts + first error", save.Message)
	}
	if save.DurationMs < 0 {
		t.Fatalf("save duration = %d, want >= 0", save.DurationMs)
	}
	mark := rec.events[1]
	if mark.EventType != "history_backfill" || !strings.Contains(mark.Message, "mark history backfilled failed") {
		t.Fatalf("mark event = %+v, want the backfill mark failure", mark)
	}
}
