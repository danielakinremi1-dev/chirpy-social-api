package api

import (
	"context"
	"net/http"

	"github.com/danielakinremi1-dev/chirpy-social-api/internal/auth"
	"github.com/google/uuid"
)

type contextKey int

const userIDKey contextKey = iota

// requireAuth rejects requests without a valid access token and stores the
// authenticated user's ID in the request context.
func (a *API) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, err := auth.GetBearerToken(r.Header)
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "Missing access token", err)
			return
		}
		userID, err := auth.ValidateJWT(token, a.jwtSecret)
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "Invalid access token", err)
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), userIDKey, userID)))
	}
}

// userIDFromContext returns the user ID stored by requireAuth.
func userIDFromContext(ctx context.Context) uuid.UUID {
	userID, _ := ctx.Value(userIDKey).(uuid.UUID)
	return userID
}

func (a *API) metricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}
