package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUsers(w http.ResponseWriter, req *http.Request) {

	type parameter struct {
		Email string `json:"email"`
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

	userData, err := cfg.db.CreateUser(req.Context(), params.Email)
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
