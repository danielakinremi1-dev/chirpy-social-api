package main

import (
	"net/http"
	"time"

	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerChirp(w http.ResponseWriter, req *http.Request) {

	type chirps struct {
		ID         uuid.UUID `json:"id"`
		Created_At time.Time `json:"created_at"`
		Updated_At time.Time `json:"updated_at"`
		Body       string    `json:"body"`
		User_ID    uuid.UUID `json:"user_id"`
	}

	parsedID, err := uuid.Parse(req.PathValue("chirpID"))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error parsing provided chirp ID", err)
		return
	}

	chirpData, err := cfg.db.GetChirp(req.Context(), parsedID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Chirp ID not found", err)
		return
	}

	payload := chirps{
		ID:         chirpData.ID,
		Created_At: chirpData.CreatedAt,
		Updated_At: chirpData.UpdatedAt,
		Body:       chirpData.Body,
		User_ID:    chirpData.UserID,
	}

	respondWithJSON(w, http.StatusOK, payload)
}
