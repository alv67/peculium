package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/alv67/peculium/internal/cache"
	"github.com/alv67/peculium/internal/model"
	"github.com/alv67/peculium/internal/price"
	"github.com/alv67/peculium/internal/repository"
)

func newExecService(jr *fakeJobRepo, yf yahooFetcher, etf price.ETFFetcher, a *fakeAssetRepo) *Service {
	repos := &repository.Repository{
		Asset:     a,
		Portfolio: &fakePortfolioRepo{},
		Exposure:  &fakeExposureRepo{},
		FX:        &fakeFXRepo{},
		Lookup:    &fakeLookupRepo{},
		Job:       jr,
	}
	return New(repos, nil, yf, etf, time.Minute, time.Hour, cache.New(nil), 0, 0, nil, nil)
}

// issuesFetcher overrides both stale-refresh seams to return a canned report
// carrying issues.
type issuesFetcher struct {
	*fakeYahooFetcher
	report price.RefreshReport
}

func (f *issuesFetcher) RefreshStale(ctx context.Context, assets []*model.Asset) (price.RefreshReport, error) {
	return f.report, nil
}

func (f *issuesFetcher) RefreshStaleForPortfolio(ctx context.Context, portfolioID uuid.UUID) (price.RefreshReport, error) {
	return f.report, nil
}

type failingProfile struct{ *fakeYahooFetcher }

func (f *failingProfile) FetchAssetProfile(ctx context.Context, ticker string) (string, string, string, error) {
	return "", "", "", errors.New("yahoo down")
}

func TestExecPriceRefreshStatusAndProgress(t *testing.T) {
	ctx := context.Background()

	clean := &fakeJobRepo{}
	job, _ := clean.Enqueue(ctx, &model.Job{Type: model.JobTypePriceRefresh, TargetType: model.JobTargetGlobal})
	svc := newExecService(clean, &fakeYahooFetcher{}, nil, &fakeAssetRepo{})
	status, err := svc.execPriceRefresh(ctx, job)
	if err != nil || status != model.JobStatusDone {
		t.Fatalf("clean refresh = (%q, %v), want done", status, err)
	}

	issue := uuid.New()
	dirty := &fakeJobRepo{}
	djob, _ := dirty.Enqueue(ctx, &model.Job{Type: model.JobTypePriceRefresh, TargetType: model.JobTargetPortfolio, TargetID: &issue})
	rep := price.RefreshReport{Refreshed: []string{"AAPL"}, Issues: []price.FetchIssue{{Code: "rate_limited", Message: "429"}}}
	psvc := newExecService(dirty, &issuesFetcher{fakeYahooFetcher: &fakeYahooFetcher{}, report: rep}, nil, &fakeAssetRepo{})
	status, err = psvc.execPriceRefresh(ctx, djob)
	if err != nil || status != model.JobStatusPartial {
		t.Fatalf("issue-laden refresh = (%q, %v), want partial", status, err)
	}
	if d := dirty.find(djob.ID); d == nil || d.Processed != 2 || d.Total != 2 {
		t.Fatalf("progress = %+v, want 2/2", d)
	}
}

func TestExecHistoryBackfillRunsHistoryAndSplits(t *testing.T) {
	ctx := context.Background()
	asset := &model.Asset{ID: uuid.New(), Ticker: "AAPL", PriceSource: "yahoo"}
	jr := &fakeJobRepo{}
	job, _ := jr.Enqueue(ctx, &model.Job{Type: model.JobTypeHistoryBackfill, TargetType: model.JobTargetAsset, TargetID: &asset.ID})

	yf := &fakeYahooFetcher{}
	svc := newExecService(jr, yf, nil, &fakeAssetRepo{asset: asset})
	status, err := svc.execHistoryBackfill(ctx, job)
	if err != nil || status != model.JobStatusDone {
		t.Fatalf("status = (%q, %v), want done", status, err)
	}
	if len(yf.historyTickers) != 1 || yf.historyTickers[0] != "AAPL" {
		t.Fatalf("history tickers = %v, want [AAPL]", yf.historyTickers)
	}
	if len(yf.splitTickers) != 1 || yf.splitTickers[0] != "AAPL" {
		t.Fatalf("split tickers = %v, want [AAPL] (splits ride on the history job)", yf.splitTickers)
	}
	if d := jr.find(job.ID); d == nil || d.Processed != 2 || d.Total != 2 {
		t.Fatalf("progress = %+v, want 2/2", d)
	}
}

func TestExecHistoryBackfillNeedsTarget(t *testing.T) {
	svc := newExecService(&fakeJobRepo{}, &fakeYahooFetcher{}, nil, &fakeAssetRepo{})
	if _, err := svc.execHistoryBackfill(context.Background(), &model.Job{ID: uuid.New()}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("targetless job err = %v, want ErrInvalidInput", err)
	}
}

func TestExecMetaBackfillReportsProgressAndPartial(t *testing.T) {
	ctx := context.Background()
	stocks := []*model.Asset{
		{ID: uuid.New(), Ticker: "AAPL", Type: model.AssetTypeStock},
		{ID: uuid.New(), Ticker: "MSFT", Type: model.AssetTypeStock},
	}

	jr := &fakeJobRepo{}
	job, _ := jr.Enqueue(ctx, &model.Job{Type: model.JobTypeMetaBackfill, TargetType: model.JobTargetGlobal})
	svc := newExecService(jr, &fakeYahooFetcher{country: "United States"}, nil, &fakeAssetRepo{stocks: stocks})
	status, err := svc.execMetaBackfill(ctx, job)
	if err != nil || status != model.JobStatusDone {
		t.Fatalf("status = (%q, %v), want done", status, err)
	}
	if d := jr.find(job.ID); d == nil || d.Processed != 2 || d.Total != 2 {
		t.Fatalf("progress = %+v, want 2/2 walked by the callback", d)
	}

	fj := &fakeJobRepo{}
	fjob, _ := fj.Enqueue(ctx, &model.Job{Type: model.JobTypeMetaBackfill, TargetType: model.JobTargetGlobal})
	fsvc := newExecService(fj, &failingProfile{fakeYahooFetcher: &fakeYahooFetcher{}}, nil, &fakeAssetRepo{stocks: stocks})
	status, err = fsvc.execMetaBackfill(ctx, fjob)
	if err != nil || status != model.JobStatusPartial {
		t.Fatalf("failing profiles = (%q, %v), want partial", status, err)
	}
}

func TestRegisterJobExecutorsRunsThroughRunner(t *testing.T) {
	ctx := context.Background()
	jr := &fakeJobRepo{}
	meta := &model.Asset{ID: uuid.New(), Ticker: "AAPL", Type: model.AssetTypeStock}
	job, _ := jr.Enqueue(ctx, &model.Job{Type: model.JobTypeMetaBackfill, TargetType: model.JobTargetGlobal})

	svc := newExecService(jr, &fakeYahooFetcher{}, nil, &fakeAssetRepo{stocks: []*model.Asset{meta}})
	runner := NewJobRunner(jr, time.Minute)
	svc.RegisterJobExecutors(runner)

	if ran, err := runner.RunOnce(ctx); !ran || err != nil {
		t.Fatalf("RunOnce = (%v, %v)", ran, err)
	}
	if d := jr.find(job.ID); d == nil || d.Status != model.JobStatusDone || d.Error != "" {
		t.Fatalf("job after drain = %+v, want done with no error", d)
	}
}

func TestRecordEventStampsJobIDFromContext(t *testing.T) {
	repo := &fakeHealthRepo{}
	hs := NewHealthService(&repository.Repository{Health: repo})
	ctx := context.Background()

	jobID := uuid.New()
	if err := hs.RecordEvent(WithJobID(ctx, jobID), &model.HealthEvent{EventType: "price_refresh", Status: "success"}); err != nil {
		t.Fatal(err)
	}
	if err := hs.RecordEvent(ctx, &model.HealthEvent{EventType: "price_refresh", Status: "success"}); err != nil {
		t.Fatal(err)
	}

	if got := repo.recorded[0].JobID; got == nil || *got != jobID {
		t.Fatalf("job-tagged event id = %v, want %v", got, jobID)
	}
	if repo.recorded[1].JobID != nil {
		t.Fatal("request-path event must keep a NULL job_id")
	}
}
