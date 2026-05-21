package repository

import (
	"log"

	_ "tic-tac-toe/internal/logging"
)

const repositoryLogPrefix = "[infrastructure/postgres/repository]"

func logRepository(format string, args ...any) {
	log.Printf(repositoryLogPrefix+" "+format, args...)
}
