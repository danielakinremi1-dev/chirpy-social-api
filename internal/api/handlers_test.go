package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/danielakinremi1-dev/chirpy-social-api/internal/auth"
	"github.com/danielakinremi1-dev/chirpy-social-api/internal/config"
	"github.com/danielakinremi1-dev/chirpy-social-api/internal/database"
	"github.com/google/uuid"
)

const (
	testSecret   = "test-secret"
	testPolkaKey = "test-polka-key"
	testPassword = "password123"
)

func TestMain(m *testing.M) {
	slog.SetDefault(slog.New(slog.DiscardHandler))
	os.Exit(m.Run())
}

type testServer struct {
	t       *testing.T
	store   *fakeStore
	handler http.Handler
}

func newTestServer(t *testing.T) *testServer {
	t.Helper()
	store := newFakeStore()
	a := New(store, config.Config{JWTSecret: testSecret, PolkaKey: testPolkaKey, Platform: "dev"})
	return &testServer{t: t, store: store, handler: a.Routes()}
}

// do sends a request through the full middleware stack. body may be nil, a
// string sent as is, or a value encoded as JSON.
func (s *testServer) do(method, path, authorization string, body any) *httptest.ResponseRecorder {
	s.t.Helper()
	var reader io.Reader
	switch b := body.(type) {
	case nil:
	case string:
		reader = strings.NewReader(b)
	default:
		data, err := json.Marshal(b)
		if err != nil {
			s.t.Fatalf("encoding request body: %v", err)
		}
		reader = bytes.NewReader(data)
	}
	req := httptest.NewRequest(method, path, reader)
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	rec := httptest.NewRecorder()
	s.handler.ServeHTTP(rec, req)
	return rec
}

type loginBody struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	IsChirpyRed  bool      `json:"is_chirpy_red"`
	Token        string    `json:"token"`
	RefreshToken string    `json:"refresh_token"`
}

// signUp creates a user and logs in, failing the test on any error.
func (s *testServer) signUp(email string) loginBody {
	s.t.Helper()
	if rec := s.do("POST", "/api/users", "", credentials{Email: email, Password: testPassword}); rec.Code != http.StatusCreated {
		s.t.Fatalf("creating user: status %d: %s", rec.Code, rec.Body)
	}
	rec := s.do("POST", "/api/login", "", credentials{Email: email, Password: testPassword})
	if rec.Code != http.StatusOK {
		s.t.Fatalf("logging in: status %d: %s", rec.Code, rec.Body)
	}
	return decodeBody[loginBody](s.t, rec)
}

func (s *testServer) postChirp(token, body string) chirpResponse {
	s.t.Helper()
	rec := s.do("POST", "/api/chirps", "Bearer "+token, map[string]string{"body": body})
	if rec.Code != http.StatusCreated {
		s.t.Fatalf("posting chirp: status %d: %s", rec.Code, rec.Body)
	}
	return decodeBody[chirpResponse](s.t, rec)
}

func decodeBody[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("decoding response %q: %v", rec.Body, err)
	}
	return v
}

func expectStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rec.Code != want {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, want, rec.Body)
	}
}

func TestHealthz(t *testing.T) {
	s := newTestServer(t)
	rec := s.do("GET", "/api/healthz", "", nil)
	expectStatus(t, rec, http.StatusOK)
	if rec.Header().Get("X-Request-ID") == "" {
		t.Error("missing X-Request-ID header")
	}
}

func TestCreateUser(t *testing.T) {
	tests := []struct {
		name       string
		body       any
		storeErr   error
		wantStatus int
	}{
		{name: "Valid", body: credentials{Email: "a@example.com", Password: testPassword}, wantStatus: http.StatusCreated},
		{name: "Malformed JSON", body: "{", wantStatus: http.StatusBadRequest},
		{name: "Invalid email", body: credentials{Email: "nope", Password: testPassword}, wantStatus: http.StatusBadRequest},
		{name: "Short password", body: credentials{Email: "a@example.com", Password: "short"}, wantStatus: http.StatusBadRequest},
		{name: "Duplicate email", body: credentials{Email: "taken@example.com", Password: testPassword}, wantStatus: http.StatusConflict},
		{name: "Store error", body: credentials{Email: "a@example.com", Password: testPassword}, storeErr: errors.New("boom"), wantStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestServer(t)
			s.do("POST", "/api/users", "", credentials{Email: "taken@example.com", Password: testPassword})
			s.store.err = tt.storeErr

			rec := s.do("POST", "/api/users", "", tt.body)
			expectStatus(t, rec, tt.wantStatus)
			if tt.wantStatus == http.StatusCreated {
				user := decodeBody[map[string]any](t, rec)
				if user["email"] != "a@example.com" {
					t.Errorf("email = %v, want a@example.com", user["email"])
				}
				if _, ok := user["hashed_password"]; ok {
					t.Error("response leaks hashed_password")
				}
			}
		})
	}
}

func TestLogin(t *testing.T) {
	s := newTestServer(t)
	user := s.signUp("a@example.com")

	t.Run("Valid", func(t *testing.T) {
		gotID, err := auth.ValidateJWT(user.Token, testSecret)
		if err != nil || gotID != user.ID {
			t.Errorf("ValidateJWT() = %v, %v; want %v", gotID, err, user.ID)
		}
		if _, ok := s.store.refreshTokens[user.RefreshToken]; !ok {
			t.Error("refresh token not stored")
		}
	})

	for _, tt := range []struct {
		name       string
		body       any
		wantStatus int
	}{
		{name: "Wrong password", body: credentials{Email: "a@example.com", Password: "wrong-password"}, wantStatus: http.StatusUnauthorized},
		{name: "Unknown email", body: credentials{Email: "b@example.com", Password: testPassword}, wantStatus: http.StatusUnauthorized},
		{name: "Malformed JSON", body: "{", wantStatus: http.StatusBadRequest},
	} {
		t.Run(tt.name, func(t *testing.T) {
			rec := s.do("POST", "/api/login", "", tt.body)
			expectStatus(t, rec, tt.wantStatus)
			if tt.wantStatus == http.StatusUnauthorized {
				if got := decodeBody[map[string]string](t, rec)["error"]; got != "Incorrect email or password" {
					t.Errorf("error = %q, want the same message for every credential failure", got)
				}
			}
		})
	}

	t.Run("Store error", func(t *testing.T) {
		s := newTestServer(t)
		s.store.err = errors.New("boom")
		rec := s.do("POST", "/api/login", "", credentials{Email: "a@example.com", Password: testPassword})
		expectStatus(t, rec, http.StatusInternalServerError)
	})
}

func TestLoginRateLimit(t *testing.T) {
	s := newTestServer(t)
	body := credentials{Email: "nobody@example.com", Password: testPassword}

	for i := range 10 {
		if rec := s.do("POST", "/api/login", "", body); rec.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: status = %d, want 401", i+1, rec.Code)
		}
	}
	rec := s.do("POST", "/api/login", "", body)
	expectStatus(t, rec, http.StatusTooManyRequests)
	if rec.Header().Get("Retry-After") == "" {
		t.Error("missing Retry-After header")
	}
}

func TestIPRateLimiterSweepsIdleClients(t *testing.T) {
	l := newIPRateLimiter(time.Second, 1, false)
	if !l.allow("192.0.2.1") {
		t.Fatal("first request should be allowed")
	}
	if l.allow("192.0.2.1") {
		t.Fatal("second request should be limited")
	}

	l.clients["192.0.2.1"].lastSeen = time.Now().Add(-time.Hour)
	l.lastSweep = time.Now().Add(-time.Hour)
	l.allow("192.0.2.2")

	if _, ok := l.clients["192.0.2.1"]; ok {
		t.Error("idle client was not swept")
	}
}

func TestUpdateUser(t *testing.T) {
	s := newTestServer(t)
	user := s.signUp("a@example.com")
	s.signUp("taken@example.com")
	auth := "Bearer " + user.Token

	tests := []struct {
		name          string
		authorization string
		body          any
		wantStatus    int
	}{
		{name: "Missing token", body: credentials{Email: "new@example.com", Password: testPassword}, wantStatus: http.StatusUnauthorized},
		{name: "Invalid token", authorization: "Bearer not-a-jwt", body: credentials{Email: "new@example.com", Password: testPassword}, wantStatus: http.StatusUnauthorized},
		{name: "Malformed JSON", authorization: auth, body: "{", wantStatus: http.StatusBadRequest},
		{name: "Short password", authorization: auth, body: credentials{Email: "new@example.com", Password: "short"}, wantStatus: http.StatusBadRequest},
		{name: "Email taken", authorization: auth, body: credentials{Email: "taken@example.com", Password: testPassword}, wantStatus: http.StatusConflict},
		{name: "Valid", authorization: auth, body: credentials{Email: "new@example.com", Password: "new-password"}, wantStatus: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := s.do("PUT", "/api/users", tt.authorization, tt.body)
			expectStatus(t, rec, tt.wantStatus)
		})
	}

	rec := s.do("POST", "/api/login", "", credentials{Email: "new@example.com", Password: "new-password"})
	expectStatus(t, rec, http.StatusOK)

	t.Run("Deleted user", func(t *testing.T) {
		delete(s.store.users, user.ID)
		rec := s.do("PUT", "/api/users", auth, credentials{Email: "new2@example.com", Password: testPassword})
		expectStatus(t, rec, http.StatusNotFound)
	})

	t.Run("Store error", func(t *testing.T) {
		s.store.err = errors.New("boom")
		defer func() { s.store.err = nil }()
		rec := s.do("PUT", "/api/users", auth, credentials{Email: "new3@example.com", Password: testPassword})
		expectStatus(t, rec, http.StatusInternalServerError)
	})
}

func TestCreateChirp(t *testing.T) {
	s := newTestServer(t)
	user := s.signUp("a@example.com")
	auth := "Bearer " + user.Token

	t.Run("Valid", func(t *testing.T) {
		chirp := s.postChirp(user.Token, "what a Kerfuffle")
		if chirp.Body != "what a ****" {
			t.Errorf("body = %q, want filtered", chirp.Body)
		}
		if chirp.UserID != user.ID {
			t.Errorf("user_id = %v, want %v", chirp.UserID, user.ID)
		}
	})

	tests := []struct {
		name          string
		authorization string
		body          any
		wantStatus    int
	}{
		{name: "Missing token", body: map[string]string{"body": "hi"}, wantStatus: http.StatusUnauthorized},
		{name: "Malformed JSON", authorization: auth, body: "{", wantStatus: http.StatusBadRequest},
		{name: "Too long", authorization: auth, body: map[string]string{"body": strings.Repeat("a", 141)}, wantStatus: http.StatusBadRequest},
		{name: "140 multibyte characters", authorization: auth, body: map[string]string{"body": strings.Repeat("é", 140)}, wantStatus: http.StatusCreated},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectStatus(t, s.do("POST", "/api/chirps", tt.authorization, tt.body), tt.wantStatus)
		})
	}

	t.Run("Store error", func(t *testing.T) {
		s.store.err = errors.New("boom")
		defer func() { s.store.err = nil }()
		expectStatus(t, s.do("POST", "/api/chirps", auth, map[string]string{"body": "hi"}), http.StatusInternalServerError)
	})
}

func TestGetAndDeleteChirp(t *testing.T) {
	s := newTestServer(t)
	owner := s.signUp("owner@example.com")
	other := s.signUp("other@example.com")
	chirp := s.postChirp(owner.Token, "hello")
	path := "/api/chirps/" + chirp.ID.String()

	rec := s.do("GET", path, "", nil)
	expectStatus(t, rec, http.StatusOK)
	if got := decodeBody[chirpResponse](t, rec); got.ID != chirp.ID || got.Body != "hello" {
		t.Errorf("got %+v, want %+v", got, chirp)
	}

	expectStatus(t, s.do("GET", "/api/chirps/not-a-uuid", "", nil), http.StatusBadRequest)
	expectStatus(t, s.do("GET", "/api/chirps/"+uuid.NewString(), "", nil), http.StatusNotFound)

	expectStatus(t, s.do("DELETE", path, "", nil), http.StatusUnauthorized)
	expectStatus(t, s.do("DELETE", "/api/chirps/not-a-uuid", "Bearer "+owner.Token, nil), http.StatusBadRequest)
	expectStatus(t, s.do("DELETE", "/api/chirps/"+uuid.NewString(), "Bearer "+owner.Token, nil), http.StatusNotFound)
	expectStatus(t, s.do("DELETE", path, "Bearer "+other.Token, nil), http.StatusForbidden)
	expectStatus(t, s.do("DELETE", path, "Bearer "+owner.Token, nil), http.StatusNoContent)
	expectStatus(t, s.do("GET", path, "", nil), http.StatusNotFound)

	s.store.err = errors.New("boom")
	expectStatus(t, s.do("GET", path, "", nil), http.StatusInternalServerError)
	expectStatus(t, s.do("DELETE", path, "Bearer "+owner.Token, nil), http.StatusInternalServerError)
}

func TestDeleteChirpStoreError(t *testing.T) {
	s := newTestServer(t)
	owner := s.signUp("owner@example.com")
	chirp := s.postChirp(owner.Token, "hello")

	// Fail only the delete, after the ownership check has read the chirp.
	store := &failingDeleteStore{fakeStore: s.store}
	s.handler = New(store, config.Config{JWTSecret: testSecret}).Routes()
	expectStatus(t, s.do("DELETE", "/api/chirps/"+chirp.ID.String(), "Bearer "+owner.Token, nil), http.StatusInternalServerError)
}

type failingDeleteStore struct {
	*fakeStore
}

func (f *failingDeleteStore) DeleteChirp(ctx context.Context, id uuid.UUID) error {
	return errors.New("boom")
}

func TestListChirps(t *testing.T) {
	s := newTestServer(t)
	alice := s.signUp("alice@example.com")
	bob := s.signUp("bob@example.com")
	for _, c := range []struct{ token, body string }{
		{alice.Token, "a1"}, {bob.Token, "b1"}, {alice.Token, "a2"}, {bob.Token, "b2"}, {alice.Token, "a3"},
	} {
		s.postChirp(c.token, c.body)
	}

	list := func(t *testing.T, query string) ([]string, string) {
		t.Helper()
		rec := s.do("GET", "/api/chirps"+query, "", nil)
		expectStatus(t, rec, http.StatusOK)
		var bodies []string
		for _, c := range decodeBody[[]chirpResponse](t, rec) {
			bodies = append(bodies, c.Body)
		}
		return bodies, rec.Header().Get("X-Next-Cursor")
	}

	// collect follows X-Next-Cursor until the last page.
	collect := func(t *testing.T, query string) [][]string {
		t.Helper()
		var pages [][]string
		cursor := ""
		for {
			q := query
			if cursor != "" {
				q += "&cursor=" + cursor
			}
			page, next := list(t, q)
			pages = append(pages, page)
			if next == "" {
				return pages
			}
			cursor = next
		}
	}

	tests := []struct {
		name  string
		query string
		want  string
	}{
		{name: "Ascending pages", query: "?limit=2", want: "[[a1 b1] [a2 b2] [a3]]"},
		{name: "Descending pages", query: "?limit=2&sort=desc", want: "[[a3 b2] [a2 b1] [a1]]"},
		{name: "Author filter", query: "?limit=2&author_id=" + alice.ID.String(), want: "[[a1 a2] [a3]]"},
		{name: "Author filter descending", query: "?sort=desc&author_id=" + bob.ID.String(), want: "[[b2 b1]]"},
		{name: "Limit above max", query: "?limit=1000", want: "[[a1 b1 a2 b2 a3]]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := fmt.Sprint(collect(t, tt.query)); got != tt.want {
				t.Errorf("pages = %s, want %s", got, tt.want)
			}
		})
	}

	t.Run("Default limit", func(t *testing.T) {
		for i := range 20 {
			s.postChirp(alice.Token, fmt.Sprint("extra", i))
		}
		page, next := list(t, "")
		if len(page) != defaultChirpsLimit || next == "" {
			t.Errorf("got %d chirps and cursor %q, want %d and a cursor", len(page), next, defaultChirpsLimit)
		}
	})

	for _, query := range []string{"?limit=0", "?limit=abc", "?cursor=garbage", "?author_id=nope"} {
		t.Run("Invalid "+query, func(t *testing.T) {
			expectStatus(t, s.do("GET", "/api/chirps"+query, "", nil), http.StatusBadRequest)
		})
	}

	t.Run("Store error", func(t *testing.T) {
		s.store.err = errors.New("boom")
		defer func() { s.store.err = nil }()
		expectStatus(t, s.do("GET", "/api/chirps", "", nil), http.StatusInternalServerError)
	})
}

func TestRefreshAndRevoke(t *testing.T) {
	s := newTestServer(t)
	user := s.signUp("a@example.com")

	rec := s.do("POST", "/api/refresh", "Bearer "+user.RefreshToken, nil)
	expectStatus(t, rec, http.StatusOK)
	rotated := decodeBody[loginBody](t, rec)
	if rotated.RefreshToken == "" || rotated.RefreshToken == user.RefreshToken {
		t.Fatalf("refresh token was not rotated: %q", rotated.RefreshToken)
	}
	if gotID, err := auth.ValidateJWT(rotated.Token, testSecret); err != nil || gotID != user.ID {
		t.Errorf("new access token: ValidateJWT() = %v, %v", gotID, err)
	}

	expectStatus(t, s.do("POST", "/api/refresh", "Bearer "+user.RefreshToken, nil), http.StatusUnauthorized)
	expectStatus(t, s.do("POST", "/api/refresh", "", nil), http.StatusUnauthorized)
	expectStatus(t, s.do("POST", "/api/refresh", "Bearer unknown", nil), http.StatusUnauthorized)

	expectStatus(t, s.do("POST", "/api/revoke", "Bearer "+rotated.RefreshToken, nil), http.StatusNoContent)
	expectStatus(t, s.do("POST", "/api/revoke", "Bearer "+rotated.RefreshToken, nil), http.StatusUnauthorized)
	expectStatus(t, s.do("POST", "/api/revoke", "", nil), http.StatusUnauthorized)
	expectStatus(t, s.do("POST", "/api/refresh", "Bearer "+rotated.RefreshToken, nil), http.StatusUnauthorized)

	t.Run("Expired", func(t *testing.T) {
		s.store.refreshTokens["expired"] = database.RefreshToken{Token: "expired", UserID: user.ID, ExpiresAt: time.Now().Add(-time.Minute)}
		expectStatus(t, s.do("POST", "/api/refresh", "Bearer expired", nil), http.StatusUnauthorized)
	})

	t.Run("Store error", func(t *testing.T) {
		s.store.err = errors.New("boom")
		defer func() { s.store.err = nil }()
		expectStatus(t, s.do("POST", "/api/refresh", "Bearer anything", nil), http.StatusInternalServerError)
		expectStatus(t, s.do("POST", "/api/revoke", "Bearer anything", nil), http.StatusInternalServerError)
	})
}

func TestPolkaWebhook(t *testing.T) {
	s := newTestServer(t)
	user := s.signUp("a@example.com")
	upgrade := func(userID string) map[string]any {
		return map[string]any{"event": "user.upgraded", "data": map[string]string{"user_id": userID}}
	}
	key := "ApiKey " + testPolkaKey

	tests := []struct {
		name          string
		authorization string
		body          any
		wantStatus    int
	}{
		{name: "Missing key", body: upgrade(user.ID.String()), wantStatus: http.StatusUnauthorized},
		{name: "Wrong key", authorization: "ApiKey wrong", body: upgrade(user.ID.String()), wantStatus: http.StatusUnauthorized},
		{name: "Wrong key with malformed body", authorization: "ApiKey wrong", body: "{", wantStatus: http.StatusUnauthorized},
		{name: "Malformed JSON", authorization: key, body: "{", wantStatus: http.StatusBadRequest},
		{name: "Other event", authorization: key, body: map[string]any{"event": "user.payment_failed"}, wantStatus: http.StatusNoContent},
		{name: "Invalid user ID", authorization: key, body: upgrade("nope"), wantStatus: http.StatusBadRequest},
		{name: "Unknown user", authorization: key, body: upgrade(uuid.NewString()), wantStatus: http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expectStatus(t, s.do("POST", "/api/polka/webhooks", tt.authorization, tt.body), tt.wantStatus)
		})
	}
	if s.store.users[user.ID].IsChirpyRed {
		t.Fatal("user upgraded before a valid upgrade event")
	}

	expectStatus(t, s.do("POST", "/api/polka/webhooks", key, upgrade(user.ID.String())), http.StatusNoContent)
	if !s.store.users[user.ID].IsChirpyRed {
		t.Error("user was not upgraded")
	}

	s.store.err = errors.New("boom")
	expectStatus(t, s.do("POST", "/api/polka/webhooks", key, upgrade(user.ID.String())), http.StatusInternalServerError)
}

func TestAdmin(t *testing.T) {
	s := newTestServer(t)
	s.signUp("a@example.com")

	expectStatus(t, s.do("GET", "/app/", "", nil), http.StatusOK)
	expectStatus(t, s.do("GET", "/app/assets/logo.png", "", nil), http.StatusOK)
	rec := s.do("GET", "/admin/metrics", "", nil)
	expectStatus(t, rec, http.StatusOK)
	if !strings.Contains(rec.Body.String(), "visited 2 times") {
		t.Errorf("metrics body = %q, want 2 visits", rec.Body)
	}

	expectStatus(t, s.do("POST", "/admin/reset", "", nil), http.StatusOK)
	if len(s.store.users) != 0 {
		t.Errorf("reset left %d users", len(s.store.users))
	}

	s.store.err = errors.New("boom")
	expectStatus(t, s.do("POST", "/admin/reset", "", nil), http.StatusInternalServerError)

	prod := New(newFakeStore(), config.Config{Platform: "production"}).Routes()
	rec = httptest.NewRecorder()
	prod.ServeHTTP(rec, httptest.NewRequest("POST", "/admin/reset", nil))
	expectStatus(t, rec, http.StatusForbidden)
}

func TestStaticFilesOnlyServeWebDir(t *testing.T) {
	s := newTestServer(t)
	for _, path := range []string{"/app/.env", "/app/go.mod", "/app/web.go"} {
		if rec := s.do("GET", path, "", nil); rec.Code != http.StatusNotFound {
			t.Errorf("GET %s: status = %d, want 404", path, rec.Code)
		}
	}
}

func TestBodySizeLimit(t *testing.T) {
	s := newTestServer(t)
	body := fmt.Sprintf(`{"email":"a@example.com","password":%q}`, strings.Repeat("a", maxBodyBytes))
	expectStatus(t, s.do("POST", "/api/users", "", body), http.StatusRequestEntityTooLarge)
}

func TestRecoverPanics(t *testing.T) {
	handler := recoverPanics(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	expectStatus(t, rec, http.StatusInternalServerError)

	t.Run("ErrAbortHandler is re-panicked", func(t *testing.T) {
		defer func() {
			if err := recover(); err != http.ErrAbortHandler {
				t.Errorf("recovered %v, want http.ErrAbortHandler", err)
			}
		}()
		abort := recoverPanics(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic(http.ErrAbortHandler)
		}))
		abort.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	})
}

func TestLogHandlerAddsRequestID(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(NewLogHandler(slog.NewJSONHandler(&buf, nil))).With("app", "chirpy").WithGroup("g")

	ctx := context.WithValue(context.Background(), requestIDKey, "req-123")
	logger.InfoContext(ctx, "hello", "k", "v")

	var record map[string]any
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		t.Fatalf("decoding log line %q: %v", buf.String(), err)
	}
	group, _ := record["g"].(map[string]any)
	if record["app"] != "chirpy" || group["request_id"] != "req-123" || group["k"] != "v" {
		t.Errorf("log record = %v", record)
	}
}

func TestRespondWithJSONMarshalError(t *testing.T) {
	rec := httptest.NewRecorder()
	respondWithJSON(rec, http.StatusOK, make(chan int))
	expectStatus(t, rec, http.StatusInternalServerError)
}
