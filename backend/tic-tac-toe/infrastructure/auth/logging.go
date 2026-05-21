package auth

import "tic-tac-toe/internal/logging"

const authLogPrefix = "[infrastructure/auth]"

func logAuth(format string, args ...any) {
	log.Printf(authLogPrefix+" "+format, args...)
}
