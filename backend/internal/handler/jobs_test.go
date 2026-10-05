package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/alv67/peculium/internal/auth"
	"github.com/alv67/peculium/internal/cache"
	"github.com/alv67/peculium/internal/model"
	"github.com/alv67/peculium/internal/repository"
	"github.com/alv67/peculium/internal/service"
)

// fakeJobQueue is an in-memory repository.JobRepository keeping just enough
// behavior for the async routes: enqueue stores the job, get/list read back.
type fakeJobQueue struct {
	jobs       []*model.Job
	lastLimit  int
	lastOffset int
}

var _ repository.JobRepository = (*fakeJobQueue)(nil)

func (f *fakeJobQueue) Enqueue(ctx context.Context, job *model.Job) (*model.Job, error) {
	queued := &model.Job{
		ID: uuid.New(), Type: job.Type, TargetType: job.TargetType, TargetID: job.TargetID,
		Status: model.JobStatusQueued, RequestedBy: job.RequestedBy,
		CreatedAt: time.Now(),
	}
	f.jobs = append(f.jobs, queued)
	return queued, nil
}

func (f *fakeJobQueue) ClaimNext(ctx context.Context) (*model.Job, error) { return nil, nil }
func (f *fakeJobQueue) UpdateProgress(ctx context.Context, id uuid.UUID, processed, total int) error {
	return nil
}
func (f *fakeJobQueue) SetCheckpoint(ctx context.Context, id uuid.UUID, checkpoint []byte) error {
	return nil
}
func (f *fakeJobQueue) Finish(ctx context.Context, id uuid.UUID, status string, errMsg string) error {
	return nil
}
func (f *fakeJobQueue) DeleteFinishedBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	return 0, nil
}

func (f *fakeJobQueue) GetByID(ctx context.Context, id uuid.UUID) (*model.Job, error) {
	for _, j := range f.jobs {
		if j.ID == id {
			return j, nil
		}
	}
	return nil, pgx.ErrNoRows
}

func (f *fakeJobQueue) List(ctx context.Context, limit, offset int) ([]*model.Job, error) {
	f.lastLimit, f.lastOffset = limit, offset
	return f.jobs, nil
}

func newJobTestHandler() (*Handler, *fakeJobQueue) {
	fq := &fakeJobQueue{}
	svc := service.New(&repository.Repository{Job: fq}, nil, nil, nil, time.Minute, time.Hour, cache.New(nil), 0, 0, nil, nil)
	return New(svc, nil), fq
}

// authRequest builds an authenticated request with chi URL params, mirroring
// what the mounted routes deliver to the handlers.
func authRequest(method, target string, params map[string]string) *http.Request {
	r := httptest.NewRequest(method, target, nil)
	rc := chi.NewRouteContext()
	for k, v := range params {
		rc.URLParams.Add(k, v)
	}
	ctx := context.WithValue(r.Context(), chi.RouteCtxKey, rc)
	claims := &auth.Claims{UserID: uuid.New(), Email: "user@example.com", Role: "user"}
	return r.WithContext(context.WithValue(ctx, auth.UserContextKey, claims))
}

func decode202(t *testing.T, rec *httptest.ResponseRecorder) map[string]string {
	t.Helper()
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202 (body %q)", rec.Code, rec.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if id, err := uuid.Parse(body["job_id"]); err != nil || id == uuid.Nil {
		t.Fatalf("job_id = %q, want a parsed uuid", body["job_id"])
	}
	if body["status"] != model.JobStatusQueued {
		t.Fatalf("status = %q, want queued", body["status"])
	}
	return body
}

func TestAsyncRoutesEnqueueJobs(t *testing.T) {
	assetID := uuid.New()

	tcs := []struct {
		name       string
		handler    func(*Handler) http.HandlerFunc
		target     string
		params     map[string]string
		wantType   string
		wantTarget string
		wantID     *uuid.UUID
	}{
		{"prices/refresh global", func(h *Handler) http.HandlerFunc { return h.RefreshPrices }, "/prices/refresh", nil, model.JobTypePriceRefresh, model.JobTargetGlobal, nil},
		{"assets/sync", func(h *Handler) http.HandlerFunc { return h.SyncAssets }, "/assets/sync", nil, model.JobTypeAssetSync, model.JobTargetGlobal, nil},
		{"backfill-history", func(h *Handler) http.HandlerFunc { return h.BackfillAssetHistory }, "/assets/" + assetID.String() + "/backfill-history", map[string]string{"id": assetID.String()}, model.JobTypeHistoryBackfill, model.JobTargetAsset, &assetID},
		{"backfill-meta", func(h *Handler) http.HandlerFunc { return h.BackfillAssetMeta }, "/assets/backfill-meta", nil, model.JobTypeMetaBackfill, model.JobTargetGlobal, nil},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			h, fq := newJobTestHandler()
			rec := httptest.NewRecorder()
			tc.handler(h)(rec, authRequest(http.MethodPost, tc.target, tc.params))

			decode202(t, rec)
			if len(fq.jobs) != 1 {
				t.Fatalf("queued jobs = %d, want 1", len(fq.jobs))
			}
			job := fq.jobs[0]
			if job.Type != tc.wantType || job.TargetType != tc.wantTarget {
				t.Fatalf("job = %s/%s, want %s/%s", job.Type, job.TargetType, tc.wantType, tc.wantTarget)
			}
			if (job.TargetID == nil) != (tc.wantID == nil) || (tc.wantID != nil && *job.TargetID != *tc.wantID) {
				t.Fatalf("target_id = %v, want %v", job.TargetID, tc.wantID)
			}
			if job.RequestedBy == nil {
				t.Fatal("requested_by must be stamped from the caller")
			}
		})
	}
}

func TestEnqueueRejectsBadAssetID(t *testing.T) {
	h, fq := newJobTestHandler()
	rec := httptest.NewRecorder()
	h.BackfillAssetHistory(rec, authRequest(http.MethodPost, "/assets/nope/backfill-history", map[string]string{"id": "nope"}))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if len(fq.jobs) != 0 {
		t.Fatal("invalid id must not enqueue")
	}
}

func TestEnqueueUnauthorized(t *testing.T) {
	h, _ := newJobTestHandler()
	rec := httptest.NewRecorder()
	h.BackfillAssetMeta(rec, httptest.NewRequest(http.MethodPost, "/assets/backfill-meta", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestGetAndListJobs(t *testing.T) {
	h, fq := newJobTestHandler()
	rec := httptest.NewRecorder()
	h.RefreshPrices(rec, authRequest(http.MethodPost, "/prices/refresh", nil))
	jobID := decode202(t, rec)["job_id"]

	rec = httptest.NewRecorder()
	// Stamp a finished run: GET derives duration_ms from the timestamps,
	// while the summary stays absent without a health repository.
	started := time.Now().Add(-3 * time.Second)
	finished := started.Add(2 * time.Second)
	fq.jobs[0].StartedAt, fq.jobs[0].FinishedAt = &started, &finished
	h.GetJob(rec, authRequest(http.MethodGet, "/jobs/"+jobID, map[string]string{"id": jobID}))
	if rec.Code != http.StatusOK {
		t.Fatalf("get status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
	var got model.Job
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.ID.String() != jobID || got.Type != model.JobTypePriceRefresh {
		t.Fatalf("job = %+v, want the enqueued price_refresh", got)
	}
	if got.DurationMs != 2000 {
		t.Fatalf("duration_ms = %d, want 2000", got.DurationMs)
	}
	if got.Summary != nil {
		t.Fatalf("summary = %+v, want none without a health repo", got.Summary)
	}

	rec = httptest.NewRecorder()
	h.GetJob(rec, authRequest(http.MethodGet, "/jobs/not-a-uuid", map[string]string{"id": "not-a-uuid"}))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bad id status = %d, want 400", rec.Code)
	}

	rec = httptest.NewRecorder()
	missing := uuid.New().String()
	h.GetJob(rec, authRequest(http.MethodGet, "/jobs/"+missing, map[string]string{"id": missing}))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown id status = %d, want 404", rec.Code)
	}

	rec = httptest.NewRecorder()
	h.ListJobs(rec, authRequest(http.MethodGet, "/jobs?limit=2", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d, want 200", rec.Code)
	}
	if fq.lastLimit != 2 {
		t.Fatalf("list limit = %d, want 2", fq.lastLimit)
	}
	var list []*model.Job
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil || len(list) != 1 {
		t.Fatalf("list body = %q, want 1 job", rec.Body.String())
	}
}
