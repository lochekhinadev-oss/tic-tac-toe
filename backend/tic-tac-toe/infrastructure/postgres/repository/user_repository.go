package repository

import (
	"context"
	"database/sql"
	"errors"

	googleuuid "github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"tic-tac-toe/app/domain"
	"tic-tac-toe/infrastructure/postgres/datasource"
	"tic-tac-toe/infrastructure/postgres/mapper"
)

var (
	ErrUserNotFound       = domain.ErrUserNotFound
	ErrLoginAlreadyExists = errors.New("login already exists")
)

const (
	saveUserQuery = `WITH inserted AS (
    INSERT INTO users (uuid, login, password) VALUES ($1, $2, $3)
    RETURNING uuid
)
INSERT INTO user_roles (user_uuid, role_name)
SELECT uuid, $4 FROM inserted`
	getUserByLoginQuery     = "SELECT uuid, login, password FROM users WHERE login = $1 AND deleted_at IS NULL"
	getUserByUUIDQuery      = "SELECT uuid, login, password FROM users WHERE uuid = $1 AND deleted_at IS NULL"
	updateUserPasswordQuery = "UPDATE users SET password = $2 WHERE uuid = $1 AND deleted_at IS NULL"
	deleteUserQuery         = "UPDATE users SET deleted_at = NOW() WHERE uuid = $1 AND deleted_at IS NULL"
)

type PostgresUserRepository struct {
	db datasource.Database
}

func NewUserRepository(db datasource.Database) domain.UserRepository {
	return &PostgresUserRepository{db: db}
}

func (r *PostgresUserRepository) SaveUser(ctx context.Context, user domain.User) error {
	logRepository("save user", "uuid", user.UUID, "login", user.Login)
	datasourceUser := mapper.ToDatasourceUser(user)

	_, err := r.db.Exec(
		ctx,
		saveUserQuery,
		datasourceUser.UUID,
		datasourceUser.Login,
		datasourceUser.Password,
		domain.DefaultPlayerRole,
	)
	if isUniqueViolation(err) {
		logRepository("save user duplicate", "login", user.Login, "error", err)
		return ErrLoginAlreadyExists
	}
	if err != nil {
		logRepository("save user failed", "uuid", user.UUID, "login", user.Login, "error", err)
		return err
	}

	logRepository("save user ok", "uuid", user.UUID, "login", user.Login)
	return nil
}

func (r *PostgresUserRepository) GetUserByLogin(ctx context.Context, login string) (domain.User, error) {
	logRepository("get user by login", "login", login)
	var scannedUUID sql.NullString
	var scannedLogin sql.NullString
	var scannedPassword sql.NullString

	err := r.db.QueryRow(
		ctx,
		getUserByLoginQuery,
		login,
	).Scan(&scannedUUID, &scannedLogin, &scannedPassword)
	if errors.Is(err, pgx.ErrNoRows) {
		logRepository("get user by login not found", "login", login)
		return domain.User{}, ErrUserNotFound
	}
	if err != nil {
		logRepository("get user by login failed", "login", login, "error", err)
		return domain.User{}, err
	}

	user, err := buildDatasourceUser(scannedUUID, scannedLogin, scannedPassword)
	if err != nil {
		logRepository("get user by login invalid row", "login", login, "error", err)
		return domain.User{}, err
	}

	logRepository("get user by login ok", "login", user.Login, "uuid", user.UUID)
	return mapper.ToDomainUser(user), nil
}

func (r *PostgresUserRepository) GetUserByUUID(ctx context.Context, uuid googleuuid.UUID) (domain.User, error) {
	logRepository("get user by uuid", "uuid", uuid)
	var scannedUUID sql.NullString
	var login sql.NullString
	var password sql.NullString

	err := r.db.QueryRow(
		ctx,
		getUserByUUIDQuery,
		uuid.String(),
	).Scan(&scannedUUID, &login, &password)
	if errors.Is(err, pgx.ErrNoRows) {
		logRepository("get user by uuid not found", "uuid", uuid)
		return domain.User{}, ErrUserNotFound
	}
	if err != nil {
		logRepository("get user by uuid failed", "uuid", uuid, "error", err)
		return domain.User{}, err
	}

	user, err := buildDatasourceUser(scannedUUID, login, password)
	if err != nil {
		logRepository("get user by uuid invalid row", "uuid", uuid, "error", err)
		return domain.User{}, err
	}

	logRepository("get user by uuid ok", "uuid", uuid, "login", user.Login)
	return mapper.ToDomainUser(user), nil
}

func (r *PostgresUserRepository) UpdateUserPassword(ctx context.Context, uuid googleuuid.UUID, password string) error {
	logRepository("update user password", "uuid", uuid)
	result, err := r.db.Exec(
		ctx,
		updateUserPasswordQuery,
		uuid.String(),
		password,
	)
	if err != nil {
		logRepository("update user password failed", "uuid", uuid, "error", err)
		return err
	}
	if result.RowsAffected() == 0 {
		logRepository("update user password not found", "uuid", uuid)
		return ErrUserNotFound
	}
	logRepository("update user password ok", "uuid", uuid)
	return nil
}

func (r *PostgresUserRepository) DeleteUser(ctx context.Context, uuid googleuuid.UUID) error {
	logRepository("delete user", "uuid", uuid)
	result, err := r.db.Exec(ctx, deleteUserQuery, uuid.String())
	if err != nil {
		logRepository("delete user failed", "uuid", uuid, "error", err)
		return err
	}
	if result.RowsAffected() == 0 {
		logRepository("delete user not found", "uuid", uuid)
		return ErrUserNotFound
	}
	logRepository("delete user ok", "uuid", uuid)
	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func buildDatasourceUser(uuid sql.NullString, login sql.NullString, password sql.NullString) (datasource.User, error) {
	userUUID, err := requiredString("users.uuid", uuid)
	if err != nil {
		return datasource.User{}, err
	}
	userLogin, err := requiredString("users.login", login)
	if err != nil {
		return datasource.User{}, err
	}
	userPassword, err := requiredString("users.password", password)
	if err != nil {
		return datasource.User{}, err
	}
	return datasource.User{UUID: userUUID, Login: userLogin, Password: userPassword}, nil
}
