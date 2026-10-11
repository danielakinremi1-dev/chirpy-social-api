package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/danielakinremi1-dev/chirpy/projects/internal/auth"
	"github.com/danielakinremi1-dev/chirpy/projects/internal/database"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUpdateLogin(w http.ResponseWriter, req *http.Request) {

	type parameter struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}

	type returnVals struct {
		ID          uuid.UUID `json:"id"`
		Created_At  time.Time `json:"created_at"`
		Updated_At  time.Time `json:"updated_at"`
		Email       string    `json:"email"`
		IsChirpyRed bool      `json:"is_chirpy_red"`
	}

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

	decoder := json.NewDecoder(req.Body)
	params := parameter{}
	err = decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error decoding request data", err)
		return
	}

	params.Password, err = auth.HashPassword(params.Password)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error storing hashed user password", err)
		return
	}

	queryArgs := database.UpdateUserParams{
		Email:          params.Email,
		HashedPassword: params.Password,
		ID:             userID,
	}

	newUserData, err := cfg.db.UpdateUser(req.Context(), queryArgs)
	if errors.Is(err, sql.ErrNoRows) {
		respondWithError(w, http.StatusNotFound, "User data not found", err)
		return
	} else if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error updating user records", err)
		return
	}

	payload := returnVals{
		ID:          newUserData.ID,
		Created_At:  newUserData.CreatedAt,
		Updated_At:  newUserData.UpdatedAt,
		Email:       newUserData.Email,
		IsChirpyRed: newUserData.IsChirpyRed,
	}

	respondWithJSON(w, http.StatusOK, payload)
}
