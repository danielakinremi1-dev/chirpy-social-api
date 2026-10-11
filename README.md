# Chirpy

A RESTful API built with Go and PostgreSQL that simulates the backend of a microblogging platform (similar to Twitter/X).  

## Goal 

This project was built to practice core backend architecture and Go fundamentals without relying on heavy web frameworks. Key areas explored include:

- **HTTP Server Routing & Middleware:** Structuring idiomatic Go HTTP handlers, custom routing, and request metrics.
- **Authentication & Security:** Implementing password hashing with `bcrypt`, stateless authentication with JWTs, and secure refresh token rotation.
- **Database Management:** Writing raw SQL queries and generating type-safe Go database code using `sqlc` and PostgreSQL.
- **RESTful API Design:** Designing clean endpoints with CRUD operations, webhook handling, and query-parameter-based sorting and filtering.

## Install and run

Requirements: Go 1.27.1 or newer, PostgreSQL, and the [Goose](https://github.com/pressly/goose) migration CLI.

1. Clone the repository and enter the project directory:

   ```sh
   git clone https://github.com/danielakinremi1-dev/chirpy.git
   cd chirpy
   ```

   Install Goose if needed: `go install github.com/pressly/goose/v3/cmd/goose@latest`.
2. Create an empty PostgreSQL database named `chirpy`.
3. Set the following environment variables in your shell or in a `.env` file at the project root:

   ```dotenv
   DB_URL=postgres://USER:PASSWORD@localhost:5432/chirpy?sslmode=disable
   PLATFORM=dev
   SECRET=replace-with-a-long-random-secret
   POLKA_KEY=replace-with-your-webhook-api-key
   ```

   `PLATFORM=dev` enables the destructive admin reset endpoint; use another value outside local development. Keep `SECRET` and `POLKA_KEY` private.
4. Apply the database migrations:

   ```sh
   goose -dir sql/schema postgres "$DB_URL" up
   ```
5. Start the server:

   ```sh
   go run .
   ```

The server listens on port `8080`. Go downloads the project dependencies on the first run. 

## HTTP handlers

JSON endpoints accept and return JSON unless noted. Authenticated requests use `Authorization: Bearer <token>`; the webhook uses `Authorization: ApiKey <key>`.

| Method and path | Use |
| --- | --- |
| `GET /api/healthz` | Readiness check. |
| `POST /api/users` | Create an account. Body: `{"email":"...","password":"..."}`. |
| `PUT /api/users` | Update the authenticated user's email and password. |
| `POST /api/login` | Log in with email and password; returns a one-hour access token and a refresh token (valid for 60 days). |
| `POST /api/refresh` | Validate a refresh token (Bearer) for a new access token. |
| `POST /api/revoke` | Revoke a refresh token (Bearer). |
| `POST /api/chirps` | Create a chirp with `{"body":"..."}`. Requires an access token; text is limited to 140 characters. |
| `GET /api/chirps` | List chirps. Optional `author_id=<uuid>` filters by author; `sort=desc` sorts newest first (default is oldest first). |
| `GET /api/chirps/{chirpID}` | Get one chirp. |
| `DELETE /api/chirps/{chirpID}` | Delete your own chirp. Requires an access token. |
| `POST /api/polka/webhooks` | Process the `user.upgraded` event and grant Chirpy Red subscription. Requires the configured API key. |
| `GET /admin/metrics` | Show the number of requests served from `/app/`. |
| `POST /admin/reset` | Reset that count and delete all users and related data. Available only when `PLATFORM=dev`. |
| `GET /app/` | Serve static frontend assets. |
