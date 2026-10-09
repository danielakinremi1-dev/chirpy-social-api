package main

import (
	"net/http"
	"time"

	"github.com/danielakinremi1-dev/chirpy/projects/internal/auth"
)

func (cfg *apiConfig) handlerRefresh(w http.ResponseWriter, req *http.Request) {

	type returnVals struct {
		AccessToken string `json:"token"`
	}

	refreshToken, err := auth.GetBearerToken(req.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Missing refresh token", err)
		return
	}

	refreshTokenData, err := cfg.db.GetUserFromRefreshToken(req.Context(), refreshToken)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid refresh token", err)
		return
	} else if refreshTokenData.RevokedAt.Valid != false {
		respondWithError(w, http.StatusUnauthorized, "Revoked refresh token", err)
		return
	} else if refreshTokenData.ExpiresAt.Compare(time.Now()) <= 0 {
		respondWithError(w, http.StatusUnauthorized, "Expired refresh token", err)
		return
	}

	accessTokenString, err := auth.MakeJWT(refreshTokenData.UserID, cfg.secret)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating access token", err)
		return
	}

	payload := returnVals{
		AccessToken: accessTokenString,
	}

	respondWithJSON(w, http.StatusOK, payload)
}
