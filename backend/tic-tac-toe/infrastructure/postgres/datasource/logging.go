package datasource

import (
	"log"

	_ "tic-tac-toe/internal/logging"
)

const databaseLogPrefix = "[infrastructure/postgres/datasource]"

func logDatabase(format string, args ...any) {
	log.Printf(databaseLogPrefix+" "+format, args...)
}
