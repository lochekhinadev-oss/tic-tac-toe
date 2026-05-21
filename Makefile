-include .env

GOCACHE ?= /tmp/go-cache
BACKEND_DIR := backend/tic-tac-toe
POSTGRES_DB ?= tic_tac_toe
POSTGRES_USER ?= postgres
POSTGRES_PASSWORD ?= postgres
DATABASE_URL ?= postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@localhost:5432/$(POSTGRES_DB)?sslmode=disable
SEED_USERS ?= 10000
SEED_GAMES ?= 100000

.PHONY: test coverage docs docs-swag run backend-run seed-db repo-check compose-build compose-up compose-down

test: docs
	cd $(BACKEND_DIR) && GOCACHE=$(GOCACHE) go test ./...

coverage:
	cd $(BACKEND_DIR) && GOCACHE=$(GOCACHE) go test ./... -cover

docs:
	cd $(BACKEND_DIR) && GOCACHE=$(GOCACHE) go run ./cmd/docsgen

docs-swag: docs

run: docs
	docker compose up --build

backend-run:
	cd $(BACKEND_DIR) && GOCACHE=$(GOCACHE) go run ./cmd/app

seed-db:
	cd $(BACKEND_DIR) && GOCACHE=$(GOCACHE) DATABASE_URL="$(DATABASE_URL)" SEED_USERS=$(SEED_USERS) SEED_GAMES=$(SEED_GAMES) go run ./cmd/seed

repo-check:
	../scripts/check-forbidden-git-files.sh
	cd $(BACKEND_DIR) && GOCACHE=$(GOCACHE) go test ./...
	cd $(BACKEND_DIR) && GOCACHE=$(GOCACHE) go vet ./...
