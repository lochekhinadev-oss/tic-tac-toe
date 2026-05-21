package auth

import "context"

type AuthService interface {
	SignUp(ctx context.Context, request SignUpRequest) (bool, error)
	Authenticate(ctx context.Context, request JwtRequest) (JwtResponse, error)
	RefreshAccessToken(ctx context.Context, refreshToken string) (JwtResponse, error)
	RefreshRefreshToken(ctx context.Context, refreshToken string) (JwtResponse, error)
	Logout(ctx context.Context, refreshToken string) error
	LogoutAll(ctx context.Context, refreshToken string) error
	AuthenticateToken(ctx context.Context, header string) (string, error)
}

type SignUpRequest struct {
	Login    string
	Password string
}

type JwtRequest struct {
	Login    string
	Password string
}

type JwtResponse struct {
	Type         string
	AccessToken  string
	RefreshToken string
}

type RefreshJwtRequest struct {
	RefreshToken string
}
