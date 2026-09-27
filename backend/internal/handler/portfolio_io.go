package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/alv67/peculium/internal/auth"
	"github.com/alv67/peculium/internal/model"
	"github.com/alv67/peculium/internal/service"
)

// ExportPortfolio streams a portfolio as a downloadable JSON document.
func (h *Handler) ExportPortfolio(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	pid, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid portfolio id")
		return
	}

	doc, err := h.svc.ExportPortfolio(r.Context(), pid, claims.UserID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrForbidden):
			respondError(w, http.StatusForbidden, "forbidden")
		case errors.Is(err, service.ErrNotFound):
			respondError(w, http.StatusNotFound, "not found")
		default:
			log.Error().Err(err).Msg("export portfolio failed")
			respondError(w, http.StatusInternalServerError, "export failed")
		}
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, exportFilename(doc.Portfolio.Name)))
	respond(w, http.StatusOK, doc)
}

// ImportPortfolio restores a portfolio from an exported document, either as a
// brand new portfolio or by overwriting an existing one.
func (h *Handler) ImportPortfolio(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		Document          model.PortfolioExport `json:"document"`
		Mode              string                `json:"mode"`
		Name              string                `json:"name"`
		TargetPortfolioID string                `json:"target_portfolio_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var target *uuid.UUID
	if req.TargetPortfolioID != "" {
		uid, err := uuid.Parse(req.TargetPortfolioID)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid target portfolio id")
			return
		}
		target = &uid
	}

	pf, err := h.svc.ImportPortfolio(r.Context(), claims.UserID, &req.Document, req.Mode, req.Name, target)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidInput):
			respondError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, service.ErrForbidden):
			respondError(w, http.StatusForbidden, "forbidden")
		default:
			log.Error().Err(err).Msg("import portfolio failed")
			respondError(w, http.StatusInternalServerError, "import failed")
		}
		return
	}

	respond(w, http.StatusCreated, pf)
}

// SyncAssets refreshes asset-level market data (splits re-check + price
// history up to date) for every asset. Called once per app load.
func (h *Handler) SyncAssets(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if err := h.svc.SyncAssetData(r.Context()); err != nil {
		log.Error().Err(err).Msg("asset sync failed")
		respondError(w, http.StatusInternalServerError, "sync failed")
		return
	}

	respond(w, http.StatusOK, map[string]string{"status": "ok"})
}

func exportFilename(name string) string {
	slug := strings.ToLower(strings.TrimSpace(name))
	var b strings.Builder
	for _, r := range slug {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		case r == ' ':
			b.WriteByte('-')
		}
	}
	if b.Len() == 0 {
		b.WriteString("portfolio")
	}
	return "peculium-" + b.String() + ".json"
}
