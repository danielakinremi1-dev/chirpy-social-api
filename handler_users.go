package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/danielakinremi1-dev/chirpy-social-api/internal/auth"
	"github.com/danielakinremi1-dev/chirpy-social-api/internal/database"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUsers(w http.ResponseWriter, req *http.Request) {

	type parameter struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	type returnVals struct {
		ID          uuid.UUID `json:"id"`
		Created_At  time.Time `json:"created_at"`
		Updated_At  time.Time `json:"updated_at"`
		Email       string    `json:"email"`
		IsChirpyRed bool      `json:"is_chirpy_red"`
	}

	decoder := json.NewDecoder(req.Body)
	params := parameter{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if err := validateEmail(params.Email); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid email address", err)
		return
	}
	if err := validatePassword(params.Password); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error(), err)
		return
	}

	hashPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error storing hashed user password", err)
		return
	}

	queryArgs := database.CreateUserParams{
		Email:          params.Email,
		HashedPassword: hashPassword,
	}

	userData, err := cfg.db.CreateUser(req.Context(), queryArgs)
	if isUniqueViolation(err) {
		respondWithError(w, http.StatusConflict, "Email is already registered", err)
		return
	} else if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating user", err)
		return
	}

	payload := returnVals{
		ID:          userData.ID,
		Created_At:  userData.CreatedAt,
		Updated_At:  userData.UpdatedAt,
		Email:       userData.Email,
		IsChirpyRed: userData.IsChirpyRed,
	}

	respondWithJSON(w, http.StatusCreated, payload)
}
