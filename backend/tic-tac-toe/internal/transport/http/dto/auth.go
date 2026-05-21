package dto

import "github.com/google/uuid"

type SignUpRequest struct {
	// User login. Must not be empty.
	Login string `json:"login" example:"player"`
	// User password. Must not be empty.
	Password string `json:"password" example:"secret"`
}

type SignUpResponse struct {
	// Registration result.
	Success bool `json:"success" example:"true"`
}

type JwtRequest struct {
	// User login. Must not be empty.
	Login string `json:"login" example:"player"`
	// User password. Must not be empty.
	Password string `json:"password" example:"secret"`
}

type JwtResponse struct {
	// Token type.
	Type string `json:"type" example:"Bearer"`
	// Short-lived access token.
	AccessToken string `json:"accessToken" example:"eyJhbGciOiJSUzI1NiIsImtpZCI6InRpYy10YWMtdG9lLW1haW4iLCJ0eXAiOiJKV1QifQ..."`
	// Long-lived refresh token.
	RefreshToken string `json:"refreshToken" example:"eyJhbGciOiJSUzI1NiIsImtpZCI6InRpYy10YWMtdG9lLW1haW4iLCJ0eXAiOiJKV1QifQ..."`
}

type RefreshJwtRequest struct {
	// Refresh token.
	RefreshToken string `json:"refreshToken" example:"eyJhbGciOiJSUzI1NiIsImtpZCI6InRpYy10YWMtdG9lLW1haW4iLCJ0eXAiOiJKV1QifQ..."`
}

type AuthResponse struct {
	// Authenticated user UUID.
	UUID uuid.UUID `json:"uuid" format:"uuid" example:"123e4567-e89b-42d3-a456-426614174000"`
}

type UserResponse struct {
	// User UUID.
	UUID uuid.UUID `json:"uuid" format:"uuid" example:"123e4567-e89b-42d3-a456-426614174000"`
	// User login.
	Login string `json:"login" example:"player"`
}
