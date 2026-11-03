package api

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/danielakinremi1-dev/chirpy-social-api/internal/auth"
	"github.com/danielakinremi1-dev/chirpy-social-api/internal/database"
)

func (a *API) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var params credentials
	if !decodeJSON(w, r, &params) {
		return
	}
	if err := params.validate(); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error(), err)
		return
	}

	hashedPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error hashing password", err)
		return
	}

	user, err := a.store.CreateUser(r.Context(), database.CreateUserParams{
		Email:          params.Email,
		HashedPassword: hashedPassword,
	})
	if isUniqueViolation(err) {
		respondWithError(w, http.StatusConflict, "Email is already registered", err)
		return
	} else if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating user", err)
		return
	}

	respondWithJSON(w, http.StatusCreated, toUserResponse(user))
}

func (a *API) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	var params credentials
	if !decodeJSON(w, r, &params) {
		return
	}
	if err := params.validate(); err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error(), err)
		return
	}

	hashedPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error hashing password", err)
		return
	}

	user, err := a.store.UpdateUser(r.Context(), database.UpdateUserParams{
		Email:          params.Email,
		HashedPassword: hashedPassword,
		ID:             userIDFromContext(r.Context()),
	})
	if errors.Is(err, sql.ErrNoRows) {
		respondWithError(w, http.StatusNotFound, "User not found", err)
		return
	} else if isUniqueViolation(err) {
		respondWithError(w, http.StatusConflict, "Email is already registered", err)
		return
	} else if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error updating user", err)
		return
	}

	respondWithJSON(w, http.StatusOK, toUserResponse(user))
}
