package main

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

func (cfg *apiConfig) handlerRefresh(w http.ResponseWriter, req *http.Request) {

	type returnVals struct {
		AccessToken  string `json:"token"`
		RefreshToken string `json:"refresh_token"`
	}

	refreshToken, err := auth.GetBearerToken(req.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Missing refresh token", err)
		return
	}

	tx, err := cfg.sqlDB.BeginTx(req.Context(), nil)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error starting transaction", err)
		return
	}
	defer tx.Rollback()
	qtx := cfg.db.WithTx(tx)

	refreshTokenData, err := qtx.GetUserFromRefreshToken(req.Context(), refreshToken)
	if errors.Is(err, sql.ErrNoRows) {
		respondWithError(w, http.StatusUnauthorized, "Invalid refresh token", err)
		return
	} else if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error retrieving refresh token", err)
		return
	} else if refreshTokenData.RevokedAt.Valid {
		respondWithError(w, http.StatusUnauthorized, "Revoked refresh token", nil)
		return
	} else if refreshTokenData.ExpiresAt.Compare(time.Now()) <= 0 {
		respondWithError(w, http.StatusUnauthorized, "Expired refresh token", nil)
		return
	}

	// The revoke only matches unrevoked tokens, so if two requests race with
	// the same token, only one of them can rotate it.
	revoked, err := qtx.RevokeRefreshToken(req.Context(), refreshToken)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error revoking refresh token", err)
		return
	} else if revoked == 0 {
		respondWithError(w, http.StatusUnauthorized, "Revoked refresh token", nil)
		return
	}

	newRefreshToken, err := createRefreshToken(req.Context(), qtx, refreshTokenData.UserID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating refresh token", err)
		return
	}

	accessTokenString, err := auth.MakeJWT(refreshTokenData.UserID, cfg.secret, accessTokenTTL)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating access token", err)
		return
	}

	if err := tx.Commit(); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error rotating refresh token", err)
		return
	}

	payload := returnVals{
		AccessToken:  accessTokenString,
		RefreshToken: newRefreshToken,
	}

	respondWithJSON(w, http.StatusOK, payload)
}

func createRefreshToken(ctx context.Context, db *database.Queries, userID uuid.UUID) (string, error) {
	token, err := auth.MakeRefreshToken()
	if err != nil {
		return "", err
	}

	refreshTokenData, err := db.CreateRefresh(ctx, database.CreateRefreshParams{
		Token:     token,
		UserID:    userID,
		ExpiresAt: time.Now().UTC().Add(refreshTokenTTL),
	})
	if err != nil {
		return "", err
	}

	return refreshTokenData.Token, nil
}
