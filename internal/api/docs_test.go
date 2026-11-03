package api

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"testing"

	openapi "github.com/danielakinremi1-dev/chirpy-social-api/api"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/google/uuid"
)

func loadSpec(t *testing.T) *openapi3.T {
	t.Helper()
	doc, err := openapi3.NewLoader().LoadFromData(openapi.Spec)
	if err != nil {
		t.Fatalf("loading spec: %v", err)
	}
	if err := doc.Validate(context.Background()); err != nil {
		t.Fatalf("invalid spec: %v", err)
	}
	return doc
}

// TestSpecMatchesRoutes sends a bare request (no auth, no body) to every
// operation in the spec and checks the server answers with a documented status,
// which catches routes that are missing or renamed on either side.
func TestSpecMatchesRoutes(t *testing.T) {
	doc := loadSpec(t)
	s := newTestServer(t)

	for path, item := range doc.Paths.Map() {
		for method, op := range item.Operations() {
			t.Run(method+" "+path, func(t *testing.T) {
				rec := s.do(method, strings.ReplaceAll(path, "{chirpID}", uuid.NewString()), "", nil)
				if op.Responses.Value(strconv.Itoa(rec.Code)) == nil {
					t.Errorf("status %d is not documented; body: %s", rec.Code, rec.Body)
				}
			})
		}
	}
}

func TestDocsRoutes(t *testing.T) {
	s := newTestServer(t)

	rec := s.do("GET", "/docs", "", nil)
	expectStatus(t, rec, http.StatusOK)
	if !strings.Contains(rec.Body.String(), "/docs/openapi.yaml") {
		t.Error("docs page doesn't load the spec")
	}

	rec = s.do("GET", "/docs/openapi.yaml", "", nil)
	expectStatus(t, rec, http.StatusOK)
	if rec.Header().Get("Content-Type") != "application/yaml" || !strings.HasPrefix(rec.Body.String(), "openapi: 3") {
		t.Errorf("unexpected spec response: %s %.40q", rec.Header().Get("Content-Type"), rec.Body)
	}
}
