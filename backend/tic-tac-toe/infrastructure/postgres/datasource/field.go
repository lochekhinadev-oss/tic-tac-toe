package datasource

import "time"

type Field [][]int

type Game struct {
	UUID           string    `db:"uuid"`
	Field          Field     `db:"field"`
	Mode           string    `db:"mode"`
	State          string    `db:"state"`
	CreatedAt      time.Time `db:"created_at"`
	NextPlayerUUID string    `db:"next_player_uuid"`
	WinnerUUID     string    `db:"winner_uuid"`
	PlayerXUUID    string    `db:"player_x_uuid"`
	PlayerOUUID    string    `db:"player_o_uuid"`
}
type User struct {
	UUID     string `db:"uuid"`
	Login    string `db:"login"`
	Password string `db:"password"`
}
