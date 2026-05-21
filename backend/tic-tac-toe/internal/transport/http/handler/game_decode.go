package handler

import (
	"net/http"

	"tic-tac-toe/app/domain"
	"tic-tac-toe/internal/transport/http/dto"
)

func (h *GameHandler) decodeRequest(r *http.Request) (dto.GameRequest, error) {
	return decodeJSONBody[dto.GameRequest](r)
}

func (h *GameHandler) decodeCreateRequest(r *http.Request) (dto.CreateGameRequest, error) {
	return decodeOptionalJSONBody(r, dto.CreateGameRequest{Mode: string(domain.GameModeComputer)})
}
