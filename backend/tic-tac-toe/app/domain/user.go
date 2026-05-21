package domain

import "context"

type User struct {
	UUID     string `db:"uuid"`
	Login    string `db:"login"`
	Password string `db:"password"`
}

type UserRepository interface {
	SaveUser(ctx context.Context, user User) error
	GetUserByLogin(ctx context.Context, login string) (User, error)
	GetUserByUUID(ctx context.Context, uuid string) (User, error)
	UpdateUserPassword(ctx context.Context, uuid string, password string) error
}

type UserService interface {
	CreateUser(ctx context.Context, user User) error
	GetUserByLogin(ctx context.Context, login string) (User, error)
	GetUserByUUID(ctx context.Context, uuid string) (User, error)
	UpdatePassword(ctx context.Context, uuid string, password string) error
	VerifyPassword(user User, password string) (bool, bool)
}
