package api

import (
	"net/http"

	"github.com/danielakinremi1-dev/chirpy-social-api/web"
)

// Routes returns the handler for every Chirpy endpoint, wrapped in request
// ID, logging, panic recovery, and body size middleware.
func (a *API) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.Handle("/app/", http.StripPrefix("/app", a.metricsInc(http.FileServerFS(web.FS))))

	mux.HandleFunc("GET /docs", handleDocs)
	mux.HandleFunc("GET /docs/openapi.yaml", handleOpenAPISpec)

	mux.HandleFunc("GET /api/healthz", handleReadiness)

	mux.HandleFunc("POST /api/users", a.handleCreateUser)
	mux.HandleFunc("PUT /api/users", a.requireAuth(a.handleUpdateUser))

	mux.HandleFunc("POST /api/login", a.loginLimiter.limit(a.handleLogin))
	mux.HandleFunc("POST /api/refresh", a.handleRefresh)
	mux.HandleFunc("POST /api/revoke", a.handleRevoke)

	mux.HandleFunc("GET /api/chirps", a.handleListChirps)
	mux.HandleFunc("POST /api/chirps", a.requireAuth(a.handleCreateChirp))
	mux.HandleFunc("GET /api/chirps/{chirpID}", a.handleGetChirp)
	mux.HandleFunc("DELETE /api/chirps/{chirpID}", a.requireAuth(a.handleDeleteChirp))

	mux.HandleFunc("POST /api/polka/webhooks", a.handlePolkaWebhook)

	mux.HandleFunc("GET /admin/metrics", a.handleMetrics)
	mux.HandleFunc("POST /admin/reset", a.handleReset)

	return withRequestID(logRequests(recoverPanics(limitBodySize(mux))))
}
