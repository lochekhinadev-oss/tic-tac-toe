package application

import (
	"log"

	_ "tic-tac-toe/internal/logging"
)

const applicationLogPrefix = "[app/application]"

func logApplication(format string, args ...any) {
	log.Printf(applicationLogPrefix+" "+format, args...)
}
