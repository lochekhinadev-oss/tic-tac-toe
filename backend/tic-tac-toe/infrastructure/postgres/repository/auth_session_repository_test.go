package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"tic-tac-toe/infrastructure/postgres/datasource"
)

func TestAuthSessionRepositoryCreateFindAndRevoke(t *testing.T) {
	db := &authSessionDatabaseStub{}
	repo := NewAuthSessionRepository(db)
	now := time.Date(2026, 5, 16, 10, 0, 0, 0, time.UTC)
	session := RefreshSession{
		RefreshJTIHash: "hash-1",
		UserUUID:       "user-1",
		CreatedAt:      now,
		ExpiresAt:      now.Add(time.Hour),
		LastUsedAt:     now,
	}

	if err := repo.CreateSession(context.Background(), session); err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}
	if db.lastExecQuery != createAuthSessionQuery {
		t.Fatalf("unexpected create query: %q", db.lastExecQuery)
	}
	if db.createdSession.RefreshJTIHash != session.RefreshJTIHash || db.createdSession.UserUUID != session.UserUUID {
		t.Fatalf("unexpected created session: %#v", db.createdSession)
	}

	found, err := repo.FindActiveSessionByRefreshJTIHash(context.Background(), "hash-1")
	if err != nil {
		t.Fatalf("unexpected find error: %v", err)
	}
	if found.UserUUID != "user-1" || found.RefreshJTIHash != "hash-1" {
		t.Fatalf("unexpected found session: %#v", found)
	}
	if db.lastUsedAt.IsZero() {
		t.Fatal("expected last_used_at to be updated")
	}

	if err := repo.RevokeSession(context.Background(), "hash-1"); err != nil {
		t.Fatalf("unexpected revoke error: %v", err)
	}
	if db.revokedToken != "hash-1" {
		t.Fatalf("unexpected revoked token: %q", db.revokedToken)
	}

	rows, err := repo.RevokeSessionsByUserUUID(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("unexpected revoke all error: %v", err)
	}
	if rows != 1 {
		t.Fatalf("expected 1 revoked row, got %d", rows)
	}
	if db.revokedUser != "user-1" {
		t.Fatalf("unexpected revoked user: %q", db.revokedUser)
	}
}

func TestAuthSessionRepositoryErrors(t *testing.T) {
	repo := NewAuthSessionRepository(&authSessionDatabaseStub{queryErr: pgx.ErrNoRows})
	if _, err := repo.FindActiveSessionByRefreshJTIHash(context.Background(), "missing"); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestAuthSessionRepositoryUsesParameterizedQueries(t *testing.T) {
	t.Run("create session", func(t *testing.T) {
		db := &authSessionDatabaseStub{}
		repo := NewAuthSessionRepository(db)
		now := time.Date(2026, 5, 16, 10, 0, 0, 0, time.UTC)

		err := repo.CreateSession(context.Background(), RefreshSession{
			RefreshJTIHash: sqlInjectionPayload,
			UserUUID:       "user-1",
			CreatedAt:      now,
			ExpiresAt:      now.Add(time.Hour),
			LastUsedAt:     now,
		})
		if err != nil {
			t.Fatalf("unexpected create error: %v", err)
		}

		assertQueryDoesNotContainPayload(t, db.lastExecQuery)
		assertArgsContainPayload(t, db.lastExecArgs)
	})

	t.Run("find active session", func(t *testing.T) {
		db := &authSessionDatabaseStub{}
		repo := NewAuthSessionRepository(db)

		_, err := repo.FindActiveSessionByRefreshJTIHash(context.Background(), sqlInjectionPayload)
		if err != nil {
			t.Fatalf("unexpected find error: %v", err)
		}

		assertQueryDoesNotContainPayload(t, db.lastQueryQuery)
		assertArgsContainPayload(t, db.lastQueryArgs)
	})

	t.Run("revoke session", func(t *testing.T) {
		db := &authSessionDatabaseStub{}
		repo := NewAuthSessionRepository(db)

		err := repo.RevokeSession(context.Background(), sqlInjectionPayload)
		if err != nil {
			t.Fatalf("unexpected revoke error: %v", err)
		}

		assertQueryDoesNotContainPayload(t, db.lastExecQuery)
		assertArgsContainPayload(t, db.lastExecArgs)
	})

	t.Run("revoke all sessions", func(t *testing.T) {
		db := &authSessionDatabaseStub{}
		repo := NewAuthSessionRepository(db)

		_, err := repo.RevokeSessionsByUserUUID(context.Background(), sqlInjectionPayload)
		if err != nil {
			t.Fatalf("unexpected revoke all error: %v", err)
		}

		assertQueryDoesNotContainPayload(t, db.lastExecQuery)
		assertArgsContainPayload(t, db.lastExecArgs)
	})
}

type authSessionDatabaseStub struct {
	datasource.Database
	lastExecQuery  string
	lastExecArgs   []any
	lastQueryQuery string
	lastQueryArgs  []any
	createdSession RefreshSession
	lastUsedAt     time.Time
	revokedToken   string
	revokedUser    string
	queryErr       error
}

func (d *authSessionDatabaseStub) Exec(_ context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	d.lastExecQuery = sql
	d.lastExecArgs = arguments
	switch sql {
	case createAuthSessionQuery:
		d.createdSession = RefreshSession{
			RefreshJTIHash: arguments[0].(string),
			UserUUID:       arguments[1].(string),
			CreatedAt:      arguments[2].(time.Time),
			ExpiresAt:      arguments[3].(time.Time),
			LastUsedAt:     arguments[4].(time.Time),
		}
	case revokeAuthSessionQuery:
		d.revokedToken = arguments[0].(string)
	case revokeAllAuthSessionsQuery:
		d.revokedUser = arguments[0].(string)
	default:
		return pgconn.CommandTag{}, errors.New("unexpected query")
	}
	return pgconn.NewCommandTag("UPDATE 1"), nil
}

func (d *authSessionDatabaseStub) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	d.lastQueryQuery = sql
	d.lastQueryArgs = args
	if d.queryErr != nil {
		return authSessionRowStub{err: d.queryErr}
	}
	if sql != findActiveAuthSessionQuery {
		return authSessionRowStub{err: errors.New("unexpected query")}
	}
	d.lastUsedAt = time.Now().UTC()
	return authSessionRowStub{
		session: RefreshSession{
			RefreshJTIHash: args[0].(string),
			UserUUID:       "user-1",
			CreatedAt:      time.Now().UTC(),
			ExpiresAt:      time.Now().UTC().Add(time.Hour),
			LastUsedAt:     d.lastUsedAt,
		},
	}
}

func (d *authSessionDatabaseStub) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, nil
}

func (d *authSessionDatabaseStub) Ping(context.Context) error { return nil }

type authSessionRowStub struct {
	session RefreshSession
	err     error
}

func (r authSessionRowStub) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	setAuthSessionString(dest[0], r.session.RefreshJTIHash)
	setAuthSessionString(dest[1], r.session.UserUUID)
	setAuthSessionTime(dest[2], r.session.CreatedAt)
	setAuthSessionTime(dest[3], r.session.ExpiresAt)
	setAuthSessionTime(dest[4], r.session.LastUsedAt)
	return nil
}

func setAuthSessionString(dest any, value string) {
	switch target := dest.(type) {
	case *string:
		*target = value
	case *sql.NullString:
		target.String = value
		target.Valid = true
	}
}

func setAuthSessionTime(dest any, value time.Time) {
	switch target := dest.(type) {
	case *time.Time:
		*target = value
	case *sql.NullTime:
		target.Time = value
		target.Valid = true
	}
}
