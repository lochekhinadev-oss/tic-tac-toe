package handler

import (
	"context"

	"tic-tac-toe/app/domain"
	"tic-tac-toe/infrastructure/auth"
)

type GameStorage interface {
	GameCommandStorage
	GameQueryStorage
}

type GameCommandService interface {
	CreateGame(uuid string, creatorUUID string, mode domain.GameMode) (domain.Game, error)
	JoinGame(game domain.Game, userUUID string) (domain.Game, error)
	ApplyMove(previous domain.Game, current domain.Game, userUUID string) (domain.Game, error)
}

type GameQueryService interface {
	GetGame(ctx context.Context, uuid string) (domain.Game, error)
	ListActiveGames(ctx context.Context) ([]domain.Game, error)
	ListCompletedGamesByUserUUID(ctx context.Context, userUUID string) ([]domain.Game, error)
	ListTopPlayers(ctx context.Context, limit int) ([]domain.WonGameInfo, error)
}

type GameCommandStorage interface {
	SaveGame(ctx context.Context, game domain.Game) error
	SaveGameIfUnchanged(ctx context.Context, previous domain.Game, next domain.Game) error
	JoinGame(ctx context.Context, uuid string, userUUID string) (domain.Game, error)
}

type GameQueryStorage interface {
	GetGame(ctx context.Context, uuid string) (domain.Game, error)
	ListActiveGames(ctx context.Context) ([]domain.Game, error)
	ListCompletedGamesByUserUUID(ctx context.Context, userUUID string) ([]domain.Game, error)
	ListTopPlayers(ctx context.Context, limit int) ([]domain.WonGameInfo, error)
}

type AuthService interface {
	SignUp(ctx context.Context, request auth.SignUpRequest) (bool, error)
	SignIn(ctx context.Context, request auth.SessionRequest) (auth.SessionResponse, error)
	RefreshSession(ctx context.Context, sessionID string) (auth.SessionResponse, error)
	Logout(ctx context.Context, sessionID string) error
	LogoutAll(ctx context.Context, sessionID string) error
	AuthenticateSession(ctx context.Context, sessionID string) (string, error)
}

type UserQueryService interface {
	GetUserByUUID(ctx context.Context, uuid string) (domain.User, error)
	DeleteUser(ctx context.Context, uuid string) error
}
