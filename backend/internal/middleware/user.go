package middleware

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/alv67/peculium/internal/auth"
	"github.com/alv67/peculium/internal/model"
)

// UserLoader is the subset of the user repository the middleware needs.
type UserLoader interface {
	FindByID(ctx context.Context, id uuid.UUID) (*model.User, error)
}

type contextKey string

const currentUserKey contextKey = "current_user"

func errorJSON(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// RequireActiveUser loads the account behind the JWT claims on every request
// and only lets it through while the stored status is active. Revoking or
// approving an account therefore takes effect immediately, even for tokens
// that were issued earlier. Chain it after the JWT middleware.
func RequireActiveUser(loader UserLoader) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := auth.GetClaims(r.Context())
			if claims == nil {
				errorJSON(w, http.StatusUnauthorized, "unauthorized")
				return
			}

			user, err := loader.FindByID(r.Context(), claims.UserID)
			if err != nil || user == nil {
				errorJSON(w, http.StatusUnauthorized, "user not found")
				return
			}
			if user.Status != model.StatusActive {
				if user.Status == model.StatusPending {
					errorJSON(w, http.StatusForbidden, "account pending approval")
				} else {
					errorJSON(w, http.StatusForbidden, "account disabled")
				}
				return
			}

			ctx := context.WithValue(r.Context(), currentUserKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserFromContext returns the user loaded by RequireActiveUser, or nil.
func UserFromContext(ctx context.Context) *model.User {
	user, ok := ctx.Value(currentUserKey).(*model.User)
	if !ok {
		return nil
	}
	return user
}

// RequireAdmin allows only users whose role is owner or admin. It reads the
// role of the user loaded by RequireActiveUser so role changes apply to
// existing tokens immediately; when no loaded user is present it falls back
// to the token claim. Chain it after the JWT middleware.
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if user := UserFromContext(r.Context()); user != nil {
			if user.Role.IsAdmin() {
				next.ServeHTTP(w, r)
				return
			}
			errorJSON(w, http.StatusForbidden, "admin access required")
			return
		}

		claims := auth.GetClaims(r.Context())
		if claims == nil {
			errorJSON(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		if model.Role(claims.Role).IsAdmin() {
			next.ServeHTTP(w, r)
			return
		}
		errorJSON(w, http.StatusForbidden, "admin access required")
	})
}
