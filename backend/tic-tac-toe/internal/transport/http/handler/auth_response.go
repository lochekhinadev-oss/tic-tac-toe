package handler

import (
	"tic-tac-toe/infrastructure/auth"
	"tic-tac-toe/internal/transport/http/dto"
)

func jwtResponseToDTO(response auth.JwtResponse) dto.JwtResponse {
	return dto.JwtResponse{
		Type:         response.Type,
		AccessToken:  response.AccessToken,
		RefreshToken: response.RefreshToken,
	}
}
