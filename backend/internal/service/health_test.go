package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/alv67/peculium/internal/cache"
	"github.com/alv67/peculium/internal/model"
	"github.com/alv67/peculium/internal/price"
	"github.com/alv67/peculium/internal/repository"
)

type fakeHealthRepo struct {
	recorded []*model.HealthEvent

	sinceCalls []time.Time
	lastNCalls []int
	pageCalls  int
	pageLimit  int
	pageOffset int
	// pageEventType/pageStatus and countEventType/countStatus record the
	// filters forwarded by GetPriceHealth, so the filter tests can assert both
	// the page and the total are filtered consistently.
	pageEventType  string
	pageStatus     string
	countEventType string
	countStatus    string
	// pageJobID/countJobID record the job filter forwarded by GetPriceHealth,
	// so the detail-panel filter test can assert page and total stay in sync.
	pageJobID  string
	countJobID string

	// activityItems/activityTotal are the canned union rows returned by
	// GetActivityPage/CountActivity; the activity* fields record what
	// GetActivity forwarded.
	activityItems   []*model.ActivityItem
	activityTotal   int
	activityLimit   int
	activityOffset  int
	activityType    string
	activityStatus  string
	activityCalls   int
	activityCounted int

	sinceSummary *model.HealthSummary
	lastNSummary *model.HealthSummary
	count        int
	events       []*model.HealthEvent

	countsByJobs map[uuid.UUID]*model.JobHealthCounts
	countsIDs    []uuid.UUID
	// writeCtxErr records ctx.Err() as seen by the last RecordEvent, to
	// prove expired contexts are rewritten on a fresh one.
	writeCtxErr       error
	deleteBeforeCalls []time.Time
	deleteN           int64
	clearAllCalls     int
	clearN            int64
}

func (f *fakeHealthRepo) RecordEvent(ctx context.Context, event *model.HealthEvent) error {
	f.recorded = append(f.recorded, event)
	f.writeCtxErr = ctx.Err()
	return nil
}

func (f *fakeHealthRepo) CountsByJobs(ctx context.Context, jobIDs []uuid.UUID) (map[uuid.UUID]*model.JobHealthCounts, error) {
	f.countsIDs = jobIDs
	return f.countsByJobs, nil
}

func (f *fakeHealthRepo) GetEventsPage(ctx context.Context, limit, offset int, eventType, status, jobID string) ([]*model.HealthEvent, error) {
	f.pageCalls++
	f.pageLimit = limit
	f.pageOffset = offset
	f.pageEventType = eventType
	f.pageStatus = status
	f.pageJobID = jobID
	return f.events, nil
}

func (f *fakeHealthRepo) CountEvents(ctx context.Context, eventType, status, jobID string) (int, error) {
	f.countEventType = eventType
	f.countStatus = status
	f.countJobID = jobID
	return f.count, nil
}

func (f *fakeHealthRepo) GetActivityPage(ctx context.Context, limit, offset int, itemType, status string) ([]*model.ActivityItem, error) {
	f.activityCalls++
	f.activityLimit = limit
	f.activityOffset = offset
	f.activityType = itemType
	f.activityStatus = status
	return f.activityItems, nil
}

func (f *fakeHealthRepo) CountActivity(ctx context.Context, itemType, status string) (int, error) {
	f.activityCounted++
	return f.activityTotal, nil
}

func (f *fakeHealthRepo) SummarySince(ctx context.Context, since time.Time) (*model.HealthSummary, error) {
	f.sinceCalls = append(f.sinceCalls, since)
	return f.sinceSummary, nil
}

func (f *fakeHealthRepo) SummaryLastN(ctx context.Context, n int) (*model.HealthSummary, error) {
	f.lastNCalls = append(f.lastNCalls, n)
	return f.lastNSummary, nil
}

func (f *fakeHealthRepo) DeleteStandaloneBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	f.deleteBeforeCalls = append(f.deleteBeforeCalls, cutoff)
	return f.deleteN, nil
}

func (f *fakeHealthRepo) ClearAll(ctx context.Context) (int64, error) {
	f.clearAllCalls++
	return f.clearN, nil
}

var _ repository.HealthRepository = (*fakeHealthRepo)(nil)

func TestHealthWindowFor(t *testing.T) {
	now := time.Date(2026, 9, 13, 14, 30, 45, 0, time.UTC)

	tcs := []struct {
		period     string
		wantPeriod string
		wantSince  time.Time
		wantLastN  int
	}{
		{period: "", wantPeriod: "today", wantSince: time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)},
		{period: "today", wantPeriod: "today", wantSince: time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)},
		{period: "24h", wantPeriod: "24h", wantSince: now.Add(-24 * time.Hour)},
		{period: "100", wantPeriod: "100", wantLastN: 100},
		{period: "week", wantPeriod: "today", wantSince: time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)},
	}

	for _, tc := range tcs {
		t.Run("period="+tc.period, func(t *testing.T) {
			got := healthWindowFor(tc.period, now)
			if got.period != tc.wantPeriod {
				t.Fatalf("period = %q, want %q", got.period, tc.wantPeriod)
			}
			if got.lastN != tc.wantLastN {
				t.Fatalf("lastN = %d, want %d", got.lastN, tc.wantLastN)
			}
			if tc.wantLastN == 0 && !got.since.Equal(tc.wantSince) {
				t.Fatalf("since = %v, want %v", got.since, tc.wantSince)
			}
			if tc.wantLastN > 0 && !got.since.IsZero() {
				t.Fatalf("since = %v, want zero (window is count-based)", got.since)
			}
		})
	}
}

func TestGetPriceHealthTodayUsesSummarySince(t *testing.T) {
	repo := &fakeHealthRepo{
		sinceSummary: &model.HealthSummary{Successes: 7, Failures: 3, RateLimited: 1},
	}
	svc := NewHealthService(&repository.Repository{Health: repo})

	summary, events, _, err := svc.GetPriceHealth(context.Background(), "today", 0, 0, "", "", "")
	if err != nil {
		t.Fatalf("GetPriceHealth: %v", err)
	}
	if len(repo.sinceCalls) != 1 || len(repo.lastNCalls) != 0 {
		t.Fatalf("SummarySince calls = %d, SummaryLastN calls = %d; want 1 and 0", len(repo.sinceCalls), len(repo.lastNCalls))
	}
	// "today" must start at the beginning of the current UTC calendar day.
	wantSince := time.Now().UTC().Truncate(24 * time.Hour)
	if !repo.sinceCalls[0].Equal(wantSince) {
		t.Fatalf("since = %v, want %v", repo.sinceCalls[0], wantSince)
	}
	if summary.Period != "today" {
		t.Fatalf("Period = %q, want %q", summary.Period, "today")
	}
	if !summary.HasData {
		t.Fatal("HasData = false, want true")
	}
	if summary.SuccessRate != 0.7 {
		t.Fatalf("SuccessRate = %v, want 0.7", summary.SuccessRate)
	}
	if summary.RateLimited != 1 {
		t.Fatalf("RateLimited = %d, want 1", summary.RateLimited)
	}
	if repo.pageCalls != 1 || repo.pageLimit != 100 || repo.pageOffset != 0 {
		t.Fatalf("GetEventsPage calls = %d with (limit=%d, offset=%d); want 1 with (100, 0)",
			repo.pageCalls, repo.pageLimit, repo.pageOffset)
	}
	_ = events
}

func TestGetPriceHealth24hUsesSummarySince(t *testing.T) {
	repo := &fakeHealthRepo{
		sinceSummary: &model.HealthSummary{Successes: 10},
	}
	svc := NewHealthService(&repository.Repository{Health: repo})

	summary, _, _, err := svc.GetPriceHealth(context.Background(), "24h", 0, 0, "", "", "")
	if err != nil {
		t.Fatalf("GetPriceHealth: %v", err)
	}
	if len(repo.sinceCalls) != 1 {
		t.Fatalf("SummarySince calls = %d, want 1", len(repo.sinceCalls))
	}
	want := time.Now().UTC().Add(-24 * time.Hour)
	if diff := repo.sinceCalls[0].Sub(want); diff > time.Minute || diff < -time.Minute {
		t.Fatalf("since = %v, want ~%v", repo.sinceCalls[0], want)
	}
	if summary.Period != "24h" {
		t.Fatalf("Period = %q, want %q", summary.Period, "24h")
	}
	if summary.SuccessRate != 1 {
		t.Fatalf("SuccessRate = %v, want 1", summary.SuccessRate)
	}
}

func TestGetPriceHealthLastNUsesSummaryLastN(t *testing.T) {
	repo := &fakeHealthRepo{
		lastNSummary: &model.HealthSummary{Successes: 4, Failures: 6},
	}
	svc := NewHealthService(&repository.Repository{Health: repo})

	summary, _, _, err := svc.GetPriceHealth(context.Background(), "100", 0, 0, "", "", "")
	if err != nil {
		t.Fatalf("GetPriceHealth: %v", err)
	}
	if len(repo.lastNCalls) != 1 || repo.lastNCalls[0] != 100 {
		t.Fatalf("SummaryLastN calls = %v, want [100]", repo.lastNCalls)
	}
	if len(repo.sinceCalls) != 0 {
		t.Fatalf("SummarySince calls = %d, want 0", len(repo.sinceCalls))
	}
	if summary.Period != "100" {
		t.Fatalf("Period = %q, want %q", summary.Period, "100")
	}
	if summary.SuccessRate != 0.4 {
		t.Fatalf("SuccessRate = %v, want 0.4", summary.SuccessRate)
	}
}

func TestGetPriceHealthInvalidPeriodFallsBackToToday(t *testing.T) {
	repo := &fakeHealthRepo{
		sinceSummary: &model.HealthSummary{},
	}
	svc := NewHealthService(&repository.Repository{Health: repo})

	summary, _, _, err := svc.GetPriceHealth(context.Background(), "nonsense", 0, 0, "", "", "")
	if err != nil {
		t.Fatalf("GetPriceHealth: %v", err)
	}
	if summary.Period != "today" {
		t.Fatalf("Period = %q, want %q", summary.Period, "today")
	}
	if summary.HasData {
		t.Fatal("HasData = true, want false for empty counts")
	}
	if summary.SuccessRate != 0 {
		t.Fatalf("SuccessRate = %v, want 0 when there is no data", summary.SuccessRate)
	}
}

func TestHealthRecordEventPersistsOnlyToDB(t *testing.T) {
	repo := &fakeHealthRepo{}
	svc := NewHealthService(&repository.Repository{Health: repo})

	event := &model.HealthEvent{EventType: "fetch", Status: "failure", Code: "rate_limited"}
	if err := svc.RecordEvent(context.Background(), event); err != nil {
		t.Fatalf("RecordEvent: %v", err)
	}
	if len(repo.recorded) != 1 || repo.recorded[0] != event {
		t.Fatalf("recorded = %v, want the single event", repo.recorded)
	}
}

func TestNormalizeHealthPage(t *testing.T) {
	tcs := []struct {
		name       string
		limit      int
		offset     int
		wantLimit  int
		wantOffset int
	}{
		{"unset falls back to defaults", 0, 0, 100, 0},
		{"negative falls back to defaults", -5, -10, 100, 0},
		{"oversized limit is capped", 1000, 0, 500, 0},
		{"max limit is allowed", 500, 0, 500, 0},
		{"min limit is allowed", 1, 0, 1, 0},
		{"valid page is kept", 25, 75, 25, 75},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			limit, offset := normalizeHealthPage(tc.limit, tc.offset)
			if limit != tc.wantLimit || offset != tc.wantOffset {
				t.Fatalf("normalizeHealthPage(%d, %d) = (%d, %d), want (%d, %d)",
					tc.limit, tc.offset, limit, offset, tc.wantLimit, tc.wantOffset)
			}
		})
	}
}

func TestGetPriceHealthPassesNormalizedPaginationAndReturnsTotal(t *testing.T) {
	repo := &fakeHealthRepo{
		sinceSummary: &model.HealthSummary{Successes: 120, Failures: 17},
		count:        137,
		events:       []*model.HealthEvent{{EventType: "price_sync", Status: "success"}},
	}
	svc := NewHealthService(&repository.Repository{Health: repo})

	_, events, total, err := svc.GetPriceHealth(context.Background(), "today", 10000, -5, "", "", "")
	if err != nil {
		t.Fatalf("GetPriceHealth: %v", err)
	}
	if repo.pageCalls != 1 {
		t.Fatalf("GetEventsPage calls = %d, want 1", repo.pageCalls)
	}
	if repo.pageLimit != 500 || repo.pageOffset != 0 {
		t.Fatalf("GetEventsPage(limit=%d, offset=%d), want (500, 0)", repo.pageLimit, repo.pageOffset)
	}
	if total != 137 {
		t.Fatalf("total = %d, want 137", total)
	}
	if len(events) != 1 {
		t.Fatalf("events = %d, want 1", len(events))
	}
}

func TestGetPriceHealthUsesDefaultPageWhenUnset(t *testing.T) {
	repo := &fakeHealthRepo{
		sinceSummary: &model.HealthSummary{Successes: 1},
		count:        42,
	}
	svc := NewHealthService(&repository.Repository{Health: repo})

	_, _, total, err := svc.GetPriceHealth(context.Background(), "today", 0, 0, "", "", "")
	if err != nil {
		t.Fatalf("GetPriceHealth: %v", err)
	}
	if repo.pageLimit != 100 || repo.pageOffset != 0 {
		t.Fatalf("GetEventsPage(limit=%d, offset=%d), want (100, 0)", repo.pageLimit, repo.pageOffset)
	}
	if total != 42 {
		t.Fatalf("total = %d, want 42", total)
	}
}

func TestGetPriceHealthKeepsValidPagination(t *testing.T) {
	repo := &fakeHealthRepo{
		sinceSummary: &model.HealthSummary{Successes: 1},
		count:        300,
	}
	svc := NewHealthService(&repository.Repository{Health: repo})

	_, _, _, err := svc.GetPriceHealth(context.Background(), "today", 25, 50, "", "", "")
	if err != nil {
		t.Fatalf("GetPriceHealth: %v", err)
	}
	if repo.pageLimit != 25 || repo.pageOffset != 50 {
		t.Fatalf("GetEventsPage(limit=%d, offset=%d), want (25, 50)", repo.pageLimit, repo.pageOffset)
	}
}

func TestRecordEventRewritesExpiredContext(t *testing.T) {
	repo := &fakeHealthRepo{}
	hs := NewHealthService(&repository.Repository{Health: repo})

	dead, cancel := context.WithCancel(context.Background())
	cancel()
	if err := hs.RecordEvent(dead, &model.HealthEvent{EventType: "history_fetch", Status: "failure"}); err != nil {
		t.Fatalf("RecordEvent on a dead context: %v", err)
	}
	if len(repo.recorded) != 1 {
		t.Fatalf("events recorded = %d, want 1", len(repo.recorded))
	}
	if repo.writeCtxErr != nil {
		t.Fatalf("the repo write saw a dead context: %v", repo.writeCtxErr)
	}
}

func TestPruneStandaloneEventsForwardsCutoffAndReturnsCount(t *testing.T) {
	repo := &fakeHealthRepo{deleteN: 7}
	svc := NewHealthService(&repository.Repository{Health: repo})

	cutoff := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	n, err := svc.PruneStandaloneEvents(context.Background(), cutoff)
	if err != nil {
		t.Fatalf("PruneStandaloneEvents: %v", err)
	}
	if n != 7 {
		t.Fatalf("deleted = %d, want 7", n)
	}
	if len(repo.deleteBeforeCalls) != 1 || !repo.deleteBeforeCalls[0].Equal(cutoff) {
		t.Fatalf("DeleteStandaloneBefore cutoffs = %v, want [%v]", repo.deleteBeforeCalls, cutoff)
	}
}

func TestClearAllReturnsCount(t *testing.T) {
	repo := &fakeHealthRepo{clearN: 12}
	svc := NewHealthService(&repository.Repository{Health: repo})

	n, err := svc.ClearAll(context.Background())
	if err != nil {
		t.Fatalf("ClearAll: %v", err)
	}
	if n != 12 {
		t.Fatalf("deleted = %d, want 12", n)
	}
	if repo.clearAllCalls != 1 {
		t.Fatalf("ClearAll calls = %d, want 1", repo.clearAllCalls)
	}
}

func TestServiceRecordHealthComputesDuration(t *testing.T) {
	repo := &fakeHealthRepo{}
	assetID := uuid.New()
	svc := New(&repository.Repository{Health: repo}, nil, nil, nil, time.Minute, time.Hour, cache.New(nil), 0, 0, NewHealthService(&repository.Repository{Health: repo}), nil)

	svc.recordHealth(context.Background(), &assetID, "meta_backfill", "failure", "error", "AAPL: boom", time.Now().Add(-120*time.Millisecond))
	if len(repo.recorded) != 1 {
		t.Fatalf("events recorded = %d, want 1", len(repo.recorded))
	}
	ev := repo.recorded[0]
	if ev.DurationMs < 100 {
		t.Fatalf("duration_ms = %d, want >= 100", ev.DurationMs)
	}
	if ev.AssetID == nil || *ev.AssetID != assetID || ev.EventType != "meta_backfill" {
		t.Fatalf("event = %+v", ev)
	}

	// nil-safe: a service without health tracking records nothing and does
	// not panic.
	bare := New(&repository.Repository{}, nil, nil, nil, time.Minute, time.Hour, cache.New(nil), 0, 0, nil, nil)
	bare.recordHealth(context.Background(), nil, "meta_backfill", "failure", "error", "x", time.Now())
}

func TestGetPriceHealthForwardsEventFilters(t *testing.T) {
	anyJob := uuid.New()
	tcs := []struct {
		name      string
		eventType string
		status    string
		jobID     string
	}{
		{"no filters pass through empty", "", "", ""},
		{"type only", "asset_meta", "", ""},
		{"status only", "", "failure", ""},
		{"type and status", "meta_backfill", "success", ""},
		{"job only", "", "", anyJob.String()},
		{"job with type and status", "etf_exposure", "failure", anyJob.String()},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeHealthRepo{
				sinceSummary: &model.HealthSummary{Successes: 4, Failures: 1},
				count:        5,
			}
			svc := NewHealthService(&repository.Repository{Health: repo})

			_, _, total, err := svc.GetPriceHealth(context.Background(), "today", 20, 5, tc.eventType, tc.status, tc.jobID)
			if err != nil {
				t.Fatalf("GetPriceHealth: %v", err)
			}
			if repo.pageCalls != 1 {
				t.Fatalf("GetEventsPage calls = %d, want 1", repo.pageCalls)
			}
			if repo.pageLimit != 20 || repo.pageOffset != 5 {
				t.Fatalf("page = (%d, %d), want (20, 5)", repo.pageLimit, repo.pageOffset)
			}
			// The page and the total must be filtered identically, otherwise
			// pagination under a filter drifts.
			if repo.pageEventType != tc.eventType || repo.pageStatus != tc.status {
				t.Fatalf("GetEventsPage filters = (%q, %q), want (%q, %q)",
					repo.pageEventType, repo.pageStatus, tc.eventType, tc.status)
			}
			if repo.countEventType != tc.eventType || repo.countStatus != tc.status {
				t.Fatalf("CountEvents filters = (%q, %q), want (%q, %q)",
					repo.countEventType, repo.countStatus, tc.eventType, tc.status)
			}
			if repo.pageJobID != tc.jobID || repo.countJobID != tc.jobID {
				t.Fatalf("job filter = page %q, count %q, want %q",
					repo.pageJobID, repo.countJobID, tc.jobID)
			}
			if total != repo.count {
				t.Fatalf("events_total = %d, want the filtered count %d", total, repo.count)
			}
			// The summary stays window-based and unfiltered.
			if len(repo.sinceCalls) != 1 {
				t.Fatalf("SummarySince calls = %d, want 1 (summary is unfiltered)", len(repo.sinceCalls))
			}
		})
	}
}

func TestGetPriceHealthEventsTotalIsTheFilteredCount(t *testing.T) {
	repo := &fakeHealthRepo{
		sinceSummary: &model.HealthSummary{Successes: 90, Failures: 10},
		count:        12,
		events:       []*model.HealthEvent{{EventType: "asset_meta", Status: "failure"}},
	}
	svc := NewHealthService(&repository.Repository{Health: repo})

	_, _, total, err := svc.GetPriceHealth(context.Background(), "today", 0, 0, "asset_meta", "failure", "")
	if err != nil {
		t.Fatalf("GetPriceHealth: %v", err)
	}
	if total != 12 {
		t.Fatalf("events_total = %d, want 12 (filtered count, not the summary total)", total)
	}
}

// TestGetActivityForwardsFiltersAndPagination covers the unified feed's
// pagination (default page size, clamp, passthrough), the type/status filters
// and the filtered items_total, including the empty-filter path.
func TestGetActivityForwardsFiltersAndPagination(t *testing.T) {
	jobID := uuid.New()
	items := []*model.ActivityItem{
		{Kind: "job", ID: jobID, Type: model.JobTypePriceRefresh, Status: model.JobStatusRunning, TargetType: model.JobTargetGlobal},
		{Kind: "event", ID: uuid.New(), Type: "etf_exposure", Status: "failure", TargetType: model.JobTargetGlobal},
	}

	tcs := []struct {
		name       string
		limit      int
		offset     int
		itemType   string
		status     string
		wantLimit  int
		wantOffset int
	}{
		{"empty filters fall back to the default page", 0, 0, "", "", 50, 0},
		{"negative page is normalized", -5, -10, "", "", 50, 0},
		{"oversized limit is capped", 10000, 0, "", "", 500, 0},
		{"valid page is kept", 25, 75, "", "", 25, 75},
		{"type filter", 20, 5, "price_refresh", "", 20, 5},
		{"status filter", 20, 5, "", "running", 20, 5},
		{"type and status filters", 20, 5, "etf_exposure", "failure", 20, 5},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeHealthRepo{activityItems: items, activityTotal: 137}
			svc := NewHealthService(&repository.Repository{Health: repo})

			got, total, err := svc.GetActivity(context.Background(), tc.limit, tc.offset, tc.itemType, tc.status)
			if err != nil {
				t.Fatalf("GetActivity: %v", err)
			}
			if repo.activityCalls != 1 || repo.activityCounted != 1 {
				t.Fatalf("calls = page %d, count %d; want 1 and 1", repo.activityCalls, repo.activityCounted)
			}
			if repo.activityLimit != tc.wantLimit || repo.activityOffset != tc.wantOffset {
				t.Fatalf("page = (%d, %d), want (%d, %d)",
					repo.activityLimit, repo.activityOffset, tc.wantLimit, tc.wantOffset)
			}
			// The page and the total ride on the same filters, otherwise
			// pagination under a filter drifts.
			if repo.activityType != tc.itemType || repo.activityStatus != tc.status {
				t.Fatalf("filters = (%q, %q), want (%q, %q)",
					repo.activityType, repo.activityStatus, tc.itemType, tc.status)
			}
			if total != 137 {
				t.Fatalf("items_total = %d, want 137", total)
			}
			if len(got) != len(items) {
				t.Fatalf("items = %d, want %d", len(got), len(items))
			}
		})
	}
}

// TestGetActivityResolvesTargetLabels checks that a job's and an event's asset
// target both resolve to the asset ticker through the batched job-label helper,
// while a global target stays empty.
func TestGetActivityResolvesTargetLabels(t *testing.T) {
	asset := &model.Asset{ID: uuid.New(), Ticker: "AAPL"}
	pf := &model.Portfolio{ID: uuid.New(), Name: "Mediolanum"}
	assetID := asset.ID
	pfID := pf.ID
	repo := &fakeHealthRepo{
		activityItems: []*model.ActivityItem{
			{Kind: "job", Type: model.JobTypeHistoryBackfill, TargetType: model.JobTargetAsset, TargetID: &assetID},
			{Kind: "job", Type: model.JobTypePriceRefresh, TargetType: model.JobTargetPortfolio, TargetID: &pfID},
			{Kind: "job", Type: model.JobTypeAssetSync, TargetType: model.JobTargetGlobal},
			{Kind: "event", Type: "etf_exposure", TargetType: model.JobTargetAsset, TargetID: &assetID},
			{Kind: "event", Type: "fx_refresh", TargetType: model.JobTargetGlobal},
		},
	}
	svc := NewHealthService(&repository.Repository{
		Health:    repo,
		Asset:     &fakeAssetRepo{assets: []*model.Asset{asset}},
		Portfolio: &fakePortfolioRepo{portfolio: pf},
	})

	items, _, err := svc.GetActivity(context.Background(), 0, 0, "", "")
	if err != nil {
		t.Fatalf("GetActivity: %v", err)
	}
	want := []string{"AAPL", "Mediolanum", "", "AAPL", ""}
	if len(items) != len(want) {
		t.Fatalf("items = %d, want %d", len(items), len(want))
	}
	for i, label := range want {
		if items[i].TargetLabel != label {
			t.Fatalf("item %d (%s/%s) label = %q, want %q",
				i, items[i].Kind, items[i].Type, items[i].TargetLabel, label)
		}
	}
}

func TestGetAssetMetaRecordsHealthEvents(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		repo := &fakeHealthRepo{}
		svc := newProviderHealthService(repo, &fakeYahooFetcher{meta: &price.AssetMeta{}}, nil)

		if _, err := svc.GetAssetMeta(ctx, "AAPL"); err != nil {
			t.Fatalf("GetAssetMeta: %v", err)
		}
		assertProviderEvent(t, repo, "asset_meta", "success", "AAPL")
	})

	t.Run("failure", func(t *testing.T) {
		repo := &fakeHealthRepo{}
		svc := newProviderHealthService(repo, &fakeYahooFetcher{metaErr: errors.New("yahoo down")}, nil)

		if _, err := svc.GetAssetMeta(ctx, "MSFT"); err == nil {
			t.Fatal("GetAssetMeta = nil error, want the provider failure")
		}
		assertProviderEvent(t, repo, "asset_meta", "failure", "MSFT")
	})

	t.Run("cache hit records nothing", func(t *testing.T) {
		repo := &fakeHealthRepo{}
		lk := &fakeLookupRepo{data: map[string][]byte{"meta:AAPL": []byte(`{}`)}}
		svc := New(&repository.Repository{Asset: &fakeAssetRepo{}, Lookup: lk}, nil,
			&fakeYahooFetcher{metaErr: errors.New("must not be called")}, nil,
			time.Minute, time.Hour, cache.New(nil), 0, 0,
			NewHealthService(&repository.Repository{Health: repo}), nil)

		if _, err := svc.GetAssetMeta(ctx, "aapl"); err != nil {
			t.Fatalf("GetAssetMeta: %v", err)
		}
		if len(repo.recorded) != 0 {
			t.Fatalf("recorded = %d, want 0 (no external call on a cache hit)", len(repo.recorded))
		}
	})
}

func TestFetchAssetExposureRecordsHealthEvents(t *testing.T) {
	ctx := context.Background()
	assetID := uuid.New()
	asset := &model.Asset{ID: assetID, Ticker: "AAPL", Type: model.AssetTypeStock}

	t.Run("success", func(t *testing.T) {
		repo := &fakeHealthRepo{}
		a := &fakeAssetRepo{asset: asset}
		svc := newProviderHealthService(repo, &fakeYahooFetcher{sector: "Technology"}, a)

		if _, err := svc.FetchAssetExposure(ctx, assetID); err != nil {
			t.Fatalf("FetchAssetExposure: %v", err)
		}
		ev := assertProviderEvent(t, repo, "asset_profile", "success", "AAPL")
		if ev.AssetID == nil || *ev.AssetID != assetID {
			t.Fatalf("event asset id = %v, want %v", ev.AssetID, assetID)
		}
	})

	t.Run("failure", func(t *testing.T) {
		repo := &fakeHealthRepo{}
		a := &fakeAssetRepo{asset: asset}
		svc := newProviderHealthService(repo, &fakeYahooFetcher{profileExtendedErr: errors.New("yahoo down")}, a)

		if _, err := svc.FetchAssetExposure(ctx, assetID); err == nil {
			t.Fatal("FetchAssetExposure = nil error, want the provider failure")
		}
		ev := assertProviderEvent(t, repo, "asset_profile", "failure", "AAPL")
		if ev.AssetID == nil || *ev.AssetID != assetID {
			t.Fatalf("event asset id = %v, want %v", ev.AssetID, assetID)
		}
	})
}

func TestBackfillAssetMetaRecordsHealthEvents(t *testing.T) {
	ctx := context.Background()
	stocks := []*model.Asset{
		{ID: uuid.New(), Ticker: "AAPL", Type: model.AssetTypeStock},
		{ID: uuid.New(), Ticker: "MSFT", Type: model.AssetTypeStock},
	}

	t.Run("success", func(t *testing.T) {
		repo := &fakeHealthRepo{}
		svc := newProviderHealthService(repo, &fakeYahooFetcher{}, &fakeAssetRepo{stocks: stocks})

		report, err := svc.BackfillAssetMeta(ctx, nil)
		if err != nil {
			t.Fatalf("BackfillAssetMeta: %v", err)
		}
		if report.Failed != 0 {
			t.Fatalf("failed = %d, want 0", report.Failed)
		}
		if got := countEventsByStatus(repo, "meta_backfill", "success"); got != 2 {
			t.Fatalf("success events = %d, want 2 (one per asset)", got)
		}
		if got := countEventsByStatus(repo, "meta_backfill", "failure"); got != 0 {
			t.Fatalf("failure events = %d, want 0", got)
		}
	})

	t.Run("failure", func(t *testing.T) {
		repo := &fakeHealthRepo{}
		svc := newProviderHealthService(repo, &failingProfile{fakeYahooFetcher: &fakeYahooFetcher{}}, &fakeAssetRepo{stocks: stocks})

		report, err := svc.BackfillAssetMeta(ctx, nil)
		if err != nil {
			t.Fatalf("BackfillAssetMeta: %v", err)
		}
		if report.Failed != 2 {
			t.Fatalf("failed = %d, want 2", report.Failed)
		}
		if got := countEventsByStatus(repo, "meta_backfill", "failure"); got != 2 {
			t.Fatalf("failure events = %d, want 2 (one per asset)", got)
		}
		if got := countEventsByStatus(repo, "meta_backfill", "success"); got != 0 {
			t.Fatalf("success events = %d, want 0", got)
		}
	})
}

// newProviderHealthService wires a Service with health tracking and no job
// queue, so the provider-boundary tests can assert the recorded events.
func newProviderHealthService(repo *fakeHealthRepo, yf yahooFetcher, a *fakeAssetRepo) *Service {
	if a == nil {
		a = &fakeAssetRepo{}
	}
	repos := &repository.Repository{
		Asset:    a,
		Exposure: &fakeExposureRepo{},
		Lookup:   &fakeLookupRepo{},
	}
	hs := NewHealthService(&repository.Repository{Health: repo})
	return New(repos, nil, yf, nil, time.Minute, time.Hour, cache.New(nil), 0, 0, hs, nil)
}

// assertProviderEvent checks that exactly one event of the wanted type and
// status was recorded and that the ticker appears in its message.
func assertProviderEvent(t *testing.T, repo *fakeHealthRepo, eventType, status, ticker string) *model.HealthEvent {
	t.Helper()
	var found *model.HealthEvent
	n := 0
	for _, ev := range repo.recorded {
		if ev.EventType != eventType {
			continue
		}
		n++
		if ev.Status == status {
			found = ev
		}
	}
	if n != 1 {
		t.Fatalf("%s events = %d, want 1 (%+v)", eventType, n, repo.recorded)
	}
	if found == nil {
		t.Fatalf("%s event status != %q", eventType, status)
	}
	if !strings.Contains(found.Message, ticker) {
		t.Fatalf("message = %q, want it to contain %q", found.Message, ticker)
	}
	return found
}

func countEventsByStatus(repo *fakeHealthRepo, eventType, status string) int {
	n := 0
	for _, ev := range repo.recorded {
		if ev.EventType == eventType && ev.Status == status {
			n++
		}
	}
	return n
}
