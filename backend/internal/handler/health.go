package handler

import (
	"net/http"
	"strconv"

	"github.com/rs/zerolog/log"

	"github.com/alv67/peculium/internal/service"
)

type HealthHandler struct {
	svc *service.HealthService
}

func NewHealthHandler(svc *service.HealthService) *HealthHandler {
	return &HealthHandler{svc: svc}
}

func (h *HealthHandler) GetPriceHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()

	summary, events, eventsTotal, err := h.svc.GetPriceHealth(ctx, q.Get("period"),
		queryInt(r, "limit", 0), queryInt(r, "offset", 0), q.Get("type"), q.Get("status"), q.Get("job_id"))
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to fetch price health")
		return
	}

	respond(w, http.StatusOK, map[string]any{
		"summary":      summary,
		"events":       events,
		"events_total": eventsTotal,
	})
}

// GetHealthActivity returns the unified activity feed: queue jobs plus the
// health events that are not tied to one, newest first. period scopes the
// list to the same window as the summary tiles; type and status filter the
// item's own type and raw status; job events are not part of this list (they
// come from GET /health/prices?job_id=...).
func (h *HealthHandler) GetHealthActivity(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	q := r.URL.Query()

	items, total, err := h.svc.GetActivity(ctx, q.Get("period"),
		queryInt(r, "limit", 0), queryInt(r, "offset", 0), q.Get("type"), q.Get("status"))
	if err != nil {
		log.Error().Err(err).Msg("list activity failed")
		respondError(w, http.StatusInternalServerError, "failed to fetch activity")
		return
	}

	respond(w, http.StatusOK, map[string]any{
		"items":       items,
		"items_total": total,
	})
}

func (h *HealthHandler) AdminClearHealthEvents(w http.ResponseWriter, r *http.Request) {
	n, err := h.svc.ClearAll(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to clear health events")
		return
	}
	respond(w, http.StatusOK, map[string]any{"deleted": n})
}

// queryInt reads an integer query parameter, returning fallback when the
// parameter is missing or not a valid integer. Out-of-range values are
// clamped by the service layer.
func queryInt(r *http.Request, name string, fallback int) int {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return v
}
