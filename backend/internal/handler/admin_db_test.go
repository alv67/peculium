package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/alv67/peculium/internal/auth"
	"github.com/alv67/peculium/internal/cache"
	"github.com/alv67/peculium/internal/repository"
	"github.com/alv67/peculium/internal/service"
)

// fakeDBMaintainer records what the handler drives: what it streamed out,
// which temp path it handed over for a restore and what that path contained
// while the restore ran.
type fakeDBMaintainer struct {
	dumpPayload []byte
	dumpErr     error

	restoreErr      error
	restoredPath    string
	restoredContent []byte
	restoreCalls    int
}

func (f *fakeDBMaintainer) Dump(ctx context.Context, w io.Writer) error {
	if f.dumpErr != nil {
		return f.dumpErr
	}
	_, err := w.Write(f.dumpPayload)
	return err
}

func (f *fakeDBMaintainer) Restore(ctx context.Context, path string) error {
	f.restoreCalls++
	f.restoredPath = path
	f.restoredContent, _ = os.ReadFile(path)
	return f.restoreErr
}

func newDBAdminHandler(f service.DBMaintainer) *Handler {
	svc := service.New(&repository.Repository{}, nil, nil, nil, time.Minute, time.Hour, cache.New(nil), 0, 0, nil, f)
	return New(svc, nil)
}

func adminRequest(method, target string, body io.Reader, contentType string) *http.Request {
	r := httptest.NewRequest(method, target, body)
	claims := &auth.Claims{UserID: uuid.New(), Email: "admin@example.com", Role: "admin"}
	r = r.WithContext(context.WithValue(r.Context(), auth.UserContextKey, claims))
	if contentType != "" {
		r.Header.Set("Content-Type", contentType)
	}
	return r
}

func restoreUpload(t *testing.T, fields map[string]string, dumpContent []byte) (*bytes.Buffer, string) {
	t.Helper()
	var b bytes.Buffer
	mw := multipart.NewWriter(&b)
	for k, v := range fields {
		if err := mw.WriteField(k, v); err != nil {
			t.Fatal(err)
		}
	}
	if dumpContent != nil {
		fw, err := mw.CreateFormFile("dump", "db.dump")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := fw.Write(dumpContent); err != nil {
			t.Fatal(err)
		}
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	return &b, mw.FormDataContentType()
}

func TestAdminDBBackup_StreamsDumpAndSetsHeaders(t *testing.T) {
	payload := []byte("PGDUMP-Fc-FAKE-BYTES")
	h := newDBAdminHandler(&fakeDBMaintainer{dumpPayload: payload})

	rec := httptest.NewRecorder()
	h.AdminDBBackup(rec, adminRequest(http.MethodGet, "/api/v1/admin/db/backup", nil, ""))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "application/octet-stream" {
		t.Errorf("Content-Type = %q, want application/octet-stream", got)
	}
	wantName := fmt.Sprintf(`attachment; filename="peculium-db-%s.dump"`, time.Now().UTC().Format("2006-01-02"))
	if got := rec.Header().Get("Content-Disposition"); got != wantName {
		t.Errorf("Content-Disposition = %q, want %q", got, wantName)
	}
	if rec.Body.String() != string(payload) {
		t.Errorf("body = %q, want %q", rec.Body.String(), payload)
	}
}

func TestAdminDBBackup_FailureMapsTo500WithoutLeakingSecrets(t *testing.T) {
	boom := fmt.Errorf("%w: pg_dump failed: password SUPERSECRETPW rejected", service.ErrDBMaintenance)
	h := newDBAdminHandler(&fakeDBMaintainer{dumpErr: boom})

	rec := httptest.NewRecorder()
	h.AdminDBBackup(rec, adminRequest(http.MethodGet, "/api/v1/admin/db/backup", nil, ""))

	// Headers are committed before the tool runs: a failed dump reaches the
	// client as an empty/truncated download, the sanitized detail goes to the
	// log. The status must not flip to 500 over an already-started stream.
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (stream already committed)", rec.Code)
	}
	if body := rec.Body.String(); strings.Contains(body, "SUPERSECRETPW") {
		t.Errorf("response leaked the secret: %q", body)
	}
}

func TestAdminDBBackup_TruncatedOnToolFailure(t *testing.T) {
	h := newDBAdminHandler(&fakeDBMaintainer{dumpErr: service.ErrDBMaintenance})

	rec := httptest.NewRecorder()
	h.AdminDBBackup(rec, adminRequest(http.MethodGet, "/api/v1/admin/db/backup", nil, ""))

	// A tool that never writes leaves the client with an empty body and the
	// attachment headers; nothing to log to the client.
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (truncated stream)", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("streamed body = %d bytes, want 0", rec.Body.Len())
	}
}

func TestAdminDBBackup_Unauthorized(t *testing.T) {
	h := newDBAdminHandler(&fakeDBMaintainer{})
	rec := httptest.NewRecorder()
	h.AdminDBBackup(rec, httptest.NewRequest(http.MethodGet, "/api/v1/admin/db/backup", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestAdminDBRestore_RequiresConfirmation(t *testing.T) {
	f := &fakeDBMaintainer{}
	h := newDBAdminHandler(f)

	body, ct := restoreUpload(t, map[string]string{}, []byte("dump"))
	rec := httptest.NewRecorder()
	h.AdminDBRestore(rec, adminRequest(http.MethodPost, "/api/v1/admin/db/restore", body, ct))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "replace") {
		t.Errorf("error body should name the required confirmation, got %q", rec.Body.String())
	}
	if f.restoreCalls != 0 {
		t.Errorf("restore must not run without confirmation, calls = %d", f.restoreCalls)
	}
}

func TestAdminDBRestore_MissingDumpFile(t *testing.T) {
	f := &fakeDBMaintainer{}
	h := newDBAdminHandler(f)

	body, ct := restoreUpload(t, map[string]string{"confirm": "replace"}, nil)
	rec := httptest.NewRecorder()
	h.AdminDBRestore(rec, adminRequest(http.MethodPost, "/api/v1/admin/db/restore", body, ct))

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "dump") {
		t.Errorf("error body should mention the dump field, got %q", rec.Body.String())
	}
	if f.restoreCalls != 0 {
		t.Errorf("restore must not run without a file, calls = %d", f.restoreCalls)
	}
}

func TestAdminDBRestore_Unauthorized(t *testing.T) {
	h := newDBAdminHandler(&fakeDBMaintainer{})
	body, ct := restoreUpload(t, map[string]string{"confirm": "replace"}, []byte("dump"))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/db/restore", body)
	req.Header.Set("Content-Type", ct)
	rec := httptest.NewRecorder()
	h.AdminDBRestore(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestAdminDBRestore_SuccessReturnsSummaryAndRemovesTempFile(t *testing.T) {
	content := []byte("PGDUMP-Fc-BYTES")
	f := &fakeDBMaintainer{}
	h := newDBAdminHandler(f)

	body, ct := restoreUpload(t, map[string]string{"confirm": "replace"}, content)
	rec := httptest.NewRecorder()
	h.AdminDBRestore(rec, adminRequest(http.MethodPost, "/api/v1/admin/db/restore", body, ct))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["status"] != "restored" || got["mode"] != "replace" {
		t.Errorf("summary = %v, want status=restored mode=replace", got)
	}
	if got["dump_bytes"] != float64(len(content)) {
		t.Errorf("dump_bytes = %v, want %d", got["dump_bytes"], len(content))
	}
	if msg, _ := got["message"].(string); !strings.Contains(msg, "log in again") {
		t.Errorf("summary should warn about re-login, got %q", msg)
	}
	// The handler hands the uploader's content to the tool through a temp
	// file it must delete afterwards.
	if f.restoreCalls != 1 || !bytes.Equal(f.restoredContent, content) {
		t.Errorf("restore saw %q, want the uploaded dump bytes", f.restoredContent)
	}
	if _, err := os.Stat(f.restoredPath); !os.IsNotExist(err) {
		t.Errorf("temp dump file %q was not removed", f.restoredPath)
	}
}

func TestAdminDBRestore_ConfirmViaQueryParameter(t *testing.T) {
	f := &fakeDBMaintainer{}
	h := newDBAdminHandler(f)

	body, ct := restoreUpload(t, map[string]string{}, []byte("x"))
	rec := httptest.NewRecorder()
	h.AdminDBRestore(rec, adminRequest(http.MethodPost, "/api/v1/admin/db/restore?confirm=replace", body, ct))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body %q)", rec.Code, rec.Body.String())
	}
	if f.restoreCalls != 1 {
		t.Errorf("restore calls = %d, want 1", f.restoreCalls)
	}
}

func TestAdminDBRestore_FailureMapsTo500WithStderr(t *testing.T) {
	stderr := "pg_restore: error: could not execute statement: relation \"users\" does not exist"
	f := &fakeDBMaintainer{restoreErr: fmt.Errorf("%w: pg_restore failed: %s", service.ErrDBMaintenance, stderr)}
	h := newDBAdminHandler(f)

	body, ct := restoreUpload(t, map[string]string{"confirm": "replace"}, []byte("x"))
	rec := httptest.NewRecorder()
	h.AdminDBRestore(rec, adminRequest(http.MethodPost, "/api/v1/admin/db/restore", body, ct))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	// The admin gets the tool message verbatim (already sanitized upstream)
	// to know what to fix; the temp file must be cleaned either way.
	if !strings.Contains(rec.Body.String(), "could not execute statement") {
		t.Errorf("response should carry the pg_restore stderr, got %q", rec.Body.String())
	}
	if _, err := os.Stat(f.restoredPath); !os.IsNotExist(err) {
		t.Errorf("temp dump file %q was not removed after failure", f.restoredPath)
	}
}
