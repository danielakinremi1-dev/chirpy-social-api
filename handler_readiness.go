package main

import "net/http"

func handlerReadiness(w http.ResponseWriter, req *http.Request) {
	respondWithCode(w, http.StatusOK)
}
