package handler

import (
	"log"

	_ "tic-tac-toe/internal/logging"
)

const handlerLogPrefix = "[transport/http/handler]"

func logHandler(format string, args ...any) {
	log.Printf(handlerLogPrefix+" "+format, args...)
}
