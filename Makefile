-include .env

GOCACHE ?= /tmp/go-cache
BACKEND_DIR := backend/tic-tac-toe
POSTGRES_DB ?= tic_tac_toe
POSTGRES_USER ?= postgres
POSTGRES_PASSWORD ?= postgres
DATABASE_URL ?= postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@localhost:5432/$(POSTGRES_DB)?sslmode=disable
REDIS_URL ?= redis://localhost:6379/0
SEED_USERS ?= 10000
SEED_GAMES ?= 100000

.PHONY: test coverage docs run seed-db cleanup-db certs

test: docs
	cd $(BACKEND_DIR) && GOCACHE=$(GOCACHE) go test ./...

coverage:
	cd $(BACKEND_DIR) && GOCACHE=$(GOCACHE) go test ./... -cover

docs:
	cd $(BACKEND_DIR) && GOCACHE=$(GOCACHE) swag init --generalInfo main.go --output docs --parseInternal --dir ./cmd/app,./app/application,./app/domain,./infrastructure/auth,./infrastructure/postgres/datasource,./infrastructure/postgres/mapper,./infrastructure/postgres/repository,./internal/di,./internal/transport/http/handler,./internal/transport/http/middleware,./internal/transport/http/response,./internal/transport/http/dto

run: docs
	cp .env.example .env
	docker compose up --build

seed-db:
	cp .env.example .env
	SEED_ENABLED=1 SEED_USERS=$(SEED_USERS) SEED_GAMES=$(SEED_GAMES) docker compose --profile seed run --rm -e SEED_ENABLED=1 -e SEED_USERS -e SEED_GAMES seed

cleanup-db:
	cp .env.example .env
	CLEANUP_ENABLED=1 docker compose run --rm cleanup

certs:
	@if [ -s certs/tic-tac-toe.crt ] && [ -s certs/tic-tac-toe.key ]; then \
		echo "HTTPS certificates already exist"; \
	else \
		./scripts/generate-https-certs.sh ./certs; \
	fi
