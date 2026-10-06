package main

import (
	"net/http"
	"time"

	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerGetChirps(w http.ResponseWriter, req *http.Request) {

	type chirps struct {
		ID         uuid.UUID `json:"id"`
		Created_At time.Time `json:"created_at"`
		Updated_At time.Time `json:"updated_at"`
		Body       string    `json:"body"`
		User_ID    uuid.UUID `json:"user_id"`
	}

	chirpData, err := cfg.db.GetAllChirps(req.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating new chirp", err)
		return
	}

	payload := []chirps{}

	for _, c := range chirpData {
		payload = append(payload, chirps{
			ID:         c.ID,
			Created_At: c.CreatedAt,
			Updated_At: c.UpdatedAt,
			Body:       c.Body,
			User_ID:    c.UserID,
		})
	}

	respondWithJSON(w, http.StatusOK, payload)
}
