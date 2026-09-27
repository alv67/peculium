package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"github.com/alv67/peculium/internal/auth"
	"github.com/alv67/peculium/internal/model"
	"github.com/alv67/peculium/internal/service"
)

type Handler struct {
	svc     *service.Service
	jwtAuth *auth.JWTAuth
	health  *HealthHandler
}

func New(svc *service.Service, jwtAuth *auth.JWTAuth) *Handler {
	return &Handler{
		svc:     svc,
		jwtAuth: jwtAuth,
		health:  NewHealthHandler(svc.Health),
	}
}

func respond(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

func respondError(w http.ResponseWriter, status int, msg string) {
	respond(w, status, map[string]string{"error": msg})
}

// refreshRequested reports whether the caller asked to bypass the provider
// exposure cache via ?refresh=1 (or refresh=true).
func refreshRequested(r *http.Request) bool {
	flag := r.URL.Query().Get("refresh")
	return flag == "1" || strings.EqualFold(flag, "true")
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Name     string `json:"name"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.svc.Register(r.Context(), req.Email, req.Name, req.Password)
	if err != nil {
		if err == service.ErrEmailExists {
			respondError(w, http.StatusConflict, "email already registered")
			return
		}
		log.Error().Err(err).Msg("registration failed")
		respondError(w, http.StatusInternalServerError, "registration failed")
		return
	}

	respond(w, http.StatusCreated, user)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, accessToken, refreshToken, err := h.svc.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if err == service.ErrInvalidCredentials {
			respondError(w, http.StatusUnauthorized, "invalid email or password")
			return
		}
		log.Error().Err(err).Msg("login failed")
		respondError(w, http.StatusInternalServerError, "login failed")
		return
	}

	respond(w, http.StatusOK, map[string]interface{}{
		"user":          user,
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

func (h *Handler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	accessToken, refreshToken, err := h.svc.RefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "invalid refresh token")
		return
	}

	respond(w, http.StatusOK, map[string]string{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

func (h *Handler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	user, err := h.svc.GetCurrentUser(r.Context(), claims)
	if err != nil {
		respondError(w, http.StatusNotFound, "user not found")
		return
	}

	respond(w, http.StatusOK, user)
}

func (h *Handler) UpdateCurrentUser(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		Name         string `json:"name"`
		Email        string `json:"email"`
		BaseCurrency string `json:"base_currency"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.svc.UpdateProfile(r.Context(), claims.UserID, req.Name, req.Email, req.BaseCurrency)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidInput):
			respondError(w, http.StatusBadRequest, "invalid name, email or base currency")
		case errors.Is(err, service.ErrEmailExists):
			respondError(w, http.StatusConflict, "email already registered")
		case errors.Is(err, service.ErrNotFound):
			respondError(w, http.StatusNotFound, "user not found")
		default:
			log.Error().Err(err).Msg("update profile failed")
			respondError(w, http.StatusInternalServerError, "update failed")
		}
		return
	}

	respond(w, http.StatusOK, user)
}

func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.svc.ChangePassword(r.Context(), claims.UserID, req.CurrentPassword, req.NewPassword); err != nil {
		switch err {
		case service.ErrInvalidCredentials:
			respondError(w, http.StatusUnauthorized, "current password is incorrect")
		case service.ErrWeakPassword:
			respondError(w, http.StatusBadRequest, "password must be at least 8 characters")
		case service.ErrNotFound:
			respondError(w, http.StatusNotFound, "user not found")
		default:
			log.Error().Err(err).Msg("change password failed")
			respondError(w, http.StatusInternalServerError, "change failed")
		}
		return
	}

	respond(w, http.StatusNoContent, nil)
}

func (h *Handler) SearchAssets(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		respondError(w, http.StatusBadRequest, "query parameter required")
		return
	}

	assets, err := h.svc.SearchAssets(r.Context(), query)
	if err != nil {
		log.Error().Err(err).Msg("search assets failed")
		respondError(w, http.StatusInternalServerError, "search failed")
		return
	}

	respond(w, http.StatusOK, assets)
}

func (h *Handler) LookupAsset(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		respondError(w, http.StatusBadRequest, "query parameter required")
		return
	}

	results, err := h.svc.LookupAsset(r.Context(), query)
	if err != nil {
		log.Error().Err(err).Msg("lookup asset failed")
		respondError(w, http.StatusInternalServerError, "lookup failed")
		return
	}

	respond(w, http.StatusOK, results)
}

func (h *Handler) ListAssets(w http.ResponseWriter, r *http.Request) {
	assets, err := h.svc.ListAssets(r.Context())
	if err != nil {
		log.Error().Err(err).Msg("list assets failed")
		respondError(w, http.StatusInternalServerError, "list failed")
		return
	}
	respond(w, http.StatusOK, assets)
}

func (h *Handler) GetAssetMeta(w http.ResponseWriter, r *http.Request) {
	ticker := r.URL.Query().Get("ticker")
	if ticker == "" {
		respondError(w, http.StatusBadRequest, "ticker parameter required")
		return
	}

	meta, err := h.svc.GetAssetMeta(r.Context(), ticker)
	if err != nil {
		log.Error().Err(err).Str("ticker", ticker).Msg("asset meta failed")
		respondError(w, http.StatusInternalServerError, "meta lookup failed")
		return
	}

	respond(w, http.StatusOK, meta)
}

func (h *Handler) DeleteAsset(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	uid, err := parseUUID(id)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid asset id")
		return
	}

	err = h.svc.DeleteAsset(r.Context(), uid)
	if err != nil {
		if err == service.ErrAssetInUse {
			respondError(w, http.StatusConflict, "asset is used in transactions and cannot be deleted")
			return
		}
		log.Error().Err(err).Msg("delete asset failed")
		respondError(w, http.StatusInternalServerError, "delete failed")
		return
	}

	respond(w, http.StatusNoContent, nil)
}

func (h *Handler) GetAsset(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	uid, err := parseUUID(id)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid asset id")
		return
	}

	asset, err := h.svc.GetAsset(r.Context(), uid)
	if err != nil {
		respondError(w, http.StatusNotFound, "asset not found")
		return
	}

	respond(w, http.StatusOK, asset)
}

func (h *Handler) UpdateAsset(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	uid, err := parseUUID(id)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid asset id")
		return
	}

	var patch model.AssetPatch
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	asset, err := h.svc.UpdateAsset(r.Context(), uid, &patch)
	if err != nil {
		if err == service.ErrAssetNotFound {
			respondError(w, http.StatusNotFound, "asset not found")
			return
		}
		if err == service.ErrInvalidAssetClass {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err == service.ErrInvalidPriceSource {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		log.Error().Err(err).Msg("update asset failed")
		respondError(w, http.StatusInternalServerError, "update failed")
		return
	}

	respond(w, http.StatusOK, asset)
}

func (h *Handler) GetAssetQuote(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	uid, err := parseUUID(id)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid asset id")
		return
	}

	quote, err := h.svc.GetAssetQuote(r.Context(), uid)
	if err != nil {
		if err == service.ErrAssetNotFound {
			respondError(w, http.StatusNotFound, "asset not found")
			return
		}
		log.Error().Err(err).Msg("get asset quote failed")
		respondError(w, http.StatusInternalServerError, "quote failed")
		return
	}

	respond(w, http.StatusOK, quote)
}

func (h *Handler) AssetSplits(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	uid, err := parseUUID(id)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid asset id")
		return
	}

	splits, err := h.svc.AssetSplits(r.Context(), uid)
	if err != nil {
		if err == service.ErrAssetNotFound {
			respondError(w, http.StatusNotFound, "asset not found")
			return
		}
		log.Error().Err(err).Msg("get asset splits failed")
		respondError(w, http.StatusInternalServerError, "splits failed")
		return
	}

	respond(w, http.StatusOK, splits)
}

func (h *Handler) FetchAssetProfile(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	uid, err := parseUUID(id)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid asset id")
		return
	}

	asset, err := h.svc.FetchAssetProfile(r.Context(), uid)
	if err != nil {
		if err == service.ErrAssetNotFound {
			respondError(w, http.StatusNotFound, "asset not found")
			return
		}
		log.Error().Err(err).Msg("fetch asset profile failed")
		respondError(w, http.StatusBadGateway, "asset profile fetch failed")
		return
	}

	respond(w, http.StatusOK, asset)
}

func (h *Handler) GetAssetExposure(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	uid, err := parseUUID(id)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid asset id")
		return
	}

	exposure, err := h.svc.GetAssetExposure(r.Context(), uid)
	if err != nil {
		if err == service.ErrAssetNotFound {
			respondError(w, http.StatusNotFound, "asset not found")
			return
		}
		log.Error().Err(err).Msg("get asset exposure failed")
		respondError(w, http.StatusInternalServerError, "exposure failed")
		return
	}

	respond(w, http.StatusOK, exposure)
}

func (h *Handler) SaveAssetExposure(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	uid, err := parseUUID(id)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid asset id")
		return
	}

	var exposure model.AssetExposure
	if err := json.NewDecoder(r.Body).Decode(&exposure); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	saved, err := h.svc.SaveAssetExposure(r.Context(), uid, &exposure)
	if err != nil {
		switch {
		case err == service.ErrAssetNotFound:
			respondError(w, http.StatusNotFound, "asset not found")
		case err == service.ErrInvalidInput || err == service.ErrInvalidWeights:
			respondError(w, http.StatusBadRequest, err.Error())
		default:
			log.Error().Err(err).Msg("save asset exposure failed")
			respondError(w, http.StatusInternalServerError, "save failed")
		}
		return
	}

	respond(w, http.StatusOK, saved)
}

func (h *Handler) FetchAssetExposure(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	uid, err := parseUUID(id)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid asset id")
		return
	}

	exposure, err := h.svc.FetchAssetExposure(r.Context(), uid)
	if err != nil {
		if err == service.ErrAssetNotFound {
			respondError(w, http.StatusNotFound, "asset not found")
			return
		}
		log.Error().Err(err).Msg("fetch asset exposure failed")
		respondError(w, http.StatusBadGateway, "asset exposure fetch failed")
		return
	}

	respond(w, http.StatusOK, exposure)
}

func (h *Handler) FetchETFExposure(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	uid, err := parseUUID(id)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid asset id")
		return
	}

	exposure, err := h.svc.FetchETFExposure(r.Context(), uid, refreshRequested(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAssetNotFound), errors.Is(err, service.ErrNotFound):
			respondError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, service.ErrNotETF), errors.Is(err, service.ErrInvalidInput):
			respondError(w, http.StatusBadRequest, err.Error())
		default:
			log.Error().Err(err).Msg("fetch etf exposure failed")
			respondError(w, http.StatusBadGateway, "etf exposure fetch failed")
		}
		return
	}

	respond(w, http.StatusOK, exposure)
}

func (h *Handler) FetchMorningstarExposure(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	uid, err := parseUUID(id)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid asset id")
		return
	}

	exposure, err := h.svc.FetchMorningstarExposure(r.Context(), uid, refreshRequested(r))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAssetNotFound), errors.Is(err, service.ErrNotFound):
			respondError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, service.ErrNotETF), errors.Is(err, service.ErrInvalidInput):
			respondError(w, http.StatusBadRequest, err.Error())
		default:
			log.Error().Err(err).Msg("fetch morningstar exposure failed")
			respondError(w, http.StatusBadGateway, "morningstar exposure fetch failed")
		}
		return
	}

	respond(w, http.StatusOK, exposure)
}

// DeriveAssetRegions previews how a country weight distribution maps onto the
// canonical macro-regions for an asset. Nothing is persisted.
func (h *Handler) DeriveAssetRegions(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	uid, err := parseUUID(id)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid asset id")
		return
	}

	var req struct {
		Countries []model.ExposureRow `json:"countries"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	regions, err := h.svc.DeriveRegions(r.Context(), uid, req.Countries)
	if err != nil {
		if err == service.ErrAssetNotFound {
			respondError(w, http.StatusNotFound, "asset not found")
			return
		}
		log.Error().Err(err).Msg("derive asset regions failed")
		respondError(w, http.StatusInternalServerError, "derive failed")
		return
	}

	respond(w, http.StatusOK, map[string][]model.ExposureRow{"regions": regions})
}

func (h *Handler) BackfillAssetHistory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	uid, err := parseUUID(id)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid asset id")
		return
	}

	if err := h.svc.BackfillAssetHistory(r.Context(), uid); err != nil {
		if err == service.ErrAssetNotFound {
			respondError(w, http.StatusNotFound, "asset not found")
			return
		}
		log.Error().Err(err).Msg("backfill asset history failed")
		respondError(w, http.StatusInternalServerError, "backfill failed")
		return
	}

	respond(w, http.StatusOK, map[string]string{"status": "ok"})
}

// BackfillAssetMeta fills missing country/sector metadata for every stock asset
// and returns a report of the changes applied.
func (h *Handler) BackfillAssetMeta(w http.ResponseWriter, r *http.Request) {
	report, err := h.svc.BackfillAssetMeta(r.Context())
	if err != nil {
		log.Error().Err(err).Msg("backfill asset meta failed")
		respondError(w, http.StatusInternalServerError, "backfill failed")
		return
	}

	respond(w, http.StatusOK, report)
}

func (h *Handler) CreateAsset(w http.ResponseWriter, r *http.Request) {
	var asset model.Asset
	if err := json.NewDecoder(r.Body).Decode(&asset); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	created, err := h.svc.CreateAsset(r.Context(), &asset)
	if err != nil {
		var dup *service.AssetExistsError
		if errors.As(err, &dup) {
			respond(w, http.StatusConflict, map[string]interface{}{
				"error":    dup.Error(),
				"asset_id": dup.Existing.ID,
				// Same key as a successful create, so clients that read the
				// response body id keep working on reruns (201 vs 409).
				"id": dup.Existing.ID,
			})
			return
		}
		log.Error().Err(err).Msg("create asset failed")
		respondError(w, http.StatusInternalServerError, "create failed")
		return
	}

	respond(w, http.StatusCreated, created)
}
func (h *Handler) GetPriceHealth(w http.ResponseWriter, r *http.Request) {
	h.health.GetPriceHealth(w, r)
}
