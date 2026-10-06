package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/danielakinremi1-dev/chirpy/projects/internal/database"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerPostChirps(w http.ResponseWriter, req *http.Request) {

	type parameter struct {
		Body    string    `json:"body"`
		User_ID uuid.UUID `json:"user_id"`
	}

	type returnVals struct {
		ID         uuid.UUID `json:"id"`
		Created_At time.Time `json:"created_at"`
		Updated_At time.Time `json:"updated_at"`
		Body       string    `json:"body"`
		User_ID    uuid.UUID `json:"user_id"`
	}

	decoder := json.NewDecoder(req.Body)
	params := parameter{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error decoding request data", err)
		return
	}

	if len(params.Body) > 140 {
		respondWithError(w, http.StatusBadRequest, "Chirp is too long", err)
		return
	}
	params.Body = filterBadWords(params.Body)

	queryArgs := database.CreateChirpParams{
		Body:   params.Body,
		UserID: params.User_ID,
	}

	chirpData, err := cfg.db.CreateChirp(req.Context(), queryArgs)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating new chirp", err)
		return
	}

	payload := returnVals{
		ID:         chirpData.ID,
		Created_At: chirpData.CreatedAt,
		Updated_At: chirpData.UpdatedAt,
		Body:       chirpData.Body,
		User_ID:    chirpData.UserID,
	}
	respondWithJSON(w, http.StatusCreated, payload)
}

func filterBadWords(text string) string {
	badWords := []string{"kerfuffle", "sharbert", "fornax",
		"Kerfuffle", "Sharbert", "Fornax"}
	cleanText := text
	for _, word := range badWords {
		cleanText = strings.ReplaceAll(cleanText, word, "****")
	}
	return cleanText
}
