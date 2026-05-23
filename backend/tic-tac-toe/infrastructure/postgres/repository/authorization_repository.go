package repository

import (
	"context"
	"database/sql"
	"strings"

	googleuuid "github.com/google/uuid"

	"tic-tac-toe/app/domain"
	"tic-tac-toe/infrastructure/postgres/datasource"
)

const (
	assignRoleToUserQuery = `
WITH inserted AS (
	INSERT INTO user_roles (user_uuid, role_name)
	VALUES ($1, $2)
	ON CONFLICT DO NOTHING
	RETURNING user_uuid
)
UPDATE users
SET authz_version = authz_version + 1
WHERE uuid = $1 AND EXISTS (SELECT 1 FROM inserted)`
	revokeRoleFromUserQuery = `
WITH deleted AS (
	DELETE FROM user_roles
	WHERE user_uuid = $1 AND role_name = $2
	RETURNING user_uuid
)
UPDATE users
SET authz_version = authz_version + 1
WHERE uuid = $1 AND EXISTS (SELECT 1 FROM deleted)`
	loadPrincipalVersionQuery = `
SELECT authz_version
FROM users
WHERE uuid = $1 AND deleted_at IS NULL`
	loadPrincipalRolesQuery = `
SELECT role_name
FROM user_roles
WHERE user_uuid = $1
ORDER BY role_name`
	loadPrincipalPermissionsQuery = `
SELECT DISTINCT permissions.resource, permissions.action
FROM user_roles
JOIN role_permissions ON role_permissions.role_name = user_roles.role_name
JOIN permissions ON permissions.resource = role_permissions.resource AND permissions.action = role_permissions.action
WHERE user_roles.user_uuid = $1
ORDER BY permissions.resource, permissions.action`
)

type PostgresAuthorizationRepository struct {
	db datasource.Database
}

func NewAuthorizationRepository(db datasource.Database) domain.AuthorizationRepository {
	return &PostgresAuthorizationRepository{db: db}
}

func (r *PostgresAuthorizationRepository) AssignRoleToUser(ctx context.Context, userUUID googleuuid.UUID, roleName string) error {
	logRepository("assign role", "user_uuid", userUUID, "role", roleName)
	if userUUID == googleuuid.Nil || strings.TrimSpace(roleName) == "" {
		return ErrInvalidDatabaseRow
	}

	if _, err := r.db.Exec(ctx, assignRoleToUserQuery, userUUID.String(), roleName); err != nil {
		logRepository("assign role failed", "user_uuid", userUUID, "role", roleName, "error", err)
		return err
	}
	logRepository("assign role ok", "user_uuid", userUUID, "role", roleName)
	return nil
}

func (r *PostgresAuthorizationRepository) RevokeRoleFromUser(ctx context.Context, userUUID googleuuid.UUID, roleName string) error {
	logRepository("revoke role", "user_uuid", userUUID, "role", roleName)
	if userUUID == googleuuid.Nil || strings.TrimSpace(roleName) == "" {
		return ErrInvalidDatabaseRow
	}

	if _, err := r.db.Exec(ctx, revokeRoleFromUserQuery, userUUID.String(), roleName); err != nil {
		logRepository("revoke role failed", "user_uuid", userUUID, "role", roleName, "error", err)
		return err
	}
	logRepository("revoke role ok", "user_uuid", userUUID, "role", roleName)
	return nil
}

func (r *PostgresAuthorizationRepository) LoadPrincipalVersion(ctx context.Context, userUUID googleuuid.UUID) (int64, error) {
	logRepository("load principal version", "user_uuid", userUUID)
	if userUUID == googleuuid.Nil {
		return 0, ErrInvalidDatabaseRow
	}

	var scannedVersion sql.NullInt64
	if err := r.db.QueryRow(ctx, loadPrincipalVersionQuery, userUUID.String()).Scan(&scannedVersion); err != nil {
		logRepository("load principal version failed", "user_uuid", userUUID, "error", err)
		return 0, err
	}
	authzVersion, err := requiredInt64("users.authz_version", scannedVersion)
	if err != nil {
		logRepository("load principal invalid version row", "user_uuid", userUUID, "error", err)
		return 0, err
	}
	logRepository("load principal version ok", "user_uuid", userUUID, "version", authzVersion)
	return authzVersion, nil
}

func (r *PostgresAuthorizationRepository) LoadPrincipal(ctx context.Context, userUUID googleuuid.UUID) (domain.Principal, error) {
	logRepository("load principal", "user_uuid", userUUID)
	if userUUID == googleuuid.Nil {
		return domain.Principal{}, ErrInvalidDatabaseRow
	}

	authzVersion, err := r.LoadPrincipalVersion(ctx, userUUID)
	if err != nil {
		return domain.Principal{}, err
	}

	roleRows, err := r.db.Query(ctx, loadPrincipalRolesQuery, userUUID.String())
	if err != nil {
		logRepository("load principal roles failed", "user_uuid", userUUID, "error", err)
		return domain.Principal{}, err
	}
	defer roleRows.Close()

	roles := make([]string, 0, 4)
	for roleRows.Next() {
		var roleName sql.NullString
		if err := roleRows.Scan(&roleName); err != nil {
			logRepository("load principal role scan failed", "user_uuid", userUUID, "error", err)
			return domain.Principal{}, err
		}
		value, err := requiredString("user_roles.role_name", roleName)
		if err != nil {
			logRepository("load principal invalid role row", "user_uuid", userUUID, "error", err)
			return domain.Principal{}, err
		}
		roles = append(roles, value)
	}
	if err := roleRows.Err(); err != nil {
		logRepository("load principal roles rows err", "user_uuid", userUUID, "error", err)
		return domain.Principal{}, err
	}

	permissionRows, err := r.db.Query(ctx, loadPrincipalPermissionsQuery, userUUID.String())
	if err != nil {
		logRepository("load principal permissions failed", "user_uuid", userUUID, "error", err)
		return domain.Principal{}, err
	}
	defer permissionRows.Close()

	permissions := make([]domain.Permission, 0, 8)
	for permissionRows.Next() {
		var resource sql.NullString
		var action sql.NullString
		if err := permissionRows.Scan(&resource, &action); err != nil {
			logRepository("load principal permission scan failed", "user_uuid", userUUID, "error", err)
			return domain.Principal{}, err
		}
		resourceValue, err := requiredString("permissions.resource", resource)
		if err != nil {
			logRepository("load principal invalid resource row", "user_uuid", userUUID, "error", err)
			return domain.Principal{}, err
		}
		actionValue, err := requiredString("permissions.action", action)
		if err != nil {
			logRepository("load principal invalid action row", "user_uuid", userUUID, "error", err)
			return domain.Principal{}, err
		}
		permissions = append(permissions, domain.Permission{Resource: resourceValue, Action: actionValue})
	}
	if err := permissionRows.Err(); err != nil {
		logRepository("load principal permissions rows err", "user_uuid", userUUID, "error", err)
		return domain.Principal{}, err
	}

	if len(roles) == 0 {
		logRepository("load principal no roles", "user_uuid", userUUID)
	}

	principal := domain.Principal{
		UserUUID:     userUUID.String(),
		AuthzVersion: authzVersion,
		Roles:        roles,
		Permissions:  permissions,
	}
	logRepository("load principal ok", "user_uuid", userUUID, "version", authzVersion, "roles", len(roles), "permissions", len(permissions))
	return principal, nil
}
