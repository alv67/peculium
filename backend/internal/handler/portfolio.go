package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/alv67/peculium/internal/auth"
	"github.com/alv67/peculium/internal/model"
	"github.com/alv67/peculium/internal/service"
)

func parseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}

func (h *Handler) CreatePortfolio(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Currency    string `json:"currency"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Currency == "" {
		req.Currency = "USD"
	}

	portfolio, err := h.svc.CreatePortfolio(r.Context(), claims.UserID, req.Name, req.Description, req.Currency)
	if err != nil {
		log.Error().Err(err).Msg("create portfolio failed")
		respondError(w, http.StatusInternalServerError, "create failed")
		return
	}

	respond(w, http.StatusCreated, portfolio)
}

func (h *Handler) ListPortfolios(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	portfolios, err := h.svc.ListPortfolios(r.Context(), claims.UserID)
	if err != nil {
		log.Error().Err(err).Msg("list portfolios failed")
		respondError(w, http.StatusInternalServerError, "list failed")
		return
	}

	respond(w, http.StatusOK, portfolios)
}

func (h *Handler) GetPortfolio(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")
	uid, err := parseUUID(id)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid portfolio id")
		return
	}

	portfolio, err := h.svc.GetPortfolio(r.Context(), uid, claims.UserID)
	if err != nil {
		if err == service.ErrForbidden {
			respondError(w, http.StatusForbidden, "forbidden")
			return
		}
		respondError(w, http.StatusNotFound, "portfolio not found")
		return
	}

	respond(w, http.StatusOK, portfolio)
}

func (h *Handler) UpdatePortfolio(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	uid, err := parseUUID(id)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid portfolio id")
		return
	}

	var p model.Portfolio
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	p.ID = uid

	if err := h.svc.UpdatePortfolio(r.Context(), &p); err != nil {
		log.Error().Err(err).Msg("update portfolio failed")
		respondError(w, http.StatusInternalServerError, "update failed")
		return
	}

	respond(w, http.StatusOK, p)
}

func (h *Handler) DeletePortfolio(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	uid, err := parseUUID(id)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid portfolio id")
		return
	}

	if err := h.svc.DeletePortfolio(r.Context(), uid); err != nil {
		log.Error().Err(err).Msg("delete portfolio failed")
		respondError(w, http.StatusInternalServerError, "delete failed")
		return
	}

	respond(w, http.StatusNoContent, nil)
}

func (h *Handler) CreateTransaction(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	portfolioID, err := parseUUID(id)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid portfolio id")
		return
	}

	var tx model.Transaction
	if err := json.NewDecoder(r.Body).Decode(&tx); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	tx.PortfolioID = portfolioID

	created, err := h.svc.AddTransaction(r.Context(), &tx)
	if err != nil {
		log.Error().Err(err).Msg("create transaction failed")
		respondError(w, http.StatusInternalServerError, "create failed")
		return
	}

	respond(w, http.StatusCreated, created)
}

func (h *Handler) UpdateTransaction(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")
	txID, err := parseUUID(id)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid transaction id")
		return
	}

	var tx model.Transaction
	if err := json.NewDecoder(r.Body).Decode(&tx); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	tx.ID = txID

	if err := h.svc.UpdateTransaction(r.Context(), claims.UserID, &tx); err != nil {
		if err == service.ErrForbidden {
			respondError(w, http.StatusForbidden, "forbidden")
			return
		}
		if err == service.ErrNotFound {
			respondError(w, http.StatusNotFound, "transaction not found")
			return
		}
		log.Error().Err(err).Msg("update transaction failed")
		respondError(w, http.StatusInternalServerError, "update failed")
		return
	}

	respond(w, http.StatusOK, tx)
}

func (h *Handler) DeleteTransaction(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")
	txID, err := parseUUID(id)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid transaction id")
		return
	}

	if err := h.svc.DeleteTransaction(r.Context(), claims.UserID, txID); err != nil {
		if err == service.ErrForbidden {
			respondError(w, http.StatusForbidden, "forbidden")
			return
		}
		if err == service.ErrNotFound {
			respondError(w, http.StatusNotFound, "transaction not found")
			return
		}
		log.Error().Err(err).Msg("delete transaction failed")
		respondError(w, http.StatusInternalServerError, "delete failed")
		return
	}

	respond(w, http.StatusNoContent, nil)
}

func (h *Handler) ListTransactions(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")
	portfolioID, err := parseUUID(id)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid portfolio id")
		return
	}

	limit := 0
	if raw := r.URL.Query().Get("limit"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid limit")
			return
		}
		limit = v
	}
	offset := 0
	if raw := r.URL.Query().Get("offset"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid offset")
			return
		}
		offset = v
	}

	filter, err := model.ParseTransactionFilter(r.URL.Query())
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	page, err := h.svc.ListTransactionsPaged(r.Context(), portfolioID, claims.UserID, limit, offset, filter)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrForbidden):
			respondError(w, http.StatusForbidden, "forbidden")
			return
		case errors.Is(err, service.ErrInvalidInput):
			respondError(w, http.StatusBadRequest, "invalid pagination parameters")
			return
		case errors.Is(err, service.ErrNotFound):
			respondError(w, http.StatusNotFound, "portfolio not found")
			return
		}
		log.Error().Err(err).Msg("list transactions failed")
		respondError(w, http.StatusInternalServerError, "list failed")
		return
	}

	respond(w, http.StatusOK, page)
}

func (h *Handler) GetPortfolioSummary(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")
	portfolioID, err := parseUUID(id)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid portfolio id")
		return
	}

	if _, err := h.svc.GetPortfolio(r.Context(), portfolioID, claims.UserID); err != nil {
		if err == service.ErrForbidden {
			respondError(w, http.StatusForbidden, "forbidden")
			return
		}
		respondError(w, http.StatusNotFound, "portfolio not found")
		return
	}

	summary, err := h.svc.GetPortfolioSummary(r.Context(), portfolioID)
	if err != nil {
		log.Error().Err(err).Msg("get summary failed")
		respondError(w, http.StatusInternalServerError, "summary failed")
		return
	}

	respond(w, http.StatusOK, summary)
}

func (h *Handler) GetPortfolioAllocation(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")
	portfolioID, err := parseUUID(id)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid portfolio id")
		return
	}

	if _, err := h.svc.GetPortfolio(r.Context(), portfolioID, claims.UserID); err != nil {
		if err == service.ErrForbidden {
			respondError(w, http.StatusForbidden, "forbidden")
			return
		}
		respondError(w, http.StatusNotFound, "portfolio not found")
		return
	}

	allocation, err := h.svc.GetPortfolioAllocation(r.Context(), portfolioID)
	if err != nil {
		log.Error().Err(err).Msg("get allocation failed")
		respondError(w, http.StatusInternalServerError, "allocation failed")
		return
	}

	respond(w, http.StatusOK, allocation)
}

func (h *Handler) GetPortfolioClassAllocation(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")
	portfolioID, err := parseUUID(id)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid portfolio id")
		return
	}

	if _, err := h.svc.GetPortfolio(r.Context(), portfolioID, claims.UserID); err != nil {
		if err == service.ErrForbidden {
			respondError(w, http.StatusForbidden, "forbidden")
			return
		}
		respondError(w, http.StatusNotFound, "portfolio not found")
		return
	}

	allocation, err := h.svc.GetPortfolioClassAllocation(r.Context(), portfolioID)
	if err != nil {
		log.Error().Err(err).Msg("get class allocation failed")
		respondError(w, http.StatusInternalServerError, "class allocation failed")
		return
	}

	respond(w, http.StatusOK, allocation)
}

func (h *Handler) GetPortfolioGeographyAllocation(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")
	portfolioID, err := parseUUID(id)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid portfolio id")
		return
	}

	if _, err := h.svc.GetPortfolio(r.Context(), portfolioID, claims.UserID); err != nil {
		if err == service.ErrForbidden {
			respondError(w, http.StatusForbidden, "forbidden")
			return
		}
		respondError(w, http.StatusNotFound, "portfolio not found")
		return
	}

	allocation, err := h.svc.GetPortfolioGeographyAllocation(r.Context(), portfolioID)
	if err != nil {
		log.Error().Err(err).Msg("get geography allocation failed")
		respondError(w, http.StatusInternalServerError, "geography allocation failed")
		return
	}

	respond(w, http.StatusOK, allocation)
}

func (h *Handler) GetPortfolioSectorAllocation(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")
	portfolioID, err := parseUUID(id)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid portfolio id")
		return
	}

	if _, err := h.svc.GetPortfolio(r.Context(), portfolioID, claims.UserID); err != nil {
		if err == service.ErrForbidden {
			respondError(w, http.StatusForbidden, "forbidden")
			return
		}
		respondError(w, http.StatusNotFound, "portfolio not found")
		return
	}

	allocation, err := h.svc.GetPortfolioSectorAllocation(r.Context(), portfolioID)
	if err != nil {
		log.Error().Err(err).Msg("get sector allocation failed")
		respondError(w, http.StatusInternalServerError, "sector allocation failed")
		return
	}

	respond(w, http.StatusOK, allocation)
}

func (h *Handler) GetPortfolioAllocationDrill(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")
	portfolioID, err := parseUUID(id)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid portfolio id")
		return
	}

	if _, err := h.svc.GetPortfolio(r.Context(), portfolioID, claims.UserID); err != nil {
		if err == service.ErrForbidden {
			respondError(w, http.StatusForbidden, "forbidden")
			return
		}
		respondError(w, http.StatusNotFound, "portfolio not found")
		return
	}

	query := r.URL.Query()
	drill, err := h.svc.GetPortfolioAllocationDrill(r.Context(), portfolioID, query.Get("dim"), query.Get("key"))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidInput):
			respondError(w, http.StatusBadRequest, "dim must be one of class, country, region, sector and key is required")
			return
		case errors.Is(err, service.ErrNotFound):
			respondError(w, http.StatusNotFound, "portfolio not found")
			return
		case errors.Is(err, service.ErrForbidden):
			respondError(w, http.StatusForbidden, "forbidden")
			return
		}
		log.Error().Err(err).Msg("get portfolio allocation drill-down failed")
		respondError(w, http.StatusInternalServerError, "allocation drill-down failed")
		return
	}

	respond(w, http.StatusOK, drill)
}

func (h *Handler) GetPortfolioPerformance(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")
	portfolioID, err := parseUUID(id)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid portfolio id")
		return
	}

	if _, err := h.svc.GetPortfolio(r.Context(), portfolioID, claims.UserID); err != nil {
		if err == service.ErrForbidden {
			respondError(w, http.StatusForbidden, "forbidden")
			return
		}
		respondError(w, http.StatusNotFound, "portfolio not found")
		return
	}

	performance, err := h.svc.GetPortfolioPerformance(r.Context(), portfolioID)
	if err != nil {
		log.Error().Err(err).Msg("get performance failed")
		respondError(w, http.StatusInternalServerError, "performance failed")
		return
	}

	respond(w, http.StatusOK, performance)
}

func (h *Handler) GetPortfolioPerformanceBuckets(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")
	portfolioID, err := parseUUID(id)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid portfolio id")
		return
	}

	granularity := "month"
	if query := r.URL.Query(); query.Has("granularity") {
		granularity = query.Get("granularity")
	}

	perf, err := h.svc.GetPortfolioPerformanceBuckets(r.Context(), portfolioID, claims.UserID, granularity)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidInput):
			respondError(w, http.StatusBadRequest, "granularity must be month or year")
			return
		case errors.Is(err, service.ErrForbidden):
			respondError(w, http.StatusForbidden, "forbidden")
			return
		case errors.Is(err, service.ErrNotFound):
			respondError(w, http.StatusNotFound, "portfolio not found")
			return
		}
		log.Error().Err(err).Msg("get portfolio performance buckets failed")
		respondError(w, http.StatusInternalServerError, "portfolio performance buckets failed")
		return
	}

	respond(w, http.StatusOK, perf)
}

func (h *Handler) GetPortfolioROI(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")
	portfolioID, err := parseUUID(id)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid portfolio id")
		return
	}

	if _, err := h.svc.GetPortfolio(r.Context(), portfolioID, claims.UserID); err != nil {
		if err == service.ErrForbidden {
			respondError(w, http.StatusForbidden, "forbidden")
			return
		}
		respondError(w, http.StatusNotFound, "portfolio not found")
		return
	}

	roi, err := h.svc.GetPortfolioROI(r.Context(), portfolioID)
	if err != nil {
		log.Error().Err(err).Msg("get roi failed")
		respondError(w, http.StatusInternalServerError, "roi failed")
		return
	}

	respond(w, http.StatusOK, roi)
}

func (h *Handler) GetPortfolioHistory(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id := chi.URLParam(r, "id")
	uid, err := parseUUID(id)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid portfolio id")
		return
	}

	if _, err := h.svc.GetPortfolio(r.Context(), uid, claims.UserID); err != nil {
		if err == service.ErrForbidden {
			respondError(w, http.StatusForbidden, "forbidden")
			return
		}
		respondError(w, http.StatusNotFound, "portfolio not found")
		return
	}

	history, err := h.svc.GetPortfolioHistory(r.Context(), uid)
	if err != nil {
		log.Error().Err(err).Msg("get portfolio history failed")
		respondError(w, http.StatusInternalServerError, "history failed")
		return
	}

	respond(w, http.StatusOK, history)
}

func (h *Handler) GetPrices(w http.ResponseWriter, r *http.Request) {
	assetID := chi.URLParam(r, "assetID")
	uid, err := parseUUID(assetID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid asset id")
		return
	}

	full := r.URL.Query().Get("full") == "1"
	prices, err := h.svc.GetPrices(r.Context(), uid, full)
	if err != nil {
		log.Error().Err(err).Msg("get prices failed")
		respondError(w, http.StatusInternalServerError, "get prices failed")
		return
	}

	respond(w, http.StatusOK, prices)
}

func (h *Handler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	dash, err := h.svc.GetDashboard(r.Context(), claims.UserID)
	if err != nil {
		log.Error().Err(err).Msg("get dashboard failed")
		respondError(w, http.StatusInternalServerError, "dashboard failed")
		return
	}

	respond(w, http.StatusOK, dash)
}

func (h *Handler) GetDashboardAllocation(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	dashAlloc, err := h.svc.GetDashboardAllocation(r.Context(), claims.UserID)
	if err != nil {
		log.Error().Err(err).Msg("get dashboard allocation failed")
		respondError(w, http.StatusInternalServerError, "dashboard allocation failed")
		return
	}

	respond(w, http.StatusOK, dashAlloc)
}

func (h *Handler) GetDashboardAllocationDrill(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	query := r.URL.Query()
	drill, err := h.svc.GetDashboardAllocationDrill(r.Context(), claims.UserID, query.Get("dim"), query.Get("key"))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidInput):
			respondError(w, http.StatusBadRequest, "dim must be one of class, country, region, sector and key is required")
			return
		case errors.Is(err, service.ErrNotFound):
			respondError(w, http.StatusNotFound, "not found")
			return
		}
		log.Error().Err(err).Msg("get dashboard allocation drill-down failed")
		respondError(w, http.StatusInternalServerError, "dashboard allocation drill-down failed")
		return
	}

	respond(w, http.StatusOK, drill)
}

func (h *Handler) GetDashboardPerformance(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	granularity := "month"
	if query := r.URL.Query(); query.Has("granularity") {
		granularity = query.Get("granularity")
	}

	perf, err := h.svc.GetDashboardPerformance(r.Context(), claims.UserID, granularity)
	if err != nil {
		if errors.Is(err, service.ErrInvalidInput) {
			respondError(w, http.StatusBadRequest, "granularity must be month or year")
			return
		}
		log.Error().Err(err).Msg("get dashboard performance failed")
		respondError(w, http.StatusInternalServerError, "dashboard performance failed")
		return
	}

	respond(w, http.StatusOK, perf)
}

func (h *Handler) RefreshPrices(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var portfolioID *uuid.UUID
	if pid := r.URL.Query().Get("portfolio_id"); pid != "" {
		uid, err := parseUUID(pid)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid portfolio id")
			return
		}
		if _, err := h.svc.GetPortfolio(r.Context(), uid, claims.UserID); err != nil {
			respondError(w, http.StatusForbidden, "forbidden")
			return
		}
		portfolioID = &uid
	}

	report, err := h.svc.RefreshPrices(r.Context(), portfolioID)
	if err != nil {
		log.Error().Err(err).Msg("refresh prices failed")
		respondError(w, http.StatusInternalServerError, "refresh failed")
		return
	}

	respond(w, http.StatusOK, report)
}
