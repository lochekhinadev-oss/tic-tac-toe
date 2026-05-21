package mapper

import (
	"testing"

	"tic-tac-toe/app/domain"
	"tic-tac-toe/infrastructure/postgres/datasource"
)

func TestToDatasourceGameClonesField(t *testing.T) {
	game := domain.Game{
		UUID: "game-1",
		Field: domain.Field{
			{1, 0, 0},
			{0, 2, 0},
			{0, 0, 0},
		},
	}

	mapped := ToDatasourceGame(game)
	game.Field[0][0] = 9

	if mapped.UUID != "game-1" || mapped.Field[0][0] != 1 {
		t.Fatalf("unexpected mapped game: %#v", mapped)
	}
}

func TestToDomainGameClonesField(t *testing.T) {
	game := datasource.Game{
		UUID: "game-2",
		Field: datasource.Field{
			{2, 0, 0},
			{0, 1, 0},
			{0, 0, 0},
		},
	}

	mapped := ToDomainGame(game)
	game.Field[0][0] = 9

	if mapped.UUID != "game-2" || mapped.Field[0][0] != 2 {
		t.Fatalf("unexpected mapped game: %#v", mapped)
	}
}
