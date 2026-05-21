package handler

type GameHandler struct {
	logic   GameLogic
	storage GameStorage
}

func NewGameHandler(logic GameLogic, storage GameStorage) *GameHandler {
	return newGameHandler(logic, storage)
}

func newGameHandler(logic GameLogic, storage GameStorage) *GameHandler {
	return &GameHandler{
		logic:   logic,
		storage: storage,
	}
}
