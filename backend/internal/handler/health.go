package handler

import (
	"net/http"
	"strconv"

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

	summary, events, eventsTotal, err := h.svc.GetPriceHealth(ctx, r.URL.Query().Get("period"),
		queryInt(r, "limit", 0), queryInt(r, "offset", 0))
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
