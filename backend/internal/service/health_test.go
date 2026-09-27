package service

import (
	"context"
	"testing"
	"time"

	"github.com/alv67/peculium/internal/model"
	"github.com/alv67/peculium/internal/repository"
)

type fakeHealthRepo struct {
	recorded []*model.HealthEvent

	sinceCalls []time.Time
	lastNCalls []int
	pageCalls  int
	pageLimit  int
	pageOffset int

	sinceSummary *model.HealthSummary
	lastNSummary *model.HealthSummary
	count        int
	events       []*model.HealthEvent
}

func (f *fakeHealthRepo) RecordEvent(ctx context.Context, event *model.HealthEvent) error {
	f.recorded = append(f.recorded, event)
	return nil
}

func (f *fakeHealthRepo) GetEventsPage(ctx context.Context, limit, offset int) ([]*model.HealthEvent, error) {
	f.pageCalls++
	f.pageLimit = limit
	f.pageOffset = offset
	return f.events, nil
}

func (f *fakeHealthRepo) CountEvents(ctx context.Context) (int, error) {
	return f.count, nil
}

func (f *fakeHealthRepo) SummarySince(ctx context.Context, since time.Time) (*model.HealthSummary, error) {
	f.sinceCalls = append(f.sinceCalls, since)
	return f.sinceSummary, nil
}

func (f *fakeHealthRepo) SummaryLastN(ctx context.Context, n int) (*model.HealthSummary, error) {
	f.lastNCalls = append(f.lastNCalls, n)
	return f.lastNSummary, nil
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

	summary, events, _, err := svc.GetPriceHealth(context.Background(), "today", 0, 0)
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

	summary, _, _, err := svc.GetPriceHealth(context.Background(), "24h", 0, 0)
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

	summary, _, _, err := svc.GetPriceHealth(context.Background(), "100", 0, 0)
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

	summary, _, _, err := svc.GetPriceHealth(context.Background(), "nonsense", 0, 0)
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

	_, events, total, err := svc.GetPriceHealth(context.Background(), "today", 10000, -5)
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

	_, _, total, err := svc.GetPriceHealth(context.Background(), "today", 0, 0)
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

	_, _, _, err := svc.GetPriceHealth(context.Background(), "today", 25, 50)
	if err != nil {
		t.Fatalf("GetPriceHealth: %v", err)
	}
	if repo.pageLimit != 25 || repo.pageOffset != 50 {
		t.Fatalf("GetEventsPage(limit=%d, offset=%d), want (25, 50)", repo.pageLimit, repo.pageOffset)
	}
}
