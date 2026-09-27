package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"github.com/alv67/peculium/internal/auth"
	"github.com/alv67/peculium/internal/service"
)

func (h *Handler) ListCurrencies(w http.ResponseWriter, r *http.Request) {
	currencies, err := h.svc.ListCurrencies(r.Context())
	if err != nil {
		log.Error().Err(err).Msg("list currencies failed")
		respondError(w, http.StatusInternalServerError, "list failed")
		return
	}
	respond(w, http.StatusOK, map[string]interface{}{"currencies": currencies})
}

func (h *Handler) CreateCurrency(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		Code string `json:"code"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	currency, err := h.svc.AddCurrency(r.Context(), req.Code, req.Name)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrCurrencyNotManaged):
			respondError(w, http.StatusUnprocessableEntity, "conversion USD->"+req.Code+" unavailable; currency not manageable")
		case errors.Is(err, service.ErrCurrencyExists):
			respondError(w, http.StatusConflict, "currency already in whitelist")
		case errors.Is(err, service.ErrInvalidInput):
			respondError(w, http.StatusBadRequest, "invalid currency code")
		default:
			log.Error().Err(err).Msg("create currency failed")
			respondError(w, http.StatusInternalServerError, "create failed")
		}
		return
	}

	respond(w, http.StatusCreated, currency)
}

func (h *Handler) DeleteCurrency(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	code := chi.URLParam(r, "code")
	if err := h.svc.DeleteCurrency(r.Context(), code); err != nil {
		switch {
		case errors.Is(err, service.ErrCurrencyInUse), errors.Is(err, service.ErrCurrencyProtected):
			respondError(w, http.StatusConflict, "currency is in use or protected and cannot be removed")
		case errors.Is(err, service.ErrNotFound):
			respondError(w, http.StatusNotFound, "currency not found")
		default:
			log.Error().Err(err).Msg("delete currency failed")
			respondError(w, http.StatusInternalServerError, "delete failed")
		}
		return
	}

	respond(w, http.StatusNoContent, nil)
}