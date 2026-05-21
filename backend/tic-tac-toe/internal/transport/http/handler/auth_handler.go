package handler

import "tic-tac-toe/infrastructure/auth"

type AuthHandler struct {
	auth AuthService
}

func NewAuthHandler(auth auth.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}
