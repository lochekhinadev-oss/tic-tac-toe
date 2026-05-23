package auth

import (
	"context"
	"time"
)

type AuthService interface {
	SignUp(ctx context.Context, request SignUpRequest) (bool, error)
	SignIn(ctx context.Context, request SessionRequest) (SessionResponse, error)
	Logout(ctx context.Context, sessionID string) error
	LogoutAll(ctx context.Context, sessionID string) error
	AuthenticateSession(ctx context.Context, sessionID string) (string, error)
}

type SignUpRequest struct {
	Login    string
	Password string
}

type SessionRequest struct {
	Login    string
	Password string
}

type SessionResponse struct {
	UserUUID  string
	SessionID string
	ExpiresAt time.Time
}
