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

type AuthRequest struct {
	// User login. Must not be empty.
	Login string `json:"login" example:"player"`
	// User password. Must not be empty.
	Password string `json:"password" example:"secret"`
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
