package handler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/alv67/peculium/internal/auth"
	"github.com/alv67/peculium/internal/service"
)

const (
	// dbMaintenanceTimeout bounds a dump or restore end to end. It replaces
	// the router-wide 30s request timeout, which would kill an operation on
	// a real database long before it can finish.
	dbMaintenanceTimeout = 30 * time.Minute
	// dbRestoreFormMemory keeps at most this many bytes of the multipart
	// body in memory; the uploaded dump spills to the OS temp area beyond it
	// and the handler copies it out to its own temp file anyway.
	dbRestoreFormMemory = 8 << 20
)

// AdminDBBackup streams a full pg_dump (custom format -Fc) archive of the
// application database as an attachment. The dump is piped straight from the
// tool into the response, never buffered whole in memory.
func (h *Handler) AdminDBBackup(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	rc := http.NewResponseController(w)
	_ = rc.SetWriteDeadline(time.Now().Add(dbMaintenanceTimeout))
	ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), dbMaintenanceTimeout)
	defer cancel()

	stamp := time.Now().UTC().Format("2006-01-02")
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="peculium-db-%s.dump"`, stamp))

	if err := h.svc.BackupDatabase(ctx, w); err != nil {
		log.Error().Err(err).Msg("admin db backup failed mid-stream")
	}
}

// AdminDBRestore replaces the whole database with an uploaded custom-format
// dump. The request is multipart: the archive in the "dump" file field and
// the destructive confirmation confirm=replace either as a form field or as
// a query parameter. The operation rewrites every table from the archive,
// users included, so existing sessions may reference rows that are gone: the
// summary tells the admin to log in again.
func (h *Handler) AdminDBRestore(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetClaims(r.Context())
	if claims == nil {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	rc := http.NewResponseController(w)
	_ = rc.SetReadDeadline(time.Now().Add(dbMaintenanceTimeout))
	_ = rc.SetWriteDeadline(time.Now().Add(dbMaintenanceTimeout))

	if err := r.ParseMultipartForm(dbRestoreFormMemory); err != nil {
		respondError(w, http.StatusBadRequest, "invalid multipart form: a dump file uploaded in the \"dump\" field is required")
		return
	}
	defer func() {
		if r.MultipartForm != nil {
			r.MultipartForm.RemoveAll()
		}
	}()

	// FormValue consults the multipart fields first and the URL query after,
	// so one check accepts the confirmation in either place.
	confirm := r.FormValue("confirm")

	files := r.MultipartForm.File["dump"]
	if len(files) == 0 {
		respondError(w, http.StatusBadRequest, `missing dump file: upload the archive in the "dump" form field`)
		return
	}

	dumpPath, size, err := saveMultipartTemp(files[0])
	if err != nil {
		log.Error().Err(err).Msg("admin db restore: could not store uploaded dump")
		respondError(w, http.StatusInternalServerError, "could not read the uploaded dump")
		return
	}
	defer os.Remove(dumpPath)

	ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), dbMaintenanceTimeout)
	defer cancel()

	if err := h.svc.RestoreDatabase(ctx, dumpPath, confirm); err != nil {
		if errors.Is(err, service.ErrInvalidInput) {
			respondError(w, http.StatusBadRequest, err.Error())
			return
		}
		// The pg_restore stderr is sanitized (credentials masked) and kept
		// in the response on purpose: a partially applied restore needs the
		// admin to read what failed, not to hunt the server logs.
		log.Error().Err(err).Msg("admin db restore failed")
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respond(w, http.StatusOK, map[string]any{
		"status":     "restored",
		"mode":       service.DBRestoreConfirm,
		"dump_bytes": size,
		"message":    "the database was replaced from the dump; the users table is part of the archive, so current sessions and tokens may no longer be valid — log in again",
	})
}

// saveMultipartTemp copies an uploaded file into a fresh temp file and
// returns its path and size; the caller owns the file and must remove it.
func saveMultipartTemp(fh *multipart.FileHeader) (string, int64, error) {
	src, err := fh.Open()
	if err != nil {
		return "", 0, err
	}
	defer src.Close()

	dst, err := os.CreateTemp("", "peculium-restore-*.dump")
	if err != nil {
		return "", 0, err
	}
	size, err := io.Copy(dst, src)
	if cerr := dst.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(dst.Name())
		return "", 0, err
	}
	return dst.Name(), size, nil
}
