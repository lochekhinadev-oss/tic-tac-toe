package auth

import (
	"tic-tac-toe/infrastructure/auth"
	root "tic-tac-toe/internal/transport/http/handler"
)

type Handler = root.AuthHandler

func New(authService auth.AuthService) *Handler {
	return root.NewAuthHandler(authService)
}
