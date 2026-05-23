package domain

import (
	"context"

	googleuuid "github.com/google/uuid"
)

type User struct {
	UUID     string `db:"uuid"`
	Login    string `db:"login"`
	Password string `db:"password"`
}

type UserRepository interface {
	SaveUser(ctx context.Context, user User) error
	GetUserByLogin(ctx context.Context, login string) (User, error)
	GetUserByUUID(ctx context.Context, uuid googleuuid.UUID) (User, error)
	UpdateUserPassword(ctx context.Context, uuid googleuuid.UUID, password string) error
	DeleteUser(ctx context.Context, uuid googleuuid.UUID) error
}

type UserService interface {
	CreateUser(ctx context.Context, user User) error
	GetUserByLogin(ctx context.Context, login string) (User, error)
	GetUserByUUID(ctx context.Context, uuid googleuuid.UUID) (User, error)
	UpdatePassword(ctx context.Context, uuid googleuuid.UUID, password string) error
	DeleteUser(ctx context.Context, uuid googleuuid.UUID) error
	VerifyPassword(user User, password string) (bool, bool)
}
