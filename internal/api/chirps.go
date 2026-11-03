package api

import (
	"database/sql"
	"encoding/base64"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/danielakinremi1-dev/chirpy-social-api/internal/database"
	"github.com/google/uuid"
)

const (
	maxChirpLength     = 140
	defaultChirpsLimit = 20
	maxChirpsLimit     = 100
)

var badWordsPattern = regexp.MustCompile(`(?i)\b(kerfuffle|sharbert|fornax)\b`)

func (a *API) handleCreateChirp(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body string `json:"body"`
	}

	var params parameters
	if !decodeJSON(w, r, &params) {
		return
	}

	if utf8.RuneCountInString(params.Body) > maxChirpLength {
		respondWithError(w, http.StatusBadRequest, "Chirp is too long", nil)
		return
	}

	chirp, err := a.store.CreateChirp(r.Context(), database.CreateChirpParams{
		Body:   filterBadWords(params.Body),
		UserID: userIDFromContext(r.Context()),
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating chirp", err)
		return
	}

	respondWithJSON(w, http.StatusCreated, toChirpResponse(chirp))
}

func (a *API) handleGetChirp(w http.ResponseWriter, r *http.Request) {
	chirpID, err := uuid.Parse(r.PathValue("chirpID"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid chirp ID", err)
		return
	}

	chirp, err := a.store.GetChirp(r.Context(), chirpID)
	if errors.Is(err, sql.ErrNoRows) {
		respondWithError(w, http.StatusNotFound, "Chirp not found", err)
		return
	} else if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error retrieving chirp", err)
		return
	}

	respondWithJSON(w, http.StatusOK, toChirpResponse(chirp))
}

func (a *API) handleDeleteChirp(w http.ResponseWriter, r *http.Request) {
	chirpID, err := uuid.Parse(r.PathValue("chirpID"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid chirp ID", err)
		return
	}

	chirp, err := a.store.GetChirp(r.Context(), chirpID)
	if errors.Is(err, sql.ErrNoRows) {
		respondWithError(w, http.StatusNotFound, "Chirp not found", err)
		return
	} else if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error retrieving chirp", err)
		return
	}

	if chirp.UserID != userIDFromContext(r.Context()) {
		respondWithError(w, http.StatusForbidden, "You can only delete your own chirps", nil)
		return
	}

	if err := a.store.DeleteChirp(r.Context(), chirpID); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error deleting chirp", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleListChirps returns a page of chirps, optionally filtered by author_id
// and sorted by sort=asc|desc. When more results exist, the X-Next-Cursor
// header holds the cursor for the next page.
func (a *API) handleListChirps(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	var authorID uuid.NullUUID
	if authorIDString := query.Get("author_id"); authorIDString != "" {
		parsedID, err := uuid.Parse(authorIDString)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid author ID", err)
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
		chirps, err = a.store.ListChirpsDesc(r.Context(), database.ListChirpsDescParams{
			AuthorID:        authorID,
			CursorCreatedAt: cursorCreatedAt,
			CursorID:        cursorID,
			RowLimit:        rowLimit,
		})
	} else {
		chirps, err = a.store.ListChirpsAsc(r.Context(), database.ListChirpsAscParams{
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

	payload := make([]chirpResponse, 0, len(chirps))
	for _, c := range chirps {
		payload = append(payload, toChirpResponse(c))
	}
	respondWithJSON(w, http.StatusOK, payload)
}

func filterBadWords(text string) string {
	return badWordsPattern.ReplaceAllString(text, "****")
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
