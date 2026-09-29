package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"github.com/alv67/peculium/internal/model"
	"github.com/alv67/peculium/internal/service"
)

func (h *Handler) ListAdminUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.svc.ListUsers(r.Context())
	if err != nil {
		log.Error().Err(err).Msg("list users failed")
		respondError(w, http.StatusInternalServerError, "list failed")
		return
	}
	respond(w, http.StatusOK, users)
}

func (h *Handler) UpdateAdminUser(w http.ResponseWriter, r *http.Request) {
	uid, err := parseUUID(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	var req struct {
		Role   *string `json:"role"`
		Status *string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var role *model.Role
	if req.Role != nil {
		rr := model.Role(*req.Role)
		role = &rr
	}
	var status *model.Status
	if req.Status != nil {
		ss := model.Status(*req.Status)
		status = &ss
	}

	user, err := h.svc.UpdateUser(r.Context(), uid, role, status)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidInput):
			respondError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, service.ErrNotFound):
			respondError(w, http.StatusNotFound, "user not found")
		case errors.Is(err, service.ErrLastAdmin):
			respondError(w, http.StatusConflict, err.Error())
		default:
			log.Error().Err(err).Msg("update user failed")
			respondError(w, http.StatusInternalServerError, "update failed")
		}
		return
	}
	respond(w, http.StatusOK, user)
}

func (h *Handler) AdminResetPassword(w http.ResponseWriter, r *http.Request) {
	uid, err := parseUUID(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.svc.AdminResetPassword(r.Context(), uid, req.Password); err != nil {
		switch {
		case errors.Is(err, service.ErrWeakPassword):
			respondError(w, http.StatusBadRequest, "password must be at least 8 characters")
		case errors.Is(err, service.ErrNotFound):
			respondError(w, http.StatusNotFound, "user not found")
		default:
			log.Error().Err(err).Msg("admin reset password failed")
			respondError(w, http.StatusInternalServerError, "reset failed")
		}
		return
	}
	respond(w, http.StatusNoContent, nil)
}

func (h *Handler) GetAdminSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := h.svc.GetSettings(r.Context())
	if err != nil {
		log.Error().Err(err).Msg("get settings failed")
		respondError(w, http.StatusInternalServerError, "settings failed")
		return
	}
	respond(w, http.StatusOK, settings)
}

func (h *Handler) UpdateAdminSettings(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AutoApproveRegistrations *bool `json:"auto_approve_registrations"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.AutoApproveRegistrations == nil {
		respondError(w, http.StatusBadRequest, "auto_approve_registrations is required")
		return
	}

	settings, err := h.svc.UpdateSettings(r.Context(), *req.AutoApproveRegistrations)
	if err != nil {
		log.Error().Err(err).Msg("update settings failed")
		respondError(w, http.StatusInternalServerError, "update failed")
		return
	}
	respond(w, http.StatusOK, settings)
}
