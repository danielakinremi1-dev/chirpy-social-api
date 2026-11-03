//go:build integration

package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/danielakinremi1-dev/chirpy-social-api/internal/config"
	"github.com/danielakinremi1-dev/chirpy-social-api/sql/schema"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

// startPostgres runs Postgres in a container and applies the migrations.
func startPostgres(t *testing.T) *sql.DB {
	t.Helper()
	ctx := context.Background()

	container, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("chirpy"),
		postgres.WithUsername("chirpy"),
		postgres.WithPassword("chirpy"),
		postgres.BasicWaitStrategies(),
	)
	testcontainers.CleanupContainer(t, container)
	if err != nil {
		t.Fatalf("starting Postgres: %v", err)
	}

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("getting connection string: %v", err)
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("opening database: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	if err := schema.Up(ctx, db); err != nil {
		t.Fatalf("applying migrations: %v", err)
	}
	return db
}

type client struct {
	t   *testing.T
	url string
}

func (c client) do(method, path, authorization string, body any) (int, http.Header, []byte) {
	c.t.Helper()
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			c.t.Fatalf("encoding body: %v", err)
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, c.url+path, reader)
	if err != nil {
		c.t.Fatalf("building request: %v", err)
	}
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		c.t.Fatalf("reading response: %v", err)
	}
	return resp.StatusCode, resp.Header, data
}

// expect sends a request, checks the status, and decodes the body into out
// when out is non-nil.
func (c client) expect(wantStatus int, method, path, authorization string, body, out any) http.Header {
	c.t.Helper()
	status, header, data := c.do(method, path, authorization, body)
	if status != wantStatus {
		c.t.Fatalf("%s %s: status = %d, want %d; body: %s", method, path, status, wantStatus, data)
	}
	if out != nil {
		if err := json.Unmarshal(data, out); err != nil {
			c.t.Fatalf("%s %s: decoding %q: %v", method, path, data, err)
		}
	}
	return header
}

func TestIntegration(t *testing.T) {
	db := startPostgres(t)

	t.Run("Migrations roll back and reapply", func(t *testing.T) {
		provider, err := schema.NewProvider(db)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := provider.DownTo(context.Background(), 0); err != nil {
			t.Fatalf("rolling back: %v", err)
		}
		if _, err := provider.Up(context.Background()); err != nil {
			t.Fatalf("reapplying: %v", err)
		}
	})

	a := New(NewSQLStore(db), config.Config{JWTSecret: testSecret, PolkaKey: testPolkaKey, Platform: "dev"})
	srv := httptest.NewServer(a.Routes())
	t.Cleanup(srv.Close)
	c := client{t: t, url: srv.URL}

	t.Run("Full flow", func(t *testing.T) {
		c := client{t: t, url: srv.URL}
		creds := credentials{Email: "alice@example.com", Password: testPassword}

		// Register.
		c.expect(http.StatusCreated, "POST", "/api/users", "", creds, nil)
		c.expect(http.StatusConflict, "POST", "/api/users", "", creds, nil)

		// Log in.
		var alice loginBody
		c.expect(http.StatusOK, "POST", "/api/login", "", creds, &alice)
		c.expect(http.StatusUnauthorized, "POST", "/api/login", "", credentials{Email: "alice@example.com", Password: "wrong-password"}, nil)

		// Post.
		var chirps []chirpResponse
		for i := range 3 {
			var chirp chirpResponse
			c.expect(http.StatusCreated, "POST", "/api/chirps", "Bearer "+alice.Token, map[string]string{"body": fmt.Sprint("chirp ", i)}, &chirp)
			chirps = append(chirps, chirp)
		}

		// List, paginating with the cursor.
		var page1, page2 []chirpResponse
		header := c.expect(http.StatusOK, "GET", "/api/chirps?limit=2&author_id="+alice.ID.String(), "", nil, &page1)
		cursor := header.Get("X-Next-Cursor")
		if len(page1) != 2 || cursor == "" {
			t.Fatalf("page 1: got %d chirps and cursor %q", len(page1), cursor)
		}
		header = c.expect(http.StatusOK, "GET", "/api/chirps?limit=2&author_id="+alice.ID.String()+"&cursor="+cursor, "", nil, &page2)
		if len(page2) != 1 || header.Get("X-Next-Cursor") != "" {
			t.Fatalf("page 2: got %d chirps and cursor %q", len(page2), header.Get("X-Next-Cursor"))
		}
		got := []chirpResponse{page1[0], page1[1], page2[0]}
		for i := range chirps {
			if got[i].ID != chirps[i].ID {
				t.Errorf("chirp %d: got %s, want %s", i, got[i].Body, chirps[i].Body)
			}
		}

		var desc []chirpResponse
		c.expect(http.StatusOK, "GET", "/api/chirps?sort=desc", "", nil, &desc)
		if len(desc) != 3 || desc[0].ID != chirps[2].ID {
			t.Errorf("descending list starts with %v, want %s", desc, chirps[2].Body)
		}

		// Refresh rotates the token.
		var rotated loginBody
		c.expect(http.StatusOK, "POST", "/api/refresh", "Bearer "+alice.RefreshToken, nil, &rotated)
		c.expect(http.StatusUnauthorized, "POST", "/api/refresh", "Bearer "+alice.RefreshToken, nil, nil)
		c.expect(http.StatusCreated, "POST", "/api/chirps", "Bearer "+rotated.Token, map[string]string{"body": "after refresh"}, nil)

		// Revoke.
		c.expect(http.StatusNoContent, "POST", "/api/revoke", "Bearer "+rotated.RefreshToken, nil, nil)
		c.expect(http.StatusUnauthorized, "POST", "/api/refresh", "Bearer "+rotated.RefreshToken, nil, nil)

		// Delete.
		path := "/api/chirps/" + chirps[0].ID.String()
		c.expect(http.StatusNoContent, "DELETE", path, "Bearer "+alice.Token, nil, nil)
		c.expect(http.StatusNotFound, "GET", path, "", nil, nil)
	})

	t.Run("Update user and webhook", func(t *testing.T) {
		c := client{t: t, url: srv.URL}
		var bob loginBody
		c.expect(http.StatusCreated, "POST", "/api/users", "", credentials{Email: "bob@example.com", Password: testPassword}, nil)
		c.expect(http.StatusOK, "POST", "/api/login", "", credentials{Email: "bob@example.com", Password: testPassword}, &bob)

		var before, after map[string]any
		c.expect(http.StatusConflict, "PUT", "/api/users", "Bearer "+bob.Token, credentials{Email: "alice@example.com", Password: testPassword}, nil)
		c.expect(http.StatusOK, "PUT", "/api/users", "Bearer "+bob.Token, credentials{Email: "robert@example.com", Password: testPassword}, &before)
		c.expect(http.StatusOK, "PUT", "/api/users", "Bearer "+bob.Token, credentials{Email: "robert@example.com", Password: "another-password"}, &after)
		if before["updated_at"] == after["updated_at"] {
			t.Error("updated_at did not change on update")
		}

		upgrade := map[string]any{"event": "user.upgraded", "data": map[string]string{"user_id": bob.ID.String()}}
		c.expect(http.StatusNoContent, "POST", "/api/polka/webhooks", "ApiKey "+testPolkaKey, upgrade, nil)
		var relogin loginBody
		c.expect(http.StatusOK, "POST", "/api/login", "", credentials{Email: "robert@example.com", Password: "another-password"}, &relogin)
		if !relogin.IsChirpyRed {
			t.Error("user was not upgraded")
		}
	})

	t.Run("Concurrent refreshes rotate once", func(t *testing.T) {
		var user loginBody
		c.expect(http.StatusCreated, "POST", "/api/users", "", credentials{Email: "carol@example.com", Password: testPassword}, nil)
		c.expect(http.StatusOK, "POST", "/api/login", "", credentials{Email: "carol@example.com", Password: testPassword}, &user)

		const attempts = 8
		statuses := make(chan int, attempts)
		var wg sync.WaitGroup
		for range attempts {
			// Not c.do: it calls t.Fatalf, which can't be used off the test goroutine.
			wg.Go(func() {
				req, err := http.NewRequest("POST", srv.URL+"/api/refresh", nil)
				if err != nil {
					t.Error(err)
					return
				}
				req.Header.Set("Authorization", "Bearer "+user.RefreshToken)
				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					t.Error(err)
					return
				}
				resp.Body.Close()
				statuses <- resp.StatusCode
			})
		}
		wg.Wait()
		close(statuses)

		counts := map[int]int{}
		for status := range statuses {
			counts[status]++
		}
		if counts[http.StatusOK] != 1 || counts[http.StatusUnauthorized] != attempts-1 {
			t.Errorf("status counts = %v, want one 200 and %d 401s", counts, attempts-1)
		}
	})

	t.Run("Reset cascades", func(t *testing.T) {
		c.expect(http.StatusOK, "POST", "/admin/reset", "", nil, nil)
		var chirps []chirpResponse
		c.expect(http.StatusOK, "GET", "/api/chirps", "", nil, &chirps)
		if len(chirps) != 0 {
			t.Errorf("%d chirps left after reset", len(chirps))
		}
	})
}
