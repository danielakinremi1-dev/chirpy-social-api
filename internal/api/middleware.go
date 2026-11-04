package api

import (
	"context"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/danielakinremi1-dev/chirpy-social-api/internal/auth"
	"github.com/google/uuid"
)

const maxBodyBytes = 1 << 20

type contextKey int

const (
	userIDKey contextKey = iota
	requestIDKey
)

// requireAuth rejects requests without a valid access token and stores the
// authenticated user's ID in the request context.
func (a *API) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, err := auth.GetBearerToken(r.Header)
		if err != nil {
			respondWithError(w, r, http.StatusUnauthorized, "Missing access token", err)
			return
		}
		userID, err := auth.ValidateJWT(token, a.jwtSecret)
		if err != nil {
			respondWithError(w, r, http.StatusUnauthorized, "Invalid access token", err)
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), userIDKey, userID)))
	}
}

// userIDFromContext returns the user ID stored by requireAuth.
func userIDFromContext(ctx context.Context) uuid.UUID {
	userID, _ := ctx.Value(userIDKey).(uuid.UUID)
	return userID
}

func (a *API) metricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

// withRequestID gives each request an ID, returned in the X-Request-ID header
// and added to every log line written with the request's context.
func withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := uuid.NewString()
		w.Header().Set("X-Request-ID", requestID)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey, requestID)))
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (rec *statusRecorder) WriteHeader(code int) {
	if rec.status == 0 {
		rec.status = code
	}
	rec.ResponseWriter.WriteHeader(code)
}

func (rec *statusRecorder) Write(b []byte) (int, error) {
	if rec.status == 0 {
		rec.status = http.StatusOK
	}
	return rec.ResponseWriter.Write(b)
}

func (rec *statusRecorder) Unwrap() http.ResponseWriter {
	return rec.ResponseWriter
}

func (a *API) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(rec, r)
		if rec.status == 0 {
			rec.status = http.StatusOK
		}
		slog.InfoContext(r.Context(), "request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"client_ip", clientIP(r, a.trustProxy),
			"duration_ms", time.Since(start).Milliseconds(),
		)
		// TEMPORARY: find out which proxy headers Render sends. Remove after.
		if a.trustProxy && r.URL.Path != "/api/healthz" {
			slog.InfoContext(r.Context(), "proxy headers",
				"remote_addr", r.RemoteAddr,
				"x_forwarded_for", r.Header.Values("X-Forwarded-For"),
				"true_client_ip", r.Header.Get("True-Client-IP"),
				"cf_connecting_ip", r.Header.Get("CF-Connecting-IP"),
				"x_real_ip", r.Header.Get("X-Real-IP"),
				"forwarded", r.Header.Get("Forwarded"),
			)
		}
	})
}

func recoverPanics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			err := recover()
			if err == nil {
				return
			}
			if err == http.ErrAbortHandler {
				panic(err)
			}
			slog.ErrorContext(r.Context(), "panic", "error", err, "stack", string(debug.Stack()))
			respondWithError(w, r, http.StatusInternalServerError, "Internal server error", nil)
		}()
		next.ServeHTTP(w, r)
	})
}

func limitBodySize(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		next.ServeHTTP(w, r)
	})
}

// NewLogHandler wraps h so records logged with a request's context include
// its request ID.
func NewLogHandler(h slog.Handler) slog.Handler {
	return logHandler{h}
}

type logHandler struct {
	slog.Handler
}

func (h logHandler) Handle(ctx context.Context, record slog.Record) error {
	if requestID, ok := ctx.Value(requestIDKey).(string); ok {
		record.AddAttrs(slog.String("request_id", requestID))
	}
	return h.Handler.Handle(ctx, record)
}

func (h logHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return logHandler{h.Handler.WithAttrs(attrs)}
}

func (h logHandler) WithGroup(name string) slog.Handler {
	return logHandler{h.Handler.WithGroup(name)}
}
