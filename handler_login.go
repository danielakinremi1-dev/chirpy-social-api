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

func (cfg *apiConfig) handlerLogin(w http.ResponseWriter, req *http.Request) {

	type parameter struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}

	type returnVals struct {
		ID           uuid.UUID `json:"id"`
		Created_At   time.Time `json:"created_at"`
		Updated_At   time.Time `json:"updated_at"`
		Email        string    `json:"email"`
		IsChirpyRed  bool      `json:"is_chirpy_red"`
		AccessToken  string    `json:"token"`
		RefreshToken string    `json:"refresh_token"`
	}

	decoder := json.NewDecoder(req.Body)
	params := parameter{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error decoding request data", err)
		return
	}

	userData, err := cfg.db.GetUser(req.Context(), params.Email)
	if errors.Is(err, sql.ErrNoRows) {
		respondWithError(w, http.StatusNotFound, "User email not found", err)
		return
	} else if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error retrieving user info", err)
		return
	}

	match, err := auth.CheckPasswordHash(params.Password, userData.HashedPassword)
	if err != nil || !match {
		respondWithError(w, http.StatusUnauthorized, "Incorrect password", err)
		return
	}

	accessTokenString, err := auth.MakeJWT(userData.ID, cfg.secret)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating access token", err)
		return
	}

	queryArgs := database.CreateRefreshParams{
		Token:     auth.MakeRefreshToken(),
		UserID:    userData.ID,
		ExpiresAt: time.Now().UTC().Add(time.Hour * 1440),
	}

	refreshTokenData, err := cfg.db.CreateRefresh(req.Context(), queryArgs)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating refresh token", err)
		return
	}

	payload := returnVals{
		ID:           userData.ID,
		Created_At:   userData.CreatedAt,
		Updated_At:   userData.UpdatedAt,
		Email:        userData.Email,
		IsChirpyRed:  userData.IsChirpyRed,
		AccessToken:  accessTokenString,
		RefreshToken: refreshTokenData.Token,
	}

	respondWithJSON(w, http.StatusOK, payload)
}
