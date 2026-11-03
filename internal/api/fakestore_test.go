package api

import (
	"bytes"
	"context"
	"database/sql"
	"slices"
	"sync"
	"time"

	"github.com/danielakinremi1-dev/chirpy-social-api/internal/database"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

// fakeStore is an in-memory Store that mirrors the SQL queries' behavior
// closely enough for handler tests.
type fakeStore struct {
	mu            sync.Mutex
	now           time.Time
	users         map[uuid.UUID]database.User
	chirps        map[uuid.UUID]database.Chirp
	refreshTokens map[string]database.RefreshToken

	// err, when set, is returned by every query.
	err error
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		now:           time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		users:         make(map[uuid.UUID]database.User),
		chirps:        make(map[uuid.UUID]database.Chirp),
		refreshTokens: make(map[string]database.RefreshToken),
	}
}

// tick returns a strictly increasing timestamp so ordering is deterministic.
func (f *fakeStore) tick() time.Time {
	f.now = f.now.Add(time.Millisecond)
	return f.now
}

func (f *fakeStore) InTx(ctx context.Context, fn func(q database.Querier) error) error {
	return fn(f)
}

func (f *fakeStore) CreateChirp(ctx context.Context, arg database.CreateChirpParams) (database.Chirp, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return database.Chirp{}, f.err
	}
	now := f.tick()
	chirp := database.Chirp{ID: uuid.New(), CreatedAt: now, UpdatedAt: now, Body: arg.Body, UserID: arg.UserID}
	f.chirps[chirp.ID] = chirp
	return chirp, nil
}

func (f *fakeStore) CreateRefresh(ctx context.Context, arg database.CreateRefreshParams) (database.RefreshToken, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return database.RefreshToken{}, f.err
	}
	now := f.tick()
	token := database.RefreshToken{Token: arg.Token, CreatedAt: now, UpdatedAt: now, UserID: arg.UserID, ExpiresAt: arg.ExpiresAt}
	f.refreshTokens[token.Token] = token
	return token, nil
}

func (f *fakeStore) CreateUser(ctx context.Context, arg database.CreateUserParams) (database.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return database.User{}, f.err
	}
	if f.emailTaken(arg.Email, uuid.Nil) {
		return database.User{}, &pq.Error{Code: "23505"}
	}
	now := f.tick()
	user := database.User{ID: uuid.New(), CreatedAt: now, UpdatedAt: now, Email: arg.Email, HashedPassword: arg.HashedPassword}
	f.users[user.ID] = user
	return user, nil
}

func (f *fakeStore) DeleteAllUsers(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	clear(f.users)
	clear(f.chirps)
	clear(f.refreshTokens)
	return nil
}

func (f *fakeStore) DeleteChirp(ctx context.Context, id uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	delete(f.chirps, id)
	return nil
}

func (f *fakeStore) GetChirp(ctx context.Context, id uuid.UUID) (database.Chirp, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return database.Chirp{}, f.err
	}
	chirp, ok := f.chirps[id]
	if !ok {
		return database.Chirp{}, sql.ErrNoRows
	}
	return chirp, nil
}

func (f *fakeStore) GetUser(ctx context.Context, email string) (database.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return database.User{}, f.err
	}
	for _, user := range f.users {
		if user.Email == email {
			return user, nil
		}
	}
	return database.User{}, sql.ErrNoRows
}

func (f *fakeStore) GetUserFromRefreshToken(ctx context.Context, token string) (database.RefreshToken, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return database.RefreshToken{}, f.err
	}
	refreshToken, ok := f.refreshTokens[token]
	if !ok {
		return database.RefreshToken{}, sql.ErrNoRows
	}
	return refreshToken, nil
}

func (f *fakeStore) ListChirpsAsc(ctx context.Context, arg database.ListChirpsAscParams) ([]database.Chirp, error) {
	return f.listChirps(arg.AuthorID, arg.CursorCreatedAt, arg.CursorID, arg.RowLimit, false)
}

func (f *fakeStore) ListChirpsDesc(ctx context.Context, arg database.ListChirpsDescParams) ([]database.Chirp, error) {
	return f.listChirps(arg.AuthorID, arg.CursorCreatedAt, arg.CursorID, arg.RowLimit, true)
}

func (f *fakeStore) listChirps(authorID uuid.NullUUID, cursorCreatedAt sql.NullTime, cursorID uuid.NullUUID, limit int32, desc bool) ([]database.Chirp, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return nil, f.err
	}

	// compare orders chirps by (created_at, id), like the SQL row comparison.
	compare := func(createdAtA time.Time, idA uuid.UUID, createdAtB time.Time, idB uuid.UUID) int {
		if c := createdAtA.Compare(createdAtB); c != 0 {
			return c
		}
		return bytes.Compare(idA[:], idB[:])
	}

	var chirps []database.Chirp
	for _, c := range f.chirps {
		if authorID.Valid && c.UserID != authorID.UUID {
			continue
		}
		if cursorCreatedAt.Valid {
			cmp := compare(c.CreatedAt, c.ID, cursorCreatedAt.Time, cursorID.UUID)
			if (!desc && cmp <= 0) || (desc && cmp >= 0) {
				continue
			}
		}
		chirps = append(chirps, c)
	}

	slices.SortFunc(chirps, func(a, b database.Chirp) int {
		if desc {
			return compare(b.CreatedAt, b.ID, a.CreatedAt, a.ID)
		}
		return compare(a.CreatedAt, a.ID, b.CreatedAt, b.ID)
	})
	if len(chirps) > int(limit) {
		chirps = chirps[:limit]
	}
	return chirps, nil
}

func (f *fakeStore) RevokeRefreshToken(ctx context.Context, token string) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return 0, f.err
	}
	refreshToken, ok := f.refreshTokens[token]
	if !ok || refreshToken.RevokedAt.Valid {
		return 0, nil
	}
	now := f.tick()
	refreshToken.RevokedAt = sql.NullTime{Time: now, Valid: true}
	refreshToken.UpdatedAt = now
	f.refreshTokens[token] = refreshToken
	return 1, nil
}

func (f *fakeStore) UpdateUser(ctx context.Context, arg database.UpdateUserParams) (database.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return database.User{}, f.err
	}
	user, ok := f.users[arg.ID]
	if !ok {
		return database.User{}, sql.ErrNoRows
	}
	if f.emailTaken(arg.Email, arg.ID) {
		return database.User{}, &pq.Error{Code: "23505"}
	}
	user.Email = arg.Email
	user.HashedPassword = arg.HashedPassword
	user.UpdatedAt = f.tick()
	f.users[user.ID] = user
	return user, nil
}

func (f *fakeStore) UpgradeUser(ctx context.Context, id uuid.UUID) (database.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return database.User{}, f.err
	}
	user, ok := f.users[id]
	if !ok {
		return database.User{}, sql.ErrNoRows
	}
	user.IsChirpyRed = true
	f.users[id] = user
	return user, nil
}

// emailTaken reports whether a user other than except has the email.
func (f *fakeStore) emailTaken(email string, except uuid.UUID) bool {
	for _, user := range f.users {
		if user.Email == email && user.ID != except {
			return true
		}
	}
	return false
}
