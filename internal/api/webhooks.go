package api

import (
	"crypto/subtle"
	"database/sql"
	"errors"
	"net/http"

	"github.com/danielakinremi1-dev/chirpy-social-api/internal/auth"
	"github.com/google/uuid"
)

func (a *API) handlePolkaWebhook(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Event string `json:"event"`
		Data  struct {
			UserID string `json:"user_id"`
		} `json:"data"`
	}

	apiKey, err := auth.GetAPIKey(r.Header)
	if err != nil {
		respondWithError(w, r, http.StatusUnauthorized, "Missing API key", err)
		return
	} else if subtle.ConstantTimeCompare([]byte(apiKey), []byte(a.polkaKey)) != 1 {
		respondWithError(w, r, http.StatusUnauthorized, "Invalid API key", nil)
		return
	}

	var params parameters
	if !decodeJSON(w, r, &params) {
		return
	}

	if params.Event != "user.upgraded" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	userID, err := uuid.Parse(params.Data.UserID)
	if err != nil {
		respondWithError(w, r, http.StatusBadRequest, "Invalid user ID", err)
		return
	}

	_, err = a.store.UpgradeUser(r.Context(), userID)
	if errors.Is(err, sql.ErrNoRows) {
		respondWithError(w, r, http.StatusNotFound, "User not found", err)
		return
	} else if err != nil {
		respondWithError(w, r, http.StatusInternalServerError, "Error upgrading user", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
