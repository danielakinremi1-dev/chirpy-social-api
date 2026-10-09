package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/danielakinremi1-dev/chirpy/projects/internal/auth"
	"github.com/danielakinremi1-dev/chirpy/projects/internal/database"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUsers(w http.ResponseWriter, req *http.Request) {

	type parameter struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	type returnVals struct {
		ID         uuid.UUID `json:"id"`
		Created_At time.Time `json:"created_at"`
		Updated_At time.Time `json:"updated_at"`
		Email      string    `json:"email"`
	}

	decoder := json.NewDecoder(req.Body)
	params := parameter{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error decoding request data", err)
		return
	}

	if len(params.Password) < 1 {
		respondWithError(w, http.StatusBadRequest, "No password provided for user setup", fmt.Errorf("Missing password"))
		return
	}

	hashPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error encoding user password", err)
		return
	}

	queryArgs := database.CreateUserParams{
		Email:          params.Email,
		HashedPassword: hashPassword,
	}

	userData, err := cfg.db.CreateUser(req.Context(), queryArgs)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating user", err)
		return
	}

	payload := returnVals{
		ID:         userData.ID,
		Created_At: userData.CreatedAt,
		Updated_At: userData.UpdatedAt,
		Email:      userData.Email,
	}

	respondWithJSON(w, http.StatusCreated, payload)
}
