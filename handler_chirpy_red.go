package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/danielakinremi1-dev/chirpy/projects/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerChirpyRed(w http.ResponseWriter, req *http.Request) {

	type parameter struct {
		Event string `json:"event"`
		Data  struct {
			User_id string `json:"user_id"`
		} `json:"data"`
	}

	decoder := json.NewDecoder(req.Body)
	params := parameter{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error decoding request data", err)
		return
	}

	authorization, err := auth.GetAPIKey(req.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Error decoding authorization data", err)
		return
	} else if authorization != cfg.polka_key {
		respondWithError(w, http.StatusUnauthorized, "Invalid authorization key", err)
		return
	}

	if params.Event != "user.upgraded" {
		respondWithCode(w, http.StatusNoContent)
		return
	}

	userID, err := uuid.Parse(params.Data.User_id)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error decoding user ID", err)
		return
	}

	_, err = cfg.db.UpgradeUser(req.Context(), userID)
	if errors.Is(err, sql.ErrNoRows) {
		respondWithError(w, http.StatusNotFound, "User data not found", err)
		return
	} else if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error upgrading user to chirp red", err)
		return
	}

	respondWithCode(w, http.StatusNoContent)
}
