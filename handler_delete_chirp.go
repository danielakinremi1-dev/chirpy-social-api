package main

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/danielakinremi1-dev/chirpy-social-api/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerDeleteChirp(w http.ResponseWriter, req *http.Request) {

	bearerToken, err := auth.GetBearerToken(req.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Error parsing user token", err)
		return
	}
	userID, err := auth.ValidateJWT(bearerToken, cfg.secret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid access token", err)
		return
	}

	parsedID, err := uuid.Parse(req.PathValue("chirpID"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid chirp ID", err)
		return
	}

	chirpData, err := cfg.db.GetChirp(req.Context(), parsedID)
	if errors.Is(err, sql.ErrNoRows) {
		respondWithError(w, http.StatusNotFound, "Chirp ID not found", err)
		return
	} else if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error retrieving chirp", err)
		return
	}

	if chirpData.UserID != userID {
		respondWithError(w, http.StatusForbidden, "Invalid credentials to modify chirp", err)
		return
	}

	err = cfg.db.DeleteChirp(req.Context(), parsedID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error deleting Chirp", err)
		return
	}

	respondWithCode(w, http.StatusNoContent)
}
