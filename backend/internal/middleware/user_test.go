package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/alv67/peculium/internal/auth"
	"github.com/alv67/peculium/internal/model"
)

type fakeLoader struct {
	user *model.User
	err  error
}

func (f *fakeLoader) FindByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.user, nil
}

func requestWithClaims(role string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
	claims := &auth.Claims{UserID: uuid.New(), Email: "u@example.com", Role: role, TokenType: "access"}
	return r.WithContext(context.WithValue(r.Context(), auth.UserContextKey, claims))
}

func okBody(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func errorStatus(t *testing.T, rec *httptest.ResponseRecorder, wantMsg string) {
	t.Helper()
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error body %q: %v", rec.Body.String(), err)
	}
	if body["error"] != wantMsg {
		t.Fatalf("error = %q, want %q", body["error"], wantMsg)
	}
}

func TestRequireActiveUser_ActivePassesAndStoresUser(t *testing.T) {
	user := &model.User{ID: uuid.New(), Role: model.RoleEditor, Status: model.StatusActive}
	loader := &fakeLoader{user: user}

	var stored *model.User
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stored = UserFromContext(r.Context())
		okBody(w, r)
	})

	rec := httptest.NewRecorder()
	RequireActiveUser(loader)(next).ServeHTTP(rec, requestWithClaims(string(model.RoleEditor)))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if stored == nil || stored.ID != user.ID {
		t.Fatalf("loaded user = %v, want it stored in the request context", stored)
	}
}

func TestRequireActiveUser_PendingForbidden(t *testing.T) {
	loader := &fakeLoader{user: &model.User{ID: uuid.New(), Status: model.StatusPending}}
	rec := httptest.NewRecorder()
	RequireActiveUser(loader)(http.HandlerFunc(okBody)).ServeHTTP(rec, requestWithClaims("viewer"))

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
	errorStatus(t, rec, "account pending approval")
}

func TestRequireActiveUser_DisabledForbidden(t *testing.T) {
	loader := &fakeLoader{user: &model.User{ID: uuid.New(), Status: model.StatusDisabled}}
	rec := httptest.NewRecorder()
	RequireActiveUser(loader)(http.HandlerFunc(okBody)).ServeHTTP(rec, requestWithClaims("viewer"))

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
	errorStatus(t, rec, "account disabled")
}

func TestRequireActiveUser_MissingUserUnauthorized(t *testing.T) {
	loader := &fakeLoader{err: errors.New("no rows")}
	rec := httptest.NewRecorder()
	RequireActiveUser(loader)(http.HandlerFunc(okBody)).ServeHTTP(rec, requestWithClaims("viewer"))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestRequireActiveUser_NoClaimsUnauthorized(t *testing.T) {
	loader := &fakeLoader{user: &model.User{Status: model.StatusActive}}
	w := httptest.NewRecorder()
	RequireActiveUser(loader)(http.HandlerFunc(okBody)).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 without claims", w.Code)
	}
}

func TestRequireAdmin_AdminRolesPassOnLoadedUser(t *testing.T) {
	for _, role := range []model.Role{model.RoleAdmin, model.RoleOwner} {
		loader := &fakeLoader{user: &model.User{ID: uuid.New(), Role: role, Status: model.StatusActive}}
		rec := httptest.NewRecorder()
		h := RequireActiveUser(loader)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			RequireAdmin(http.HandlerFunc(okBody)).ServeHTTP(w, r)
		}))
		h.ServeHTTP(rec, requestWithClaims(string(role)))
		if rec.Code != http.StatusOK {
			t.Fatalf("role %q: status = %d, want 200", role, rec.Code)
		}
	}
}

func TestRequireAdmin_NonAdminForbidden(t *testing.T) {
	for _, role := range []model.Role{model.RoleEditor, model.RoleViewer} {
		loader := &fakeLoader{user: &model.User{ID: uuid.New(), Role: role, Status: model.StatusActive}}
		rec := httptest.NewRecorder()
		h := RequireActiveUser(loader)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			RequireAdmin(http.HandlerFunc(okBody)).ServeHTTP(w, r)
		}))
		h.ServeHTTP(rec, requestWithClaims(string(role)))
		if rec.Code != http.StatusForbidden {
			t.Fatalf("role %q: status = %d, want 403", role, rec.Code)
		}
		errorStatus(t, rec, "admin access required")
	}
}

func TestRequireAdmin_RoleChangeTakesEffectOnExistingToken(t *testing.T) {
	// The token still claims admin, but the stored account was demoted: the
	// middleware must trust the loaded user, not the stale claim.
	loader := &fakeLoader{user: &model.User{ID: uuid.New(), Role: model.RoleViewer, Status: model.StatusActive}}
	rec := httptest.NewRecorder()
	h := RequireActiveUser(loader)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		RequireAdmin(http.HandlerFunc(okBody)).ServeHTTP(w, r)
	}))
	h.ServeHTTP(rec, requestWithClaims(string(model.RoleAdmin)))

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 for a demoted token holder", rec.Code)
	}
}

func TestRequireAdmin_FallsBackToClaims(t *testing.T) {
	rec := httptest.NewRecorder()
	RequireAdmin(http.HandlerFunc(okBody)).ServeHTTP(rec, requestWithClaims(string(model.RoleOwner)))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 via claim fallback", rec.Code)
	}

	rec = httptest.NewRecorder()
	RequireAdmin(http.HandlerFunc(okBody)).ServeHTTP(rec, requestWithClaims("editor"))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}
