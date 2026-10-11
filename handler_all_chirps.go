package main

import (
	"net/http"
	"sort"
	"time"

	"github.com/danielakinremi1-dev/chirpy/projects/internal/database"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerGetAllChirps(w http.ResponseWriter, req *http.Request) {

	authorIDString := req.URL.Query().Get("author_id")
	var chirps []database.Chirp
	var err error

	if authorIDString != "" {
		authorID, parseErr := uuid.Parse(authorIDString)
		if parseErr != nil {
			respondWithError(w, http.StatusBadRequest, "Error parsing author ID", parseErr)
			return
		}
		chirps, err = cfg.db.GetAuthorChirps(req.Context(), authorID)
	} else {
		chirps, err = cfg.db.GetAllChirps(req.Context())
	}

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error retrieving chirps", err)
		return
	}

	order := req.URL.Query().Get("sort")
	sendChirps(w, chirps, order)

}

func sendChirps(w http.ResponseWriter, chirpsSlice []database.Chirp, order string) {

	type chirps struct {
		ID         uuid.UUID `json:"id"`
		Created_At time.Time `json:"created_at"`
		Updated_At time.Time `json:"updated_at"`
		Body       string    `json:"body"`
		User_ID    uuid.UUID `json:"user_id"`
	}

	payload := []chirps{}

	for _, c := range chirpsSlice {
		payload = append(payload, chirps{
			ID:         c.ID,
			Created_At: c.CreatedAt,
			Updated_At: c.UpdatedAt,
			Body:       c.Body,
			User_ID:    c.UserID,
		})
	}

	if order == "desc" {
		sort.Slice(payload, func(i int, j int) bool {
			return (payload[i].Created_At).After(payload[j].Created_At)
		})
	} else {
		sort.Slice(payload, func(i int, j int) bool {
			return (payload[i].Created_At).Before(payload[j].Created_At)
		})
	}

	respondWithJSON(w, http.StatusOK, payload)

}
