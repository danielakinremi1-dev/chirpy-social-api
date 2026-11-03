package api

import (
	"net/http"

	openapi "github.com/danielakinremi1-dev/chirpy-social-api/api"
)

// handleDocs serves Swagger UI, which loads the spec from /docs/openapi.yaml.
func handleDocs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(openapi.DocsPage)
}

func handleOpenAPISpec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/yaml")
	w.Write(openapi.Spec)
}
