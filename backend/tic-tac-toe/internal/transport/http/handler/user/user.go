package user

import (
	"tic-tac-toe/app/domain"
	root "tic-tac-toe/internal/transport/http/handler"
)

type Handler = root.UserHandler

func New(users domain.UserService) *Handler {
	return root.NewUserHandler(users)
}
