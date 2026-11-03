package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"time"
	"unicode/utf8"

	"github.com/danielakinremi1-dev/chirpy-social-api/internal/auth"
	"github.com/danielakinremi1-dev/chirpy-social-api/internal/database"
	"github.com/google/uuid"
)

const maxChirpLength = 140

func (cfg *apiConfig) handlerPostChirps(w http.ResponseWriter, req *http.Request) {

	type parameter struct {
		Body string `json:"body"`
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
		respondWithError(w, http.StatusBadRequest, "Invalid request body", err)
		return
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

	if utf8.RuneCountInString(params.Body) > maxChirpLength {
		respondWithError(w, http.StatusBadRequest, "Chirp is too long", fmt.Errorf("Max chirp length exceeded"))
		return
	}
	params.Body = filterBadWords(params.Body)

	queryArgs := database.CreateChirpParams{
		Body:   params.Body,
		UserID: userID,
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

var badWordsPattern = regexp.MustCompile(`(?i)\b(kerfuffle|sharbert|fornax)\b`)

func filterBadWords(text string) string {
	return badWordsPattern.ReplaceAllString(text, "****")
}
