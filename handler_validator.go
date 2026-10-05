package main

import (
	"encoding/json"
	"net/http"
	"slices"
)

func handlerValidator(w http.ResponseWriter, req *http.Request) {

	type parameter struct {
		Body string `json:"body"`
	}

	type returnVals struct {
		Valid bool `json:"valid"`
	}

	decoder := json.NewDecoder(req.Body)
	params := parameter{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, 500, "Error decoding JSON", err)
		return
	}

	if len(params.Body) > 140 {
		respondWithError(w, 400, "Chirp is too long", err)
		return
	}
	
	params.Body = wordChecker()
	
	payload := returnVals{Valid: true}

	respondWithJSON(w, 200, payload)
}

func wordChecker (text, ) string
	badWords := []string{"kerfuffle", "sharbert", "fornax"}
		for _, word := range badWords{
			if slices.Contains(params.Body, word)

		}