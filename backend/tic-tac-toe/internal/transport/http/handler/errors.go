package handler

import (
	"errors"

	"tic-tac-toe/internal/transport/http/messages"
)

var (
	errInvalidRequestBody = errors.New(messages.InvalidRequestBody)
	errInvalidUUID        = errors.New(messages.InvalidUUID)
)
