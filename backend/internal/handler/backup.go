package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/alv67/peculium/internal/auth"
	"github.com/alv67/peculium/internal/model"
	"github.com/alv67/peculium/internal/service"
)

// DownloadBackup streams the whole data of the authenticated user as a
// single versioned JSON bundle: the portfolios the user owns with their
// transactions, the referenced assets' metadata, their exposure and manual
// prices, the base currency and the supported currencies in use. Yahoo
// prices, FX history and health events are excluded: the providers refill
// them on the restored server.
func (h *Handler) DownloadBackup(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	doc, err := h.svc.ExportUserBackup(r.Context(), claims.UserID)
	if err != nil {
		log.Error().Err(err).Msg("backup export failed")
		respondError(w, http.StatusInternalServerError, "backup failed")
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, backupFilename(claims.Email, doc.ExportedAt)))
	respond(w, http.StatusOK, doc)
}

// RestoreUserBackup rebuilds the authenticated user's data from a bundle
// produced by DownloadBackup. The body is the bundle JSON itself; the mode
// query parameter chooses the strategy: "add" (the default) imports every
// portfolio as a new one without touching existing data, "replace" deletes
// the user's portfolios first (CASCADE removes their transactions) and then
// imports. Assets are global and never deleted. "replace" asks no
// server-side confirmation: the UI confirms client-side before calling.
func (h *Handler) RestoreUserBackup(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var doc model.UserBackup
	if err := json.NewDecoder(r.Body).Decode(&doc); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	summary, err := h.svc.RestoreUserBackup(r.Context(), claims.UserID, &doc, r.URL.Query().Get("mode"))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidInput):
			respondError(w, http.StatusBadRequest, err.Error())
		default:
			log.Error().Err(err).Msg("backup restore failed")
			respondError(w, http.StatusInternalServerError, "restore failed")
		}
		return
	}

	respond(w, http.StatusOK, summary)
}

// backupFilename builds a Content-Disposition-safe download name of the
// form peculium-backup-<email>-<date>.json, dropping everything outside the
// safe character set so no quote or control character can reach the header.
func backupFilename(email string, at time.Time) string {
	var b strings.Builder
	for _, r := range strings.ToLower(email) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '.', r == '+', r == '_', r == '-', r == '@':
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		b.WriteString("user")
	}
	return "peculium-backup-" + b.String() + "-" + at.Format("2006-01-02") + ".json"
}
