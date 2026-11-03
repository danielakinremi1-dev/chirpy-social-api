package api

import (
	"fmt"
	"net/http"
)

func handleReadiness(w http.ResponseWriter, r *http.Request) {
	respondWithText(w, http.StatusOK)
}

func (a *API) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, a.fileserverHits.Load())
}

// handleReset deletes all users and resets the hit counter. It only works
// when PLATFORM=dev.
func (a *API) handleReset(w http.ResponseWriter, r *http.Request) {
	if a.platform != "dev" {
		respondWithText(w, http.StatusForbidden)
		return
	}

	a.fileserverHits.Store(0)
	if err := a.store.DeleteAllUsers(r.Context()); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error deleting users", err)
		return
	}

	respondWithText(w, http.StatusOK)
}
