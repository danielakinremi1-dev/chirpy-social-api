package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/danielakinremi1-dev/chirpy-social-api/internal/api"
	"github.com/danielakinremi1-dev/chirpy-social-api/internal/config"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	godotenv.Load()

	cfg, err := config.Load(os.Getenv)
	if err != nil {
		log.Fatal(err)
	}

	db, err := sql.Open("postgres", cfg.DBURL)
	if err != nil {
		log.Fatal(err)
	}

	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	err = db.PingContext(pingCtx)
	cancel()
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}

	const port = "8080"

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: api.New(api.NewSQLStore(db), cfg).Routes(),
	}

	log.Printf("Serving on port: %s\n", port)
	log.Fatal(srv.ListenAndServe())
}
