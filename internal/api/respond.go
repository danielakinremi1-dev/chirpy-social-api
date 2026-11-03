package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
)

func respondWithError(w http.ResponseWriter, r *http.Request, code int, msg string, err error) {
	if code > 499 {
		slog.ErrorContext(r.Context(), msg, "status", code, "error", err)
	} else if err != nil {
		slog.InfoContext(r.Context(), msg, "status", code, "error", err)
	}
	type errorResponse struct {
		Error string `json:"error"`
	}
	respondWithJSON(w, code, errorResponse{
		Error: msg,
	})
}

func respondWithJSON(w http.ResponseWriter, code int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	data, err := json.Marshal(payload)
	if err != nil {
		slog.Error("Error marshalling JSON", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(code)
	w.Write(data)
}

func respondWithText(w http.ResponseWriter, code int) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(code)
	w.Write([]byte(http.StatusText(code)))
}

// decodeJSON decodes the request body into dst. If it can't, it responds with
// 400 (or 413 for an oversized body) and returns false.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			respondWithError(w, r, http.StatusRequestEntityTooLarge, "Request body too large", err)
			return false
		}
		respondWithError(w, r, http.StatusBadRequest, "Invalid request body", err)
		return false
	}
	return true
}
