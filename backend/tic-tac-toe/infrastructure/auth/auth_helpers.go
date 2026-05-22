package auth

import (
	"log"

	_ "tic-tac-toe/internal/logging"
)

const authLogPrefix = "[infrastructure/auth]"
const SessionCookieName = "tic-tac-toe.session"

func logAuth(format string, args ...any) {
	log.Printf(authLogPrefix+" "+format, args...)
}
