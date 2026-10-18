package handler

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"github.com/alv67/peculium/internal/auth"
	"github.com/alv67/peculium/internal/model"
	"github.com/alv67/peculium/internal/service"
)

// enqueueJob queues long external-site work for the worker instead of
// running it inline, answering 202 with the job id so the client can poll
// GET /jobs/{id}. Repository-side dedup means a second request for the same
// open target returns that job (with its real status) rather than a new one.
func (h *Handler) enqueueJob(w http.ResponseWriter, r *http.Request, job *model.Job) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	job.RequestedBy = &claims.UserID

	created, err := h.svc.Jobs.Enqueue(r.Context(), job)
	if err != nil {
		log.Error().Err(err).Str("type", job.Type).Msg("job enqueue failed")
		respondError(w, http.StatusInternalServerError, "could not queue job")
		return
	}
	respond(w, http.StatusAccepted, map[string]string{"job_id": created.ID.String(), "status": created.Status})
}

// GetJob reports a single job's status/progress for the async routes.
func (h *Handler) GetJob(w http.ResponseWriter, r *http.Request) {
	if auth.GetClaims(r.Context()) == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id, err := parseUUID(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid job id")
		return
	}
	job, err := h.svc.Jobs.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			respondError(w, http.StatusNotFound, "job not found")
			return
		}
		log.Error().Err(err).Msg("get job failed")
		respondError(w, http.StatusInternalServerError, "get job failed")
		return
	}
	respond(w, http.StatusOK, job)
}

// ListJobs returns queued/finished jobs, newest first.
func (h *Handler) ListJobs(w http.ResponseWriter, r *http.Request) {
	if auth.GetClaims(r.Context()) == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	jobs, err := h.svc.Jobs.List(r.Context(), queryInt(r, "limit", 0), queryInt(r, "offset", 0))
	if err != nil {
		log.Error().Err(err).Msg("list jobs failed")
		respondError(w, http.StatusInternalServerError, "list jobs failed")
		return
	}
	respond(w, http.StatusOK, jobs)
}
