package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/danielakinremi1-dev/chirpy-social-api/internal/api"
	"github.com/danielakinremi1-dev/chirpy-social-api/internal/config"
	"github.com/danielakinremi1-dev/chirpy-social-api/sql/schema"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	slog.SetDefault(slog.New(api.NewLogHandler(slog.NewJSONHandler(os.Stdout, nil))))

	if err := run(); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	// .env is optional; the environment can supply everything.
	_ = godotenv.Load()

	cfg, err := config.Load(os.Getenv)
	if err != nil {
		return err
	}

	db, err := sql.Open("postgres", cfg.DBURL)
	if err != nil {
		return err
	}
	defer db.Close()

	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	err = db.PingContext(pingCtx)
	cancel()
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}

	if cfg.MigrateOnStart {
		if err := schema.Up(context.Background(), db); err != nil {
			return fmt.Errorf("applying migrations: %w", err)
		}
		slog.Info("migrations applied")
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           api.New(api.NewSQLStore(db), cfg).Routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serveErr := make(chan error, 1)
	go func() {
		slog.Info("serving", "port", cfg.Port)
		serveErr <- srv.ListenAndServe()
	}()

	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
	}

	slog.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
