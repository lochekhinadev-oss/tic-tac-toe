GOCACHE ?= /tmp/go-cache
BACKEND_DIR := backend/tic-tac-toe

.PHONY: test coverage docs run full-up seed-db cleanup-db certs observability-up observability-down

test: docs
	cd $(BACKEND_DIR) && GOCACHE=$(GOCACHE) go test ./...

coverage:
	cd $(BACKEND_DIR) && GOCACHE=$(GOCACHE) go test ./... -cover

docs:
	cd $(BACKEND_DIR) && GOCACHE=$(GOCACHE) swag init --generalInfo main.go --output docs --parseInternal --dir ./cmd/app,./app/application,./app/domain,./infrastructure/auth,./infrastructure/postgres/datasource,./infrastructure/postgres/mapper,./infrastructure/postgres/repository,./internal/di,./internal/transport/http/handler,./internal/transport/http/middleware,./internal/transport/http/response,./internal/transport/http/dto

run: docs
	docker compose up --build --force-recreate --remove-orphans

full-up: docs
	docker compose -f docker-compose.yml -f docker-compose.observability.yml up --build --force-recreate --remove-orphans

observability-up: docs
	docker compose -f docker-compose.yml -f docker-compose.observability.yml up -d --force-recreate prometheus grafana alertmanager postgres-exporter redis-exporter

observability-down:
	docker compose -f docker-compose.yml -f docker-compose.observability.yml stop prometheus grafana alertmanager postgres-exporter redis-exporter

seed-db:
	docker compose --profile seed run --rm seed

cleanup-db:
	docker compose run --rm cleanup

certs:
	@if [ -s certs/tic-tac-toe.crt ] && [ -s certs/tic-tac-toe.key ]; then \
		if [ "$$(uname -s)" = "Darwin" ] && command -v security >/dev/null 2>&1; then \
			if security verify-cert -c certs/tic-tac-toe.crt -p ssl -n localhost >/dev/null 2>&1; then \
				echo "HTTPS certificates already exist"; \
			else \
				./scripts/generate-https-certs.sh ./certs; \
			fi; \
		else \
			echo "HTTPS certificates already exist"; \
		fi; \
	else \
		./scripts/generate-https-certs.sh ./certs; \
	fi
