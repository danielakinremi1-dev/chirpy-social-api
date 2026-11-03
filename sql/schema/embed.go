// Package schema embeds the goose migrations in this directory so the server
// and tests can apply them without the goose CLI.
package schema

import (
	"context"
	"database/sql"
	"embed"

	"github.com/pressly/goose/v3"
)

//go:embed *.sql
var FS embed.FS

// NewProvider returns a goose provider for the embedded migrations.
func NewProvider(db *sql.DB) (*goose.Provider, error) {
	return goose.NewProvider(goose.DialectPostgres, db, FS)
}

// Up applies any pending migrations.
func Up(ctx context.Context, db *sql.DB) error {
	provider, err := NewProvider(db)
	if err != nil {
		return err
	}
	_, err = provider.Up(ctx)
	return err
}
