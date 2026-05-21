package domain

import "errors"

var (
	ErrGameNotFound    = errors.New("game not found")
	ErrGameConflict    = errors.New("game was changed by another request")
	ErrGameNotJoinable = errors.New("game is not joinable")
	ErrUserNotFound    = errors.New("user not found")
)
