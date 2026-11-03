package api

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/danielakinremi1-dev/chirpy-social-api/internal/auth"
	"github.com/danielakinremi1-dev/chirpy-social-api/internal/database"
	"github.com/google/uuid"
)

const (
	accessTokenTTL  = time.Hour
	refreshTokenTTL = 60 * 24 * time.Hour
)

var dummyPasswordHash, _ = auth.HashPassword("chirpy-dummy-password")

var errInvalidRefreshToken = errors.New("invalid refresh token")

func (a *API) handleLogin(w http.ResponseWriter, r *http.Request) {
	type response struct {
		userResponse
		Token        string `json:"token"`
		RefreshToken string `json:"refresh_token"`
	}

	var params credentials
	if !decodeJSON(w, r, &params) {
		return
	}

	user, err := a.store.GetUser(r.Context(), params.Email)
	if errors.Is(err, sql.ErrNoRows) {
		// Hash anyway so unknown emails take as long as wrong passwords.
		auth.CheckPasswordHash(params.Password, dummyPasswordHash)
		respondWithError(w, http.StatusUnauthorized, "Incorrect email or password", err)
		return
	} else if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error retrieving user", err)
		return
	}

	match, err := auth.CheckPasswordHash(params.Password, user.HashedPassword)
	if err != nil || !match {
		respondWithError(w, http.StatusUnauthorized, "Incorrect email or password", err)
		return
	}

	accessToken, err := auth.MakeJWT(user.ID, a.jwtSecret, accessTokenTTL)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating access token", err)
		return
	}

	refreshToken, err := createRefreshToken(r.Context(), a.store, user.ID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating refresh token", err)
		return
	}

	respondWithJSON(w, http.StatusOK, response{
		userResponse: toUserResponse(user),
		Token:        accessToken,
		RefreshToken: refreshToken,
	})
}

// handleRefresh exchanges a refresh token for a new access token and a new
// refresh token, revoking the one presented.
func (a *API) handleRefresh(w http.ResponseWriter, r *http.Request) {
	type response struct {
		Token        string `json:"token"`
		RefreshToken string `json:"refresh_token"`
	}

	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Missing refresh token", err)
		return
	}

	var resp response
	err = a.store.InTx(r.Context(), func(q database.Querier) error {
		tokenData, err := q.GetUserFromRefreshToken(r.Context(), refreshToken)
		if errors.Is(err, sql.ErrNoRows) {
			return errInvalidRefreshToken
		} else if err != nil {
			return err
		}
		if tokenData.RevokedAt.Valid || !tokenData.ExpiresAt.After(time.Now()) {
			return errInvalidRefreshToken
		}

		// The revoke only matches unrevoked tokens, so if two requests race
		// with the same token, only one of them can rotate it.
		revoked, err := q.RevokeRefreshToken(r.Context(), refreshToken)
		if err != nil {
			return err
		} else if revoked == 0 {
			return errInvalidRefreshToken
		}

		resp.RefreshToken, err = createRefreshToken(r.Context(), q, tokenData.UserID)
		if err != nil {
			return err
		}
		resp.Token, err = auth.MakeJWT(tokenData.UserID, a.jwtSecret, accessTokenTTL)
		return err
	})
	if errors.Is(err, errInvalidRefreshToken) {
		respondWithError(w, http.StatusUnauthorized, "Invalid refresh token", err)
		return
	} else if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error refreshing tokens", err)
		return
	}

	respondWithJSON(w, http.StatusOK, resp)
}

func (a *API) handleRevoke(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Missing refresh token", err)
		return
	}

	revoked, err := a.store.RevokeRefreshToken(r.Context(), refreshToken)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error revoking refresh token", err)
		return
	} else if revoked == 0 {
		respondWithError(w, http.StatusUnauthorized, "Invalid refresh token", nil)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func createRefreshToken(ctx context.Context, q database.Querier, userID uuid.UUID) (string, error) {
	token, err := auth.MakeRefreshToken()
	if err != nil {
		return "", err
	}

	tokenData, err := q.CreateRefresh(ctx, database.CreateRefreshParams{
		Token:     token,
		UserID:    userID,
		ExpiresAt: time.Now().UTC().Add(refreshTokenTTL),
	})
	if err != nil {
		return "", err
	}

	return tokenData.Token, nil
}
