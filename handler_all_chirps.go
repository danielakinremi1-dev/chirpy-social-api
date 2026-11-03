package main

import (
	"database/sql"
	"encoding/base64"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/danielakinremi1-dev/chirpy-social-api/internal/database"
	"github.com/google/uuid"
)

const (
	defaultChirpsLimit = 20
	maxChirpsLimit     = 100
)

func (cfg *apiConfig) handlerGetAllChirps(w http.ResponseWriter, req *http.Request) {

	query := req.URL.Query()

	var authorID uuid.NullUUID
	if authorIDString := query.Get("author_id"); authorIDString != "" {
		parsedID, err := uuid.Parse(authorIDString)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Error parsing author ID", err)
			return
		}
		authorID = uuid.NullUUID{UUID: parsedID, Valid: true}
	}

	limit := defaultChirpsLimit
	if limitString := query.Get("limit"); limitString != "" {
		parsedLimit, err := strconv.Atoi(limitString)
		if err != nil || parsedLimit < 1 {
			respondWithError(w, http.StatusBadRequest, "Invalid limit", err)
			return
		}
		limit = min(parsedLimit, maxChirpsLimit)
	}

	var cursorCreatedAt sql.NullTime
	var cursorID uuid.NullUUID
	if cursor := query.Get("cursor"); cursor != "" {
		createdAt, id, err := decodeChirpCursor(cursor)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid cursor", err)
			return
		}
		cursorCreatedAt = sql.NullTime{Time: createdAt, Valid: true}
		cursorID = uuid.NullUUID{UUID: id, Valid: true}
	}

	// Fetch one extra row to tell whether there is another page.
	rowLimit := int32(limit + 1)

	var chirps []database.Chirp
	var err error
	if query.Get("sort") == "desc" {
		chirps, err = cfg.db.ListChirpsDesc(req.Context(), database.ListChirpsDescParams{
			AuthorID:        authorID,
			CursorCreatedAt: cursorCreatedAt,
			CursorID:        cursorID,
			RowLimit:        rowLimit,
		})
	} else {
		chirps, err = cfg.db.ListChirpsAsc(req.Context(), database.ListChirpsAscParams{
			AuthorID:        authorID,
			CursorCreatedAt: cursorCreatedAt,
			CursorID:        cursorID,
			RowLimit:        rowLimit,
		})
	}
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error retrieving chirps", err)
		return
	}

	if len(chirps) > limit {
		chirps = chirps[:limit]
		last := chirps[len(chirps)-1]
		w.Header().Set("X-Next-Cursor", encodeChirpCursor(last.CreatedAt, last.ID))
	}

	sendChirps(w, chirps)

}

func sendChirps(w http.ResponseWriter, chirpsSlice []database.Chirp) {

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

	respondWithJSON(w, http.StatusOK, payload)

}

// A cursor is the (created_at, id) of the last chirp on a page, encoded so
// clients treat it as opaque.
func encodeChirpCursor(createdAt time.Time, id uuid.UUID) string {
	raw := createdAt.UTC().Format(time.RFC3339Nano) + "," + id.String()
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func decodeChirpCursor(cursor string) (time.Time, uuid.UUID, error) {
	raw, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return time.Time{}, uuid.Nil, err
	}

	createdAtString, idString, found := strings.Cut(string(raw), ",")
	if !found {
		return time.Time{}, uuid.Nil, errors.New("malformed cursor")
	}

	createdAt, err := time.Parse(time.RFC3339Nano, createdAtString)
	if err != nil {
		return time.Time{}, uuid.Nil, err
	}

	id, err := uuid.Parse(idString)
	if err != nil {
		return time.Time{}, uuid.Nil, err
	}

	return createdAt, id, nil
}
