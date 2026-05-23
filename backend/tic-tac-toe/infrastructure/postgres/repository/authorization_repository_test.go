package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	googleuuid "github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestAuthorizationRepositoryAssignRoleToUserBumpsVersion(t *testing.T) {
	db := &authorizationDatabaseStub{execTag: pgconn.NewCommandTag("UPDATE 1")}
	repo := NewAuthorizationRepository(db)
	userUUID := googleuuid.MustParse("123e4567-e89b-42d3-a456-426614174001")

	if err := repo.AssignRoleToUser(context.Background(), userUUID, sqlInjectionPayload); err != nil {
		t.Fatalf("unexpected assign error: %v", err)
	}

	if !strings.Contains(db.lastExecQuery, "authz_version = authz_version + 1") {
		t.Fatalf("expected authz version bump, got query: %s", db.lastExecQuery)
	}
	assertArgsContainPayload(t, db.lastExecArgs)
}

func TestAuthorizationRepositoryRevokeRoleFromUserBumpsVersion(t *testing.T) {
	db := &authorizationDatabaseStub{execTag: pgconn.NewCommandTag("UPDATE 1")}
	repo := NewAuthorizationRepository(db)
	userUUID := googleuuid.MustParse("123e4567-e89b-42d3-a456-426614174001")

	if err := repo.RevokeRoleFromUser(context.Background(), userUUID, sqlInjectionPayload); err != nil {
		t.Fatalf("unexpected revoke error: %v", err)
	}

	if !strings.Contains(db.lastExecQuery, "authz_version = authz_version + 1") {
		t.Fatalf("expected authz version bump, got query: %s", db.lastExecQuery)
	}
	assertArgsContainPayload(t, db.lastExecArgs)
}

func TestAuthorizationRepositoryLoadPrincipalIncludesVersion(t *testing.T) {
	db := &authorizationDatabaseStub{
		versionRow: authzRowStub{values: []any{int64(7)}},
		roleRows: &authzRowsStub{
			scans: [][]any{
				{"player"},
				{"admin"},
			},
		},
		permissionRows: &authzRowsStub{
			scans: [][]any{
				{"games", "create"},
				{"games", "read"},
				{"users", "read_self"},
			},
		},
	}
	repo := NewAuthorizationRepository(db)
	userUUID := googleuuid.MustParse("123e4567-e89b-42d3-a456-426614174001")

	principal, err := repo.LoadPrincipal(context.Background(), userUUID)
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}

	if principal.AuthzVersion != 7 {
		t.Fatalf("expected authz version 7, got %d", principal.AuthzVersion)
	}
	if len(principal.Roles) != 2 || len(principal.Permissions) != 3 {
		t.Fatalf("unexpected principal: %#v", principal)
	}
}

func TestAuthorizationRepositoryLoadsVersionErrors(t *testing.T) {
	repo := NewAuthorizationRepository(&authorizationDatabaseStub{versionErr: errors.New("db failed")})

	if _, err := repo.LoadPrincipal(context.Background(), googleuuid.Nil); err == nil {
		t.Fatal("expected error")
	}
}

type authorizationDatabaseStub struct {
	lastExecQuery  string
	lastExecArgs   []any
	execTag        pgconn.CommandTag
	versionRow     authzRowStub
	versionErr     error
	roleRows       *authzRowsStub
	permissionRows *authzRowsStub
}

func (d *authorizationDatabaseStub) Exec(_ context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	d.lastExecQuery = sql
	d.lastExecArgs = arguments
	if d.execTag.String() != "" {
		return d.execTag, nil
	}
	return pgconn.NewCommandTag("UPDATE 1"), nil
}

func (d *authorizationDatabaseStub) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	if d.versionErr != nil {
		return authzRowStub{err: d.versionErr}
	}
	return d.versionRow
}

func (d *authorizationDatabaseStub) Query(_ context.Context, sql string, args ...any) (pgx.Rows, error) {
	if strings.Contains(sql, "DISTINCT permissions.resource") {
		return d.permissionRows, nil
	}
	return d.roleRows, nil
}

func (d *authorizationDatabaseStub) Ping(context.Context) error { return nil }

func (d *authorizationDatabaseStub) Close() {}

type authzRowStub struct {
	values []any
	err    error
}

func (r authzRowStub) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	for i := range dest {
		switch target := dest[i].(type) {
		case *sql.NullInt64:
			target.Int64 = r.values[i].(int64)
			target.Valid = true
		case *sql.NullString:
			target.String = r.values[i].(string)
			target.Valid = true
		}
	}
	return nil
}

type authzRowsStub struct {
	scans [][]any
	index int
}

func (r *authzRowsStub) Close() {}

func (r *authzRowsStub) Err() error { return nil }

func (r *authzRowsStub) CommandTag() pgconn.CommandTag { return pgconn.CommandTag{} }

func (r *authzRowsStub) FieldDescriptions() []pgconn.FieldDescription { return nil }

func (r *authzRowsStub) Next() bool {
	if r.index >= len(r.scans) {
		return false
	}
	r.index++
	return true
}

func (r *authzRowsStub) Scan(dest ...any) error {
	values := r.scans[r.index-1]
	for i := range dest {
		switch target := dest[i].(type) {
		case *sql.NullString:
			target.String = values[i].(string)
			target.Valid = true
		}
	}
	return nil
}

func (r *authzRowsStub) Values() ([]any, error) { return nil, nil }

func (r *authzRowsStub) RawValues() [][]byte { return nil }

func (r *authzRowsStub) Conn() *pgx.Conn { return nil }
