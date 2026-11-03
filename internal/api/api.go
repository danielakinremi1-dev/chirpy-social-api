// Package api implements Chirpy's HTTP handlers.
package api

import (
	"context"
	"database/sql"
	"sync/atomic"
	"time"

	"github.com/danielakinremi1-dev/chirpy-social-api/internal/config"
	"github.com/danielakinremi1-dev/chirpy-social-api/internal/database"
)

// Store is the data access the handlers need. Tests substitute a fake.
type Store interface {
	database.Querier
	// InTx runs fn inside a transaction, committing only if fn returns nil.
	InTx(ctx context.Context, fn func(q database.Querier) error) error
}

// SQLStore is the Postgres-backed Store.
type SQLStore struct {
	*database.Queries
	db *sql.DB
}

func NewSQLStore(db *sql.DB) *SQLStore {
	return &SQLStore{Queries: database.New(db), db: db}
}

func (s *SQLStore) InTx(ctx context.Context, fn func(q database.Querier) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	// Rollback is a no-op once Commit has succeeded.
	defer func() { _ = tx.Rollback() }()

	if err := fn(s.WithTx(tx)); err != nil {
		return err
	}
	return tx.Commit()
}

type API struct {
	store          Store
	jwtSecret      string
	polkaKey       string
	platform       string
	fileserverHits atomic.Int32
	loginLimiter   *ipRateLimiter
	trustProxy     bool
}

func New(store Store, cfg config.Config) *API {
	return &API{
		store:      store,
		jwtSecret:  cfg.JWTSecret,
		polkaKey:   cfg.PolkaKey,
		platform:   cfg.Platform,
		trustProxy: cfg.TrustProxy,
		// 10 attempts per IP, refilled at one every 6 seconds.
		loginLimiter: newIPRateLimiter(6*time.Second, 10, cfg.TrustProxy),
	}
}
