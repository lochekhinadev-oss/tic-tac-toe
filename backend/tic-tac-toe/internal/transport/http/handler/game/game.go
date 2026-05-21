package game

import root "tic-tac-toe/internal/transport/http/handler"

type Handler = root.GameHandler

func New(logic root.GameLogic, storage root.GameStorage) *Handler {
	return root.NewGameHandler(logic, storage)
}
